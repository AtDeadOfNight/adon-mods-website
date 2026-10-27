package download

import (
	"bytes"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestParseRange(t *testing.T) {
	const size = 70001
	partial := func(start, end int64) byteRange { return byteRange{kind: partialFile, start: start, end: end} }
	whole := byteRange{kind: wholeFile}
	outside := byteRange{kind: unsatisfiable}
	cases := map[string]byteRange{
		"":                             whole,
		"bytes=1000-":                  partial(1000, 70000),
		"bytes=10-19":                  partial(10, 19),
		"bytes=0-0":                    partial(0, 0),
		"bytes=70000-":                 partial(70000, 70000),
		"bytes=-100":                   partial(69901, 70000),
		"bytes=-70001":                 partial(0, 70000),
		"bytes=-999999999":             partial(0, 70000),
		"bytes=-0":                     outside,
		"bytes=70001-":                 outside,
		"bytes=70001-70011":            outside,
		"bytes=69996-70101":            partial(69996, 70000),
		"bytes=0-70001":                partial(0, 70000),
		"bytes=0-70000":                partial(0, 70000),
		"bytes=0-1,5-6":                whole,
		"bytes=0-1, bytes=5-6":         whole,
		"bytes=9-3":                    whole,
		"bytes=-":                      whole,
		"lines=1-2":                    whole,
		"Bytes=0-1":                    whole,
		"bytes= 0-1":                   whole,
		"bytes=0010-0019":              partial(10, 19),
		"bytes=0-99999999999999999999": partial(0, 70000),
		"bytes=99999999999999999999-":  outside,
		"bytes=-99999999999999999999":  partial(0, 70000),
		"bytes=--5":                    whole,
	}
	for header, want := range cases {
		if got := parseRange(header, size); got != want {
			t.Errorf("parseRange(%q) = %+v, want %+v", header, got, want)
		}
	}
}

const testName = "At Dead Of Night.zip"

func fileDir(t *testing.T, content []byte) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, testName), content, 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func serve(t *testing.T, dir, method string, headers ...string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, "/download", nil)
	for i := 0; i+1 < len(headers); i += 2 {
		request.Header.Add(headers[i], headers[i+1])
	}
	recorder := httptest.NewRecorder()
	if err := Serve(recorder, request, dir, testName, time.Minute); err != nil {
		t.Fatal(err)
	}
	return recorder
}

func expectHeaders(t *testing.T, header http.Header, want map[string]string) {
	t.Helper()
	for name, value := range want {
		if got := header.Get(name); got != value {
			t.Errorf("%s: %q, want %q", name, got, value)
		}
	}
}

func TestServe(t *testing.T) {
	content := bytes.Repeat([]byte("0123456789"), 10)
	dir := fileDir(t, content)

	full := serve(t, dir, http.MethodGet)
	if full.Code != http.StatusOK || !bytes.Equal(full.Body.Bytes(), content) {
		t.Fatalf("full download: %d, %d bytes", full.Code, full.Body.Len())
	}
	expectHeaders(t, full.Header(), map[string]string{
		"Accept-Ranges":       "bytes",
		"Content-Type":        "application/octet-stream",
		"Content-Disposition": `attachment; filename="At Dead Of Night.zip"`,
		"Content-Length":      "100",
		"Content-Range":       "bytes 0-99/100",
		"Last-Modified":       "",
		"ETag":                "",
	})

	part := serve(t, dir, http.MethodGet, "Range", "bytes=10-19", "If-Range", `"stale"`)
	if part.Code != http.StatusPartialContent || part.Body.String() != "0123456789" {
		t.Fatalf("range: %d %q", part.Code, part.Body.String())
	}
	expectHeaders(t, part.Header(), map[string]string{"Content-Range": "bytes 10-19/100", "Content-Length": "10"})

	suffix := serve(t, dir, http.MethodGet, "Range", "bytes=-3")
	if suffix.Code != http.StatusPartialContent || suffix.Body.String() != "789" {
		t.Fatalf("suffix: %d %q", suffix.Code, suffix.Body.String())
	}

	repeated := serve(t, dir, http.MethodGet, "Range", "bytes=10-19", "Range", "bytes=20-29")
	if repeated.Code != http.StatusOK || repeated.Body.Len() != 100 {
		t.Fatalf("repeated Range headers: %d, %d bytes", repeated.Code, repeated.Body.Len())
	}

	outside := serve(t, dir, http.MethodGet, "Range", "bytes=100-")
	if outside.Code != http.StatusRequestedRangeNotSatisfiable || outside.Body.String() != `{"ok":false,"error":"Requested Range Not Satisfiable"}` {
		t.Fatalf("unsatisfiable: %d %q", outside.Code, outside.Body.String())
	}
	expectHeaders(t, outside.Header(), map[string]string{"Content-Range": "bytes */100", "Content-Length": "54", "Content-Type": "application/json", "Accept-Ranges": ""})

	head := serve(t, dir, http.MethodHead, "Range", "bytes=10-19")
	expectHeaders(t, head.Header(), map[string]string{"Content-Length": "10", "Content-Range": "bytes 10-19/100", "Content-Disposition": `attachment; filename="At Dead Of Night.zip"`})
	if head.Code != http.StatusPartialContent || head.Body.Len() != 0 {
		t.Fatalf("HEAD: %d, %d bytes", head.Code, head.Body.Len())
	}

	conditional := serve(t, dir, http.MethodGet, "If-None-Match", "*", "If-Modified-Since", "Fri, 01 Jan 2100 00:00:00 GMT")
	if conditional.Code != http.StatusOK {
		t.Fatalf("conditional headers must be ignored, got %d", conditional.Code)
	}
}

func TestServeEmptyFile(t *testing.T) {
	dir := fileDir(t, nil)
	whole := serve(t, dir, http.MethodGet)
	if whole.Code != http.StatusOK || whole.Body.Len() != 0 {
		t.Fatalf("no Range: %d, %d bytes", whole.Code, whole.Body.Len())
	}
	expectHeaders(t, whole.Header(), map[string]string{"Content-Length": "0", "Content-Range": "", "Accept-Ranges": "bytes"})

	for _, suffix := range []string{"bytes=-5", "bytes=-1"} {
		got := serve(t, dir, http.MethodGet, "Range", suffix)
		if got.Code != http.StatusOK || got.Body.Len() != 0 || !maps.EqualFunc(got.Header(), whole.Header(), slices.Equal) {
			t.Errorf("%s: %d %v, want the answer without Range %v", suffix, got.Code, got.Header(), whole.Header())
		}
	}

	for _, rangeHeader := range []string{"bytes=0-", "bytes=0-0", "bytes=-0"} {
		got := serve(t, dir, http.MethodGet, "Range", rangeHeader)
		if got.Code != http.StatusRequestedRangeNotSatisfiable {
			t.Errorf("%s: %d, want 416", rangeHeader, got.Code)
		}
		expectHeaders(t, got.Header(), map[string]string{"Content-Range": "bytes */0", "Accept-Ranges": ""})
	}
}

func TestServeFailsBeforeWritingWithoutTheFile(t *testing.T) {
	notRegular := t.TempDir()
	if err := os.Mkdir(filepath.Join(notRegular, testName), 0o755); err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{
		"the file cannot be opened":      t.TempDir(),
		"the file is not a regular file": notRegular,
	}
	for want, dir := range cases {
		recorder := httptest.NewRecorder()
		err := Serve(recorder, httptest.NewRequest(http.MethodGet, "/download", nil), dir, testName, time.Minute)
		if err == nil || err.Error() != want {
			t.Errorf("error %v, want %q", err, want)
		}
		if len(recorder.Header()) != 0 || recorder.Body.Len() != 0 {
			t.Errorf("%s: wrote %v %q", want, recorder.Header(), recorder.Body.String())
		}
	}
}

func TestFileNames(t *testing.T) {
	for key, want := range map[string]string{
		"AtDeadOfNight.app":   "At Dead Of Night.zip",
		"atdeadofdeeznuts.7z": "atdeadofdeeznuts.7z",
	} {
		if name, ok := FileName(key); !ok || name != want {
			t.Errorf("FileName(%q) = %q, %v", key, name, ok)
		}
	}
	for _, key := range []string{"", "At Dead Of Night.zip", "__proto__", "constructor", "atdeadofnight.app"} {
		if _, ok := FileName(key); ok {
			t.Errorf("FileName(%q) was found", key)
		}
	}
}
