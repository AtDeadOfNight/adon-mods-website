package license

import (
	"errors"
	"strings"
	"testing"
	"time"
)

var testSecret = []byte("test-secret-for-parity")

// Made by jsonwebtoken 9.0.2: jwt.sign({ steamId: '76561197960010001' }, secret, { expiresIn: 31536000 }) at 1793105599.
const jsonwebtokenVector = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdGVhbUlkIjoiNzY1NjExOTc5NjAwMTAwMDEiLCJpYXQiOjE3OTMxMDU1OTksImV4cCI6MTgyNDY0MTU5OX0.KoWcBlvnHFsPW3eswzuTcoQfDsk4l1FatDpzmURKMfw"

var testNow = time.Unix(1793105599, 0)

func TestIssueMatchesJsonwebtoken(t *testing.T) {
	if token := Issue("76561197960010001", testNow, testSecret); token != jsonwebtokenVector {
		t.Fatalf("got  %s\nwant %s", token, jsonwebtokenVector)
	}
	if id, err := Verify(jsonwebtokenVector, testNow, testSecret); err != nil || id != "76561197960010001" {
		t.Fatalf("verify: %q, %v", id, err)
	}
}

func signedWith(alg string, secret []byte, header, payload string) string {
	input := encodeSegment(header) + "." + encodeSegment(payload)
	return input + "." + sign(input, hashes[alg], secret)
}

func signed(header, payload string) string {
	return signedWith("HS256", testSecret, header, payload)
}

func TestVerify(t *testing.T) {
	const head = `{"alg":"HS256","typ":"JWT"}`
	valid := signed(head, `{"steamId":"76561197960010001","exp":4102444800}`)
	accepted := map[string]string{
		"HS384":               signedWith("HS384", testSecret, `{"alg":"HS384"}`, `{"steamId":"1"}`),
		"HS512, typ JOSE":     signedWith("HS512", testSecret, `{"alg":"HS512","typ":"JOSE"}`, `{"steamId":"1"}`),
		"no exp":              signed(head, `{"steamId":"1","iat":1}`),
		"exp beyond float":    signed(head, `{"steamId":"1","exp":1e400}`),
		"fractional exp":      signed(head, `{"steamId":"1","exp":1793105599.5}`),
		"nbf now":             signed(head, `{"steamId":"1","nbf":1793105599}`),
		"last steamId wins":   signed(head, `{"steamId":"x","steamId":"1"}`),
		"extra claims":        signed(head, ` {"a":[1,{"b":null}],"steamId":"1"} `),
		"invalid UTF-8 field": signed(head, "{\"steamId\":\"1\",\"x\":\"\xff\"}"),
		"latin-1 header":      signed("{\"alg\":\"HS256\",\"x\":\"\xc3\xa9\"}", `{"steamId":"1"}`),
	}
	for name, token := range accepted {
		if id, err := Verify(token, testNow, testSecret); err != nil || id != "1" {
			t.Errorf("%s: %q, %v; want \"1\"", name, id, err)
		}
	}

	refused := map[string]struct {
		token string
		err   error
	}{
		"empty":                {"", errMalformed},
		"two parts":            {valid[:strings.LastIndexByte(valid, '.')], errMalformed},
		"no signature":         {valid[:strings.LastIndexByte(valid, '.')+1], errMalformed},
		"padded signature":     {valid + "=", errMalformed},
		"tampered signature":   {valid[:len(valid)-2] + "AA", errSignature},
		"other secret":         {signedWith("HS256", []byte("other"), head, `{"steamId":"1"}`), errSignature},
		"alg none":             {signed(`{"alg":"none"}`, `{"steamId":"1"}`), errMalformed},
		"alg RS256":            {signed(`{"alg":"RS256"}`, `{"steamId":"1"}`), errMalformed},
		"alg lowercase":        {signed(`{"alg":"hs256"}`, `{"steamId":"1"}`), errMalformed},
		"header null":          {signed(`null`, `{"steamId":"1"}`), errMalformed},
		"header array":         {signed(`["HS256"]`, `{"steamId":"1"}`), errMalformed},
		"header of HS384":      {signedWith("HS256", testSecret, `{"alg":"HS384"}`, `{"steamId":"1"}`), errSignature},
		"payload null":         {signed(head, `null`), errMalformed},
		"payload array":        {signed(head, `[{"steamId":"1"}]`), errMalformed},
		"payload string":       {signed(head, `"{\"steamId\":\"1\"}"`), errMalformed},
		"payload BOM":          {signed(head, "\xef\xbb\xbf{\"steamId\":\"1\"}"), errMalformed},
		"expired":              {signed(head, `{"steamId":"1","exp":1793105599}`), errClaims},
		"exp as string":        {signed(head, `{"steamId":"1","exp":"4102444800"}`), errClaims},
		"exp null":             {signed(head, `{"steamId":"1","exp":null}`), errClaims},
		"nbf in the future":    {signed(head, `{"steamId":"1","nbf":1793105600}`), errClaims},
		"steamId number":       {signed(head, `{"steamId":1}`), errClaims},
		"steamId empty":        {signed(head, `{"steamId":""}`), errClaims},
		"steamId with newline": {signed(head, `{"steamId":"1\n"}`), errClaims},
		"steamId Arabic":       {signed(head, `{"steamId":"٣"}`), errClaims},
		"no steamId":           {signed(head, `{"SteamId":"1"}`), errClaims},
	}
	for name, c := range refused {
		if id, err := Verify(c.token, testNow, testSecret); !errors.Is(err, c.err) {
			t.Errorf("%s: %q, %v; want %v", name, id, err, c.err)
		}
	}
}

func TestDecodeSegmentFollowsNode(t *testing.T) {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	canonical := encodeSegment(`{"x":12}`)
	last := strings.IndexByte(alphabet, canonical[len(canonical)-1])
	cases := map[string]struct{ text, encoded string }{
		"unused bits set":    {`{"x":12}`, canonical[:len(canonical)-1] + alphabet[last+1:last+2]},
		"dangling character": {`{"x":123}`, encodeSegment(`{"x":123}`) + "A"},
	}
	for name, c := range cases {
		if string(decodeSegment(c.encoded)) != c.text {
			t.Errorf("%s: %q", name, decodeSegment(c.encoded))
		}
	}
}

func TestTokenIsOneYear(t *testing.T) {
	issued := time.Unix(0, 0)
	token := Issue("1", issued, testSecret)
	if _, err := Verify(token, issued.Add(lifetime-time.Second), testSecret); err != nil {
		t.Fatalf("a second before expiry: %v", err)
	}
	if _, err := Verify(token, issued.Add(lifetime), testSecret); !errors.Is(err, errClaims) {
		t.Fatalf("at expiry: %v, want %v", err, errClaims)
	}
}
