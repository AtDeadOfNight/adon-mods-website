package license

import "testing"

func TestCookieStrings(t *testing.T) {
	if got := SetCookie("a.b_c-d"); got != "adon-mods-token=a.b_c-d; Max-Age=31536000; Path=/; HttpOnly; Secure; SameSite=Lax" {
		t.Errorf("set: %s", got)
	}
	if ClearCookie != "adon-mods-token=; Max-Age=0; Path=/; Expires=Thu, 01 Jan 1970 00:00:00 GMT; Secure; SameSite=Lax" {
		t.Errorf("clear: %s", ClearCookie)
	}
}
