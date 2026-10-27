package server

import (
	"errors"
	"log"
	"net/http"

	"git.hloth.dev/at-dead-of-night/mods.adon.hloth.dev/backend/internal/compat"
	"git.hloth.dev/at-dead-of-night/mods.adon.hloth.dev/backend/internal/download"
	"git.hloth.dev/at-dead-of-night/mods.adon.hloth.dev/backend/internal/httpx"
	"git.hloth.dev/at-dead-of-night/mods.adon.hloth.dev/backend/internal/license"
	"git.hloth.dev/at-dead-of-night/mods.adon.hloth.dev/backend/internal/steam"
)

func (a *App) redirectToSite(w http.ResponseWriter, path string) {
	httpx.Redirect(w, compat.SerializeURL(a.cfg.PublicURL+path))
}

func (a *App) redirectToPurchase(w http.ResponseWriter, file string) {
	a.redirectToSite(w, "/purchase-to-download/"+file)
}

func (a *App) clearTokenAndRedirectToPurchase(w http.ResponseWriter, file string) {
	w.Header().Add("Set-Cookie", license.ClearCookie)
	a.redirectToPurchase(w, file)
}

func (a *App) steamLogin(w http.ResponseWriter, r *http.Request) {
	httpx.Redirect(w, a.steam.LoginURL(compat.ParseQuery(r.URL.RawQuery)["file"]))
}

func (a *App) steamCallback(w http.ResponseWriter, r *http.Request) {
	query := compat.ParseQuery(r.URL.RawQuery)
	steamID, err := a.steam.Verify(r.Context(), query)
	if err != nil {
		failure := steam.ErrUnavailable
		errors.As(err, &failure)
		writeFailure(w, failure)
		return
	}
	switch a.steam.License(r.Context(), steamID) {
	case license.Owned:
		w.Header().Add("Set-Cookie", license.SetCookie(license.Issue(steamID, a.now(), a.cfg.JWTSecret)))
		if file := query["file"]; file != "" {
			a.redirectToSite(w, "/download-page/"+file)
		} else {
			a.redirectToSite(w, "")
		}
	case license.NotOwned:
		a.redirectToSite(w, "/error/game-not-owned")
	case license.Unknown:
		a.redirectToSite(w, "/error/game-owned-unknown")
	default:
		writeFailure(w, steam.ErrOwnership)
	}
}

func writeFailure(w http.ResponseWriter, failure *steam.Failure) {
	httpx.LocalizedError(w, failure.Status, failure.Message, failure.MessageRu)
}

func (a *App) download(w http.ResponseWriter, r *http.Request) {
	file, named := compat.ParseQuery(r.URL.RawQuery)["file"]
	if !named {
		// The old code joined the missing value into the redirect as the text "undefined".
		file = "undefined"
	}
	token := license.Token(r)
	if token == "" {
		a.redirectToPurchase(w, file)
		return
	}
	steamID, err := license.Verify(token, a.now(), a.cfg.JWTSecret)
	if err != nil {
		log.Printf("download: license token refused: %v", err)
		if clearUnusableToken {
			a.clearTokenAndRedirectToPurchase(w, file)
		} else {
			writeInvalidCookie(w)
		}
		return
	}
	switch a.steam.License(r.Context(), steamID) {
	case license.Owned, license.Unknown:
		// Only a games list without the app refuses, as the old Zod schema did, and Steam answers an
		// account without the game with no list (Unknown), so a refunded account keeps downloading.
		a.renewCookieAndServe(w, r, file, token)
	case license.NotOwned:
		a.clearTokenAndRedirectToPurchase(w, file)
	default:
		// The old backend's answer, although the cookie is not to blame when Steam cannot be asked.
		writeInvalidCookie(w)
	}
}

func (a *App) renewCookieAndServe(w http.ResponseWriter, r *http.Request, file, token string) {
	w.Header().Add("Set-Cookie", license.SetCookie(token))
	name, known := download.FileName(file)
	if !known {
		httpx.Error(w, http.StatusNotFound, "File not found")
		return
	}
	if err := download.Serve(w, r, a.cfg.DownloadsDir, name, a.stallTimeout); err != nil {
		log.Printf("download: %v", err)
		httpx.ServerError(w)
	}
}

func writeInvalidCookie(w http.ResponseWriter) {
	httpx.Error(w, http.StatusBadRequest, "Invalid cookie. Please clear the website's cookies and sign in with Steam again.")
}
