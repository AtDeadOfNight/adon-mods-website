package httpx

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestAnswers(t *testing.T) {
	cases := []struct {
		name   string
		write  func(http.ResponseWriter)
		status int
		header map[string]string
		body   string
	}{
		{"redirect", func(w http.ResponseWriter) { Redirect(w, "https://x/") }, http.StatusFound, map[string]string{"Location": "https://x/", "Content-Length": "0", "Content-Type": ""}, ""},
		{"error", func(w http.ResponseWriter) { Error(w, http.StatusNotFound, "Not <found> & \"gone\"") }, http.StatusNotFound,
			map[string]string{"Content-Type": "application/json", "Content-Length": "45"}, `{"ok":false,"error":"Not <found> & \"gone\""}`},
		{"server error", ServerError, http.StatusInternalServerError, map[string]string{"Content-Type": "application/json", "Content-Length": "44"}, `{"ok":false,"error":"Internal server error"}`},
		{"localized", func(w http.ResponseWriter) {
			LocalizedError(w, http.StatusBadGateway, "Steam", "Стим")
		}, http.StatusBadGateway,
			map[string]string{"Content-Length": "49"}, `{"ok":false,"error":"Steam","errorRu":"Стим"}`},
	}
	for _, c := range cases {
		recorder := httptest.NewRecorder()
		c.write(recorder)
		if recorder.Code != c.status || recorder.Body.String() != c.body {
			t.Errorf("%s: %d %q, want %d %q", c.name, recorder.Code, recorder.Body.String(), c.status, c.body)
		}
		for name, want := range c.header {
			if got := recorder.Header().Get(name); got != want {
				t.Errorf("%s: %s %q, want %q", c.name, name, got, want)
			}
		}
	}
}

func TestRecover(t *testing.T) {
	var logged bytes.Buffer
	log.SetOutput(&logged)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	cases := map[string]struct {
		handler func()
		logged  string
	}{
		"runtime error": {func() { var values []int; _ = values[1] }, "handler panic: runtime error: index out of range [1] with length 0"},
		"other value":   {func() { panic("secret") }, "handler panic"},
	}
	for name, c := range cases {
		logged.Reset()
		recorder := httptest.NewRecorder()
		func() {
			defer Recover(recorder)
			c.handler()
		}()
		if recorder.Code != http.StatusInternalServerError || recorder.Body.String() != `{"ok":false,"error":"Internal server error"}` {
			t.Errorf("%s: %d %q", name, recorder.Code, recorder.Body.String())
		}
		if !strings.HasSuffix(logged.String(), c.logged+"\n") {
			t.Errorf("%s: logged %q", name, logged.String())
		}
	}
	func() {
		defer func() {
			if recovered := recover(); recovered != http.ErrAbortHandler {
				t.Errorf("ErrAbortHandler became %v", recovered)
			}
		}()
		defer Recover(httptest.NewRecorder())
		panic(http.ErrAbortHandler)
	}()
	recorder := httptest.NewRecorder()
	func() { defer Recover(recorder) }()
	if recorder.Code != http.StatusOK || recorder.Body.Len() != 0 {
		t.Errorf("no panic: %d %q", recorder.Code, recorder.Body.String())
	}
}
