package license

import (
	"net/http"
	"strconv"
	"time"

	"git.hloth.dev/at-dead-of-night/mods.adon.hloth.dev/backend/internal/compat"
)

const (
	cookieName = "adon-mods-token"
	// The site is HTTPS only, and Lax still sends the cookie when a link leads to /download.
	securityAttributes = "; Secure; SameSite=Lax"
	ClearCookie        = cookieName + "=; Max-Age=0; Path=/; Expires=Thu, 01 Jan 1970 00:00:00 GMT" + securityAttributes
)

func Token(r *http.Request) string {
	return compat.Cookie(r, cookieName)
}

// SetCookie keeps a token for a year from now; the token's own exp does not move.
func SetCookie(token string) string {
	return cookieName + "=" + compat.EncodeURIComponent(token) +
		"; Max-Age=" + strconv.Itoa(int(lifetime/time.Second)) + "; Path=/; HttpOnly" + securityAttributes
}
