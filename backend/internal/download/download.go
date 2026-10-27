package download

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"git.hloth.dev/at-dead-of-night/mods.adon.hloth.dev/backend/internal/httpx"
)

const (
	StallTimeout = time.Minute
	chunkSize    = 128 << 10
)

// The site links to the keys; the files in DOWNLOADS_DIR keep their own names.
var files = map[string]string{
	"AtDeadOfNight.app":                               "At Dead Of Night.zip",
	"atdeadofdeeznuts.7z":                             "atdeadofdeeznuts.7z",
	"at-dead-of-night-android-v1.0.0-t.me_adonru.apk": "at-dead-of-night-android-v1.0.0-t.me_adonru.apk",
	"at-dead-of-night-ios-v1.0.0-t.me_adonru.ipa":     "at-dead-of-night-ios-v1.0.0-t.me_adonru.ipa",
}

func FileName(key string) (string, bool) {
	name, ok := files[key]
	return name, ok
}

type rangeKind int

const (
	wholeFile rangeKind = iota
	partialFile
	unsatisfiable
)

type byteRange struct {
	kind  rangeKind
	start int64
	end   int64
}

var singleRangePattern = regexp.MustCompile(`^bytes=(\d*)-(\d*)$`)

// parseRange honors one range, so a browser can resume; anything else it does not understand
// gets the whole file.
func parseRange(header string, size int64) byteRange {
	match := singleRangePattern.FindStringSubmatch(header)
	if match == nil || match[1] == "" && match[2] == "" {
		return byteRange{kind: wholeFile}
	}
	if match[1] == "" {
		suffix := digitsValue(match[2])
		if suffix == 0 {
			return byteRange{kind: unsatisfiable}
		}
		// A 206 must carry a Content-Range, and none describes zero bytes (RFC 9110 §15.3.7, §14.4).
		if size == 0 {
			return byteRange{kind: wholeFile}
		}
		return byteRange{kind: partialFile, start: int64(max(0, float64(size)-suffix)), end: size - 1}
	}
	start := digitsValue(match[1])
	end := float64(size - 1)
	if match[2] != "" {
		end = min(digitsValue(match[2]), end)
	}
	if start >= float64(size) {
		return byteRange{kind: unsatisfiable}
	}
	if end < start {
		return byteRange{kind: wholeFile}
	}
	return byteRange{kind: partialFile, start: int64(start), end: int64(end)}
}

// A bound too long for int64 rounds, which keeps it comparable with the size instead of failing.
func digitsValue(digits string) float64 {
	value, _ := strconv.ParseFloat(digits, 64)
	return value
}

// Serve fails only when the file cannot be opened, and then before writing anything.
func Serve(w http.ResponseWriter, r *http.Request, dir, name string, stallTimeout time.Duration) error {
	file, size, err := open(filepath.Join(dir, name))
	if err != nil {
		return err
	}
	defer file.Close()

	header := w.Header()
	// Bun repeats the Range header values joined with ", ", which no single range matches.
	requested := parseRange(strings.Join(r.Header.Values("Range"), ", "), size)
	if requested.kind == unsatisfiable {
		header.Set("Content-Range", "bytes */"+strconv.FormatInt(size, 10))
		httpx.Error(w, http.StatusRequestedRangeNotSatisfiable, "Requested Range Not Satisfiable")
		return nil
	}

	header.Set("Accept-Ranges", "bytes")
	header.Set("Content-Type", "application/octet-stream")
	header.Set("Content-Disposition", `attachment; filename="`+name+`"`)
	status, start, length := http.StatusOK, int64(0), size
	if requested.kind == partialFile {
		status, start, length = http.StatusPartialContent, requested.start, requested.end-requested.start+1
	}
	// Bun added Content-Range to a whole-file 200 as well.
	if length > 0 {
		header.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, start+length-1, size))
	}
	header.Set("Content-Length", strconv.FormatInt(length, 10))
	w.WriteHeader(status)
	if r.Method != http.MethodHead {
		if _, err := file.Seek(start, io.SeekStart); err == nil {
			send(w, file, length, stallTimeout)
		}
	}
	return nil
}

// The server has no WriteTimeout, which a large download would outlast, so each chunk gets its own
// deadline and a client that stopped reading is dropped.
func send(w http.ResponseWriter, file io.Reader, length int64, stallTimeout time.Duration) {
	controller := http.NewResponseController(w)
	for remaining := length; remaining > 0; {
		controller.SetWriteDeadline(time.Now().Add(stallTimeout))
		written, err := io.CopyN(w, file, min(remaining, chunkSize))
		if err != nil {
			return
		}
		remaining -= written
	}
}

func open(path string) (*os.File, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, errors.New("the file cannot be opened")
	}
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		file.Close()
		return nil, 0, errors.New("the file is not a regular file")
	}
	return file, info.Size(), nil
}
