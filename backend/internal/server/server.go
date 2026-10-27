package server

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"git.hloth.dev/at-dead-of-night/mods.adon.hloth.dev/backend/internal/compat"
	"git.hloth.dev/at-dead-of-night/mods.adon.hloth.dev/backend/internal/config"
	"git.hloth.dev/at-dead-of-night/mods.adon.hloth.dev/backend/internal/download"
	"git.hloth.dev/at-dead-of-night/mods.adon.hloth.dev/backend/internal/httpx"
	"git.hloth.dev/at-dead-of-night/mods.adon.hloth.dev/backend/internal/steam"
)

const (
	// The old backend answered an unusable token cookie with a 400 that asked people to clear
	// their cookies by hand, and kept the cookie. Set to false to restore that answer.
	clearUnusableToken = true
	requestBodyTimeout = 30 * time.Second
	shutdownGrace      = 10 * time.Second
)

type App struct {
	cfg          config.Config
	now          func() time.Time
	steam        *steam.Client
	routes       map[string]http.HandlerFunc
	stallTimeout time.Duration
}

func New(cfg config.Config, client *http.Client, now func() time.Time) *App {
	a := &App{
		cfg:          cfg,
		now:          now,
		steam:        steam.New(cfg, client, now),
		stallTimeout: download.StallTimeout,
	}
	a.routes = map[string]http.HandlerFunc{
		"/auth/steam":          a.steamLogin,
		"/auth/steam/callback": a.steamCallback,
		"/download":            a.download,
	}
	return a
}

func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer httpx.Recover(w)
	if r.ContentLength != 0 {
		httpx.GuardBody(w, requestBodyTimeout)
		defer httpx.DropUnreadBody(w)
	}
	handler := a.routes[compat.RoutePath(requestPath(r))]
	if handler == nil || (r.Method != http.MethodGet && r.Method != http.MethodHead) {
		httpx.Error(w, http.StatusNotFound, "Not found")
		return
	}
	handler(w, r)
}

// The path as sent: net/http's URL.Path is already percent-decoded, which Bun's router never did.
func requestPath(r *http.Request) string {
	path, _, _ := strings.Cut(r.RequestURI, "?")
	if !strings.HasPrefix(path, "/") {
		return r.URL.EscapedPath()
	}
	return path
}

func newHTTPServer(handler http.Handler) *http.Server {
	return &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		// Longer than Caddy's 2 minute keep-alive, so the proxy closes idle connections first.
		IdleTimeout:                  150 * time.Second,
		MaxHeaderBytes:               64 << 10,
		DisableGeneralOptionsHandler: true,
	}
}

func Serve(ctx context.Context, listener net.Listener, handler http.Handler) error {
	server := newHTTPServer(handler)
	served := make(chan error, 1)
	go func() { served <- server.Serve(listener) }()
	log.Printf("Server running on http://%s", listener.Addr())

	select {
	case err := <-served:
		return err
	case <-ctx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownGrace)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Print("shutdown: closing the connections still open")
		server.Close()
	}
	if err := <-served; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
