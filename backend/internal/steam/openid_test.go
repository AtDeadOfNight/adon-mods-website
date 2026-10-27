package steam

import (
	"errors"
	"testing"
)

const (
	testCallback = "https://mods.adon.hloth.dev/auth/steam/callback"
	testID       = "76561197960010001"
)

func validQuery() map[string]string {
	return map[string]string{
		"openid.ns":             openIDNamespace,
		"openid.mode":           "id_res",
		"openid.op_endpoint":    openIDEndpoint,
		"openid.claimed_id":     claimedIDBase + testID,
		"openid.identity":       claimedIDBase + testID,
		"openid.return_to":      testCallback + "?file=AtDeadOfNight.app",
		"openid.response_nonce": "2026-10-27T12:00:00Zabc",
		"openid.assoc_handle":   "1234567890",
		"openid.signed":         "signed,op_endpoint,claimed_id,identity,return_to,response_nonce,assoc_handle",
		"openid.sig":            "Kq3+/parity=",
		"file":                  "AtDeadOfNight.app",
	}
}

func TestAssertionChecks(t *testing.T) {
	assertion, err := parseAssertion(validQuery())
	if err != nil {
		t.Fatal(err)
	}
	if id, err := assertion.steamID(testCallback); err != nil || id != testID {
		t.Fatalf("valid assertion: %q, %v", id, err)
	}

	cases := map[string]struct {
		field, value string
		err          error
	}{
		"other namespace":       {"openid.ns", "urn:other", ErrState},
		"checkid_setup":         {"openid.mode", "checkid_setup", ErrState},
		"other endpoint":        {"openid.op_endpoint", "https://evil.example/openid/login", ErrState},
		"other return_to host":  {"openid.return_to", "https://evil.example/auth/steam/callback", ErrState},
		"longer return_to path": {"openid.return_to", testCallback + "x", ErrState},
		"return_to slash":       {"openid.return_to", testCallback + "/", ErrState},
		"return_to over http":   {"openid.return_to", "http://mods.adon.hloth.dev/auth/steam/callback", ErrState},
		"unsigned claimed_id":   {"openid.signed", "signed,op_endpoint,identity,return_to,response_nonce,assoc_handle", ErrState},
		"identity differs":      {"openid.identity", claimedIDBase + "76561197960030001", ErrState},
	}
	for name, c := range cases {
		query := validQuery()
		query[c.field] = c.value
		assertion, _ := parseAssertion(query)
		if _, err := assertion.steamID(testCallback); !errors.Is(err, c.err) {
			t.Errorf("%s: %v, want %v", name, err, c.err)
		}
	}
	for name, c := range map[string]struct {
		claimedID string
		err       error
	}{
		"other host":                         {"https://evil.example/openid/id/" + testID, ErrState},
		"http":                               {"http://steamcommunity.com/openid/id/" + testID, ErrState},
		"short":                              {claimedIDBase + "123", ErrID},
		"long":                               {claimedIDBase + "9" + testID, ErrID},
		"letters":                            {claimedIDBase + "notanumber", ErrID},
		"trailing slash":                     {claimedIDBase + testID + "/", ErrID},
		"17 bytes with a letter":             {claimedIDBase + "7656119796001000x", ErrID},
		"17 bytes with an Arabic digit":      {claimedIDBase + "765611979600100١", ErrID},
		"17 characters with an Arabic digit": {claimedIDBase + "7656119796001000١", ErrID},
	} {
		query := validQuery()
		query["openid.claimed_id"], query["openid.identity"] = c.claimedID, c.claimedID
		assertion, _ := parseAssertion(query)
		if _, err := assertion.steamID(testCallback); !errors.Is(err, c.err) {
			t.Errorf("claimed_id %s: %v, want %v", name, err, c.err)
		}
	}
}

func TestParseAssertionRequiresEveryField(t *testing.T) {
	for _, field := range []string{"ns", "mode", "op_endpoint", "claimed_id", "identity", "return_to", "response_nonce", "assoc_handle", "signed", "sig"} {
		query := validQuery()
		delete(query, "openid."+field)
		if _, err := parseAssertion(query); !errors.Is(err, ErrCallback) {
			t.Errorf("without openid.%s: %v, want %v", field, err, ErrCallback)
		}
	}
}

func TestCheckAuthenticationForm(t *testing.T) {
	assertion, _ := parseAssertion(validQuery())
	want := "openid.ns=http%3A%2F%2Fspecs.openid.net%2Fauth%2F2.0&openid.mode=check_authentication" +
		"&openid.op_endpoint=https%3A%2F%2Fsteamcommunity.com%2Fopenid%2Flogin" +
		"&openid.claimed_id=https%3A%2F%2Fsteamcommunity.com%2Fopenid%2Fid%2F76561197960010001" +
		"&openid.identity=https%3A%2F%2Fsteamcommunity.com%2Fopenid%2Fid%2F76561197960010001" +
		"&openid.return_to=https%3A%2F%2Fmods.adon.hloth.dev%2Fauth%2Fsteam%2Fcallback%3Ffile%3DAtDeadOfNight.app" +
		"&openid.response_nonce=2026-10-27T12%3A00%3A00Zabc&openid.assoc_handle=1234567890" +
		"&openid.signed=signed%2Cop_endpoint%2Cclaimed_id%2Cidentity%2Creturn_to%2Cresponse_nonce%2Cassoc_handle" +
		"&openid.sig=Kq3%2B%2Fparity%3D"
	if got := assertion.checkAuthenticationForm(); got != want {
		t.Fatalf("form\n%s\nwant\n%s", got, want)
	}
}

func TestConfirmsAssertion(t *testing.T) {
	cases := map[string]bool{
		"ns:http://specs.openid.net/auth/2.0\nis_valid:true\n":     true,
		"ns:http://specs.openid.net/auth/2.0\r\nis_valid:true\r\n": true,
		"\xef\xbb\xbfis_valid:true\n":                              true,
		"is_valid:true":                                            true,
		"ns:x\nis_valid:false\n":                                   false,
		"ns:x\nis_valid:false\ninvalidate_handle:is_valid:true\n":  false,
		"ns:x is_valid:true\n":                                     false,
		"is_valid:false\nis_valid:true\n":                          false,
		"is_valid:true\nis_valid:true\n":                           false,
		"is_valid: true\n":                                         false,
		"<html>is_valid:true</html>":                               false,
		"":                                                         false,
	}
	for body, want := range cases {
		if got := confirmsAssertion([]byte(body)); got != want {
			t.Errorf("confirmsAssertion(%q) = %v", body, got)
		}
	}
}
