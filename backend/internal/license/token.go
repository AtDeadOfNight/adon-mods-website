package license

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"errors"
	"hash"
	"regexp"
	"strconv"
	"strings"
	"time"

	"git.hloth.dev/at-dead-of-night/mods.adon.hloth.dev/backend/internal/compat"
)

const (
	lifetime     = 365 * 24 * time.Hour
	issuedHeader = `{"alg":"HS256","typ":"JWT"}`
)

var (
	errMalformed = errors.New("malformed")
	errSignature = errors.New("signature does not match")
	errClaims    = errors.New("claims refused")

	tokenPattern   = regexp.MustCompile(`^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$`)
	steamIDPattern = regexp.MustCompile(`^[0-9]+$`)
	hashes         = map[string]func() hash.Hash{"HS256": sha256.New, "HS384": sha512.New384, "HS512": sha512.New}
)

// Issue makes the bytes of jwt.sign({ steamId }, secret, { expiresIn: one year }).
func Issue(steamID string, now time.Time, secret []byte) string {
	id, _ := json.Marshal(steamID)
	payload := `{"steamId":` + string(id) +
		`,"iat":` + strconv.FormatInt(now.Unix(), 10) +
		`,"exp":` + strconv.FormatInt(now.Add(lifetime).Unix(), 10) + `}`
	input := encodeSegment(issuedHeader) + "." + encodeSegment(payload)
	return input + "." + sign(input, sha256.New, secret)
}

// Verify accepts what jwt.verify(token, secret) and the old Zod schema let through.
func Verify(token string, now time.Time, secret []byte) (string, error) {
	if !tokenPattern.MatchString(token) {
		return "", errMalformed
	}
	parts := strings.Split(token, ".")
	encodedHeader, encodedClaims, signature := parts[0], parts[1], parts[2]
	header, ok := jsonObject(latin1(decodeSegment(encodedHeader)))
	if !ok {
		return "", errMalformed
	}
	alg, _ := jsonString(header["alg"])
	newHash, supported := hashes[alg]
	if !supported {
		return "", errMalformed
	}
	if !hmac.Equal([]byte(sign(encodedHeader+"."+encodedClaims, newHash, secret)), []byte(signature)) {
		return "", errSignature
	}
	claims, ok := jsonObject(string(decodeSegment(encodedClaims)))
	if !ok {
		return "", errMalformed
	}
	return claimedSteamID(claims, float64(now.Unix()))
}

func claimedSteamID(claims map[string]json.RawMessage, now float64) (string, error) {
	if raw, present := claims["nbf"]; present {
		if notBefore, ok := jsonNumber(raw); !ok || notBefore > now {
			return "", errClaims
		}
	}
	if raw, present := claims["exp"]; present {
		if expires, ok := jsonNumber(raw); !ok || now >= expires {
			return "", errClaims
		}
	}
	steamID, ok := jsonString(claims["steamId"])
	if !ok || !steamIDPattern.MatchString(steamID) {
		return "", errClaims
	}
	return steamID, nil
}

func jsonObject(text string) (map[string]json.RawMessage, bool) {
	var fields map[string]json.RawMessage
	err := json.Unmarshal([]byte(text), &fields)
	return fields, err == nil && fields != nil
}

func jsonString(raw json.RawMessage) (string, bool) {
	var text string
	if len(raw) == 0 || raw[0] != '"' || json.Unmarshal(raw, &text) != nil {
		return "", false
	}
	return text, true
}

func jsonNumber(raw json.RawMessage) (float64, bool) {
	return compat.Number(json.Number(raw))
}

func sign(input string, newHash func() hash.Hash, secret []byte) string {
	mac := hmac.New(newHash, secret)
	mac.Write([]byte(input))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func encodeSegment(text string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(text))
}

// decodeSegment is Node's Buffer.from(segment, 'base64'): a last lone character is ignored, and
// so are the unused bits of the last character.
func decodeSegment(segment string) []byte {
	if len(segment)%4 == 1 {
		segment = segment[:len(segment)-1]
	}
	decoded, _ := base64.RawURLEncoding.DecodeString(segment)
	return decoded
}

// jws read the header with toString('binary'): one character per byte.
func latin1(data []byte) string {
	runes := make([]rune, len(data))
	for i, b := range data {
		runes[i] = rune(b)
	}
	return string(runes)
}
