package server

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"git.hloth.dev/at-dead-of-night/mods.adon.hloth.dev/backend/internal/steam"
	"git.hloth.dev/at-dead-of-night/mods.adon.hloth.dev/backend/internal/upstream"
	"git.hloth.dev/at-dead-of-night/mods.adon.hloth.dev/backend/internal/upstream/upstreamtest"
)

type rawAnswer struct {
	status string
	header http.Header
	body   string
	closes bool
}

// Test servers stay on 127.0.0.1:39560-39599, out of the parity harness's ports.
func localListener(t *testing.T) net.Listener {
	t.Helper()
	for port := 39560; port <= 39599; port++ {
		listener, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
		if err == nil {
			t.Cleanup(func() { listener.Close() })
			return listener
		}
	}
	t.Fatal("no free port in 127.0.0.1:39560-39599")
	return nil
}

func startServer(t *testing.T, config *http.Server, listener net.Listener) *httptest.Server {
	t.Helper()
	server := &httptest.Server{Listener: listener, Config: config}
	server.Start()
	t.Cleanup(server.Close)
	return server
}

// servedApp runs the app behind a real listener, with its upstream calls going to upstreamHandler through the production client.
func servedApp(t *testing.T, upstreamHandler http.Handler) (*App, *httptest.Server) {
	t.Helper()
	cfg := testConfig(t)
	if upstreamHandler != nil {
		upstreamServer := startServer(t, &http.Server{Handler: upstreamHandler}, localListener(t))
		cfg.Origins.SteamCommunity, cfg.Origins.SteamAPI = upstreamServer.URL, upstreamServer.URL
	}
	a := New(cfg, upstream.NewClient(2*time.Second, 4*time.Second), fixedClock)
	return a, startServer(t, newHTTPServer(a), localListener(t))
}

func dial(t *testing.T, server *httptest.Server) (net.Conn, *bufio.Reader) {
	t.Helper()
	conn, err := net.Dial("tcp", server.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn, bufio.NewReader(conn)
}

func readAnswer(t *testing.T, conn net.Conn, reader *bufio.Reader, within time.Duration) rawAnswer {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(within))
	response, err := http.ReadResponse(reader, nil)
	if err != nil {
		t.Fatalf("no answer within %s: %v", within, err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("answer body: %v", err)
	}
	return rawAnswer{status: response.Status, header: response.Header, body: string(body), closes: response.Close}
}

func isTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func expectClosed(t *testing.T, conn net.Conn, reader *bufio.Reader, within time.Duration) {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(within))
	_, err := reader.ReadByte()
	switch {
	case isTimeout(err):
		t.Fatalf("the connection is still open after %s", within)
	case err == nil:
		t.Fatal("unexpected bytes after the answer")
	}
}

func TestUnreadBodyClosesTheConnectionAfterTheAnswer(t *testing.T) {
	_, server := servedApp(t, nil)
	requests := map[string]string{
		"unknown route":        "GET /nope HTTP/1.1\r\nHost: t\r\nContent-Length: 100\r\n\r\n",
		"login":                "GET /auth/steam HTTP/1.1\r\nHost: t\r\nContent-Length: 100\r\n\r\n",
		"POST /download":       "POST /download HTTP/1.1\r\nHost: t\r\nContent-Length: 100\r\n\r\n",
		"chunked, no chunk":    "GET /nope HTTP/1.1\r\nHost: t\r\nTransfer-Encoding: chunked\r\n\r\n",
		"download, no cookie":  "GET /download?file=x HTTP/1.1\r\nHost: t\r\nContent-Length: 100\r\n\r\n",
		"body sent, unwanted":  "GET /auth/steam HTTP/1.1\r\nHost: t\r\nContent-Length: 5\r\n\r\nhello",
		"expects 100-continue": "POST /download HTTP/1.1\r\nHost: t\r\nExpect: 100-continue\r\nContent-Length: 100\r\n\r\n",
	}
	for name, request := range requests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			conn, reader := dial(t, server)
			conn.Write([]byte(request))
			answer := readAnswer(t, conn, reader, 2*time.Second)
			if !answer.closes {
				t.Errorf("%s without Connection: close", answer.status)
			}
			expectClosed(t, conn, reader, 2*time.Second)
		})
	}
}

func TestRequestsWithoutBodyKeepTheConnection(t *testing.T) {
	_, server := servedApp(t, nil)
	conn, reader := dial(t, server)
	for range 2 {
		conn.Write([]byte("GET /download?file=x HTTP/1.1\r\nHost: t\r\n\r\n"))
		if answer := readAnswer(t, conn, reader, 2*time.Second); answer.closes || answer.header.Get("Location") != testPublicURL+"/purchase-to-download/x" {
			t.Fatalf("got %s %v", answer.status, answer.header)
		}
	}
}

func TestClientDisconnectCancelsTheUpstreamCall(t *testing.T) {
	cases := map[string]struct {
		path    string
		request string
	}{
		"check_authentication": {"/openid/login", "GET " + steamCallbackTarget(ownerID, zipKey) + " HTTP/1.1\r\nHost: t\r\n\r\n"},
		"GetOwnedGames":        {"/IPlayerService/GetOwnedGames/v1/", "GET /download?file=" + zipKey + " HTTP/1.1\r\nHost: t\r\nCookie: " + tokenCookie(validToken(ownerID)) + "\r\n\r\n"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			arrived := make(chan struct{}, 1)
			canceled := make(chan struct{}, 1)
			_, server := servedApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// net/http notices a closed connection only once the request body has been read.
				io.ReadAll(r.Body)
				arrived <- struct{}{}
				select {
				case <-r.Context().Done():
					canceled <- struct{}{}
				case <-time.After(5 * time.Second):
				}
			}))
			conn, _ := dial(t, server)
			conn.Write([]byte(c.request))
			select {
			case <-arrived:
			case <-time.After(2 * time.Second):
				t.Fatal("the upstream call never arrived")
			}
			conn.Close()
			select {
			case <-canceled:
			case <-time.After(2 * time.Second):
				t.Fatal("the upstream call went on after the client left")
			}
		})
	}
}

func clientWithoutRedirects() *http.Client {
	return &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

func doWithoutRedirects(t *testing.T, target string, cookie string) *http.Response {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		t.Fatal(err)
	}
	if cookie != "" {
		request.Header.Set("Cookie", cookie)
	}
	response, err := clientWithoutRedirects().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, response.Body)
	response.Body.Close()
	return response
}

func TestOversizedUpstreamAnswers(t *testing.T) {
	const upstreamBodyLimit = 1 << 20
	cases := []struct {
		name     string
		hugePath string
		target   string
		cookie   string
		status   int
	}{
		{"check_authentication", "/openid/login", steamCallbackTarget(ownerID, zipKey), "", http.StatusBadGateway},
		{"GetOwnedGames in the callback", "/IPlayerService/GetOwnedGames/v1/", steamCallbackTarget(ownerID, zipKey), "", http.StatusBadGateway},
		{"GetOwnedGames for a download", "/IPlayerService/GetOwnedGames/v1/", "/download?file=" + zipKey, tokenCookie(validToken(ownerID)), http.StatusBadRequest},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, server := servedApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				padding := ""
				if r.URL.Path == c.hugePath {
					padding = strings.Repeat(" ", upstreamBodyLimit)
				}
				if r.URL.Path == "/openid/login" {
					io.WriteString(w, openIDValidAnswer+padding)
					return
				}
				io.WriteString(w, ownsTheGame+padding)
			}))
			if response := doWithoutRedirects(t, server.URL+c.target, c.cookie); response.StatusCode != c.status {
				t.Fatalf("got %d %q", response.StatusCode, response.Header.Get("Location"))
			}
		})
	}
}

type receivedRequest struct {
	server   string
	path     string
	referer  string
	keyFound bool
}

func TestRedirectedUpstreamCallKeepsTheKeyHome(t *testing.T) {
	const key = "test-steam-key"
	var mu sync.Mutex
	var received []receivedRequest
	record := func(server string, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		keyFound := strings.Contains(r.URL.String(), key) || strings.Contains(string(body), key)
		for _, values := range r.Header {
			keyFound = keyFound || strings.Contains(strings.Join(values, "\n"), key)
		}
		mu.Lock()
		received = append(received, receivedRequest{server, r.URL.Path, r.Header.Get("Referer"), keyFound})
		mu.Unlock()
	}
	landing := startServer(t, &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		record("landing", r)
		io.WriteString(w, ownsTheGame)
	})}, localListener(t))
	_, server := servedApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		record("upstream", r)
		switch r.URL.Path {
		case "/openid/login":
			io.WriteString(w, openIDValidAnswer)
		case "/IPlayerService/GetOwnedGames/v1/":
			http.Redirect(w, r, landing.URL+"/elsewhere", http.StatusFound)
		}
	}))
	response := doWithoutRedirects(t, server.URL+steamCallbackTarget(ownerID, zipKey), "")
	if location := response.Header.Get("Location"); location != testPublicURL+"/download-page/"+zipKey {
		t.Fatalf("got %d %q", response.StatusCode, location)
	}
	want := []receivedRequest{
		{"upstream", "/openid/login", "", false},
		{"upstream", "/IPlayerService/GetOwnedGames/v1/", "", true},
		{"landing", "/elsewhere", "", false},
	}
	mu.Lock()
	defer mu.Unlock()
	if !slices.Equal(received, want) {
		t.Fatalf("received %+v, want %+v", received, want)
	}
}

func TestUpstreamTimeouts(t *testing.T) {
	cases := map[string]http.HandlerFunc{
		"no answer": func(w http.ResponseWriter, r *http.Request) {
			io.ReadAll(r.Body)
			<-r.Context().Done()
		},
		"trickling body": func(w http.ResponseWriter, r *http.Request) {
			io.ReadAll(r.Body)
			for {
				io.WriteString(w, " ")
				w.(http.Flusher).Flush()
				select {
				case <-r.Context().Done():
					return
				case <-time.After(50 * time.Millisecond):
				}
			}
		},
	}
	for name, handler := range cases {
		t.Run(name, func(t *testing.T) {
			upstreamServer := startServer(t, &http.Server{Handler: handler}, localListener(t))
			cfg := testConfig(t)
			cfg.Origins.SteamCommunity = upstreamServer.URL
			a := New(cfg, upstream.NewClient(200*time.Millisecond, 500*time.Millisecond), fixedClock)
			started := time.Now()
			assertProblem(t, serve(a, http.MethodGet, steamCallbackTarget(ownerID, zipKey)), http.StatusBadGateway, failureBody(steam.ErrUnavailable))
			if waited := time.Since(started); waited > 2*time.Second {
				t.Fatalf("gave up after %s", waited)
			}
		})
	}
}

// stingyListener gives every connection a small send buffer, so a client that stops reading stalls the server quickly.
type stingyListener struct{ net.Listener }

func (l stingyListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if tcp, ok := conn.(*net.TCPConn); ok {
		tcp.SetWriteBuffer(16 << 10)
	}
	return conn, err
}

func downloadServer(t *testing.T, size int, stallTimeout time.Duration) (net.Conn, *bufio.Reader) {
	t.Helper()
	a, _ := newTestApp(t, steamAnswers(map[string]string{ownerID: ownsTheGame}))
	putFile(t, a, zipName, make([]byte, size))
	a.stallTimeout = stallTimeout
	server := startServer(t, newHTTPServer(a), stingyListener{localListener(t)})
	conn, reader := dial(t, server)
	conn.(*net.TCPConn).SetReadBuffer(16 << 10)
	conn.Write([]byte("GET /download?file=" + zipKey + " HTTP/1.1\r\nHost: t\r\nCookie: " + tokenCookie(validToken(ownerID)) + "\r\n\r\n"))
	return conn, reader
}

func TestStalledDownloadIsDropped(t *testing.T) {
	const size = 16 << 20
	conn, reader := downloadServer(t, size, 200*time.Millisecond)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if status, err := reader.ReadString('\n'); err != nil || !strings.HasPrefix(status, "HTTP/1.1 200") {
		t.Fatalf("status %q, %v", status, err)
	}
	time.Sleep(time.Second)
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	received, err := io.Copy(io.Discard, reader)
	if isTimeout(err) {
		t.Fatal("the server kept the stalled download open")
	}
	if received >= size {
		t.Fatalf("received the whole file (%d bytes) although the download stalled", received)
	}
}

func TestSlowDownloadKeepsGoing(t *testing.T) {
	const size = 2 << 20
	stallTimeout := 300 * time.Millisecond
	conn, reader := downloadServer(t, size, stallTimeout)
	conn.SetReadDeadline(time.Now().Add(20 * time.Second))
	response, err := http.ReadResponse(reader, nil)
	if err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	var received int64
	buffer := make([]byte, 16<<10)
	for {
		n, err := response.Body.Read(buffer)
		received += int64(n)
		if err != nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if received != size {
		t.Fatalf("received %d of %d bytes", received, size)
	}
	if took := time.Since(started); took < 2*stallTimeout {
		t.Fatalf("the download took %s, too fast to show that a slow client keeps going", took)
	}
	time.Sleep(2 * stallTimeout)
	conn.Write([]byte("GET /download?file=x HTTP/1.1\r\nHost: t\r\n\r\n"))
	if answer := readAnswer(t, conn, reader, 2*time.Second); answer.header.Get("Location") != testPublicURL+"/purchase-to-download/x" {
		t.Fatalf("the connection is unusable after the download: %s", answer.status)
	}
}

func TestServeShutsDownGracefully(t *testing.T) {
	arrived, release := make(chan struct{}), make(chan struct{})
	a, _ := newTestApp(t, func(call upstreamtest.Call) (int, string, error) {
		if call.Path == "/openid/login" {
			close(arrived)
			<-release
			return http.StatusOK, openIDValidAnswer, nil
		}
		return http.StatusOK, ownsTheGame, nil
	})
	listener := localListener(t)
	ctx, stop := context.WithCancel(t.Context())
	served := make(chan error, 1)
	go func() { served <- Serve(ctx, listener, a) }()

	answered := make(chan string, 1)
	go func() {
		response, err := clientWithoutRedirects().Get("http://" + listener.Addr().String() + steamCallbackTarget(ownerID, zipKey))
		if err != nil {
			answered <- err.Error()
			return
		}
		response.Body.Close()
		answered <- response.Header.Get("Location")
	}()
	select {
	case <-arrived:
	case <-time.After(2 * time.Second):
		t.Fatal("the request never reached the upstream call")
	}
	stop()
	select {
	case err := <-served:
		t.Fatalf("Serve returned with a request in flight: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	close(release)
	select {
	case err := <-served:
		if err != nil {
			t.Fatalf("Serve: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Serve did not return after the request in flight was answered")
	}
	if location := <-answered; location != testPublicURL+"/download-page/"+zipKey {
		t.Fatalf("the request in flight got %q", location)
	}
	if conn, err := net.Dial("tcp", listener.Addr().String()); err == nil {
		conn.Close()
		t.Fatal("the listener is still open")
	}
}
