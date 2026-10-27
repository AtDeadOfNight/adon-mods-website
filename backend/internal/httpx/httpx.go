package httpx

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"runtime"
	"strconv"
	"time"
)

type problem struct {
	OK      bool   `json:"ok"`
	Error   string `json:"error"`
	ErrorRu string `json:"errorRu,omitempty"`
}

// Not http.Redirect: it adds a Content-Type and an HTML body.
func Redirect(w http.ResponseWriter, location string) {
	header := w.Header()
	header.Set("Location", location)
	header.Set("Content-Length", "0")
	w.WriteHeader(http.StatusFound)
}

func Error(w http.ResponseWriter, status int, message string) {
	writeProblem(w, status, problem{Error: message})
}

func LocalizedError(w http.ResponseWriter, status int, message, messageRu string) {
	writeProblem(w, status, problem{Error: message, ErrorRu: messageRu})
}

func ServerError(w http.ResponseWriter) {
	Error(w, http.StatusInternalServerError, "Internal server error")
}

// writeProblem writes the same bytes as JSON.stringify: no HTML escaping, no trailing newline.
func writeProblem(w http.ResponseWriter, status int, answer problem) {
	var body bytes.Buffer
	encoder := json.NewEncoder(&body)
	encoder.SetEscapeHTML(false)
	encoder.Encode(answer)
	WriteBody(w, status, "application/json", bytes.TrimSuffix(body.Bytes(), []byte("\n")))
}

func WriteBody(w http.ResponseWriter, status int, contentType string, body []byte) {
	header := w.Header()
	header.Set("Content-Type", contentType)
	header.Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(status)
	w.Write(body)
}

func Recover(w http.ResponseWriter) {
	recovered := recover()
	if recovered == nil {
		return
	}
	if recovered == http.ErrAbortHandler {
		panic(recovered)
	}
	if runtimeErr, ok := recovered.(runtime.Error); ok {
		log.Printf("handler panic: %v", runtimeErr)
	} else {
		log.Print("handler panic")
	}
	ServerError(w)
}

// net/http drains an unread body before and after the answer under the connection's read deadline,
// which is otherwise unset, so a body that never arrives would hold the connection for good.
func GuardBody(w http.ResponseWriter, timeout time.Duration) {
	w.Header().Set("Connection", "close")
	http.NewResponseController(w).SetReadDeadline(time.Now().Add(timeout))
}

func DropUnreadBody(w http.ResponseWriter) {
	http.NewResponseController(w).SetReadDeadline(time.Now())
}
