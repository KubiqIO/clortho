package service

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// signAndDecode signs with a throwaway key and returns the verified claims.
func signAndDecode(t *testing.T, expiresAt *time.Time, features []string, echoes ResponseEchoes) jwt.MapClaims {
	t.Helper()

	pubKey, privKey, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	tokenStr, err := SignLicense(base64.StdEncoding.EncodeToString(privKey), "TEST-KEY", expiresAt, true, features, echoes)
	require.NoError(t, err)

	token, err := jwt.Parse(tokenStr, func(*jwt.Token) (interface{}, error) { return pubKey, nil })
	require.NoError(t, err)
	require.True(t, token.Valid)

	claims, ok := token.Claims.(jwt.MapClaims)
	require.True(t, ok)
	return claims
}

func strPtr(s string) *string { return &s }

// usageClaim pulls the reported_usage object out of decoded claims.
func usageClaim(t *testing.T, claims jwt.MapClaims) (map[string]interface{}, bool) {
	t.Helper()
	raw, present := claims["reported_usage"]
	if !present {
		return nil, false
	}
	usage, ok := raw.(map[string]interface{})
	require.True(t, ok, "reported_usage must decode as an object")
	return usage, true
}

func TestSignLicense_ReportedUsageEcho(t *testing.T) {
	t.Run("echoes every reported field verbatim", func(t *testing.T) {
		claims := signAndDecode(t, nil, nil, ResponseEchoes{ReportedUsage: map[string]string{
			"clients": "200",
			"ips":     "20",
		}})
		usage, present := usageClaim(t, claims)
		require.True(t, present)
		assert.Equal(t, map[string]interface{}{"clients": "200", "ips": "20"}, usage)
	})

	t.Run("attaches no meaning to field names", func(t *testing.T) {
		// Clortho never needs to recognise what is being counted; an unfamiliar
		// field is echoed exactly like a familiar one.
		claims := signAndDecode(t, nil, nil, ResponseEchoes{ReportedUsage: map[string]string{
			"anything_at_all": "7",
		}})
		usage, _ := usageClaim(t, claims)
		assert.Equal(t, map[string]interface{}{"anything_at_all": "7"}, usage)
	})

	t.Run("echoes a non-numeric value untouched", func(t *testing.T) {
		// The claim is a verbatim echo, not a parsed count: whatever the client
		// sent has to come back byte for byte or the comparison is worthless.
		claims := signAndDecode(t, nil, nil, ResponseEchoes{ReportedUsage: map[string]string{"odd": " 12,7 "}})
		usage, _ := usageClaim(t, claims)
		assert.Equal(t, " 12,7 ", usage["odd"])
	})

	t.Run("claim is absent when no param arrived", func(t *testing.T) {
		claims := signAndDecode(t, nil, nil, ResponseEchoes{})
		_, present := claims["reported_usage"]
		assert.False(t, present, "reported_usage must be absent, not defaulted")
	})

	t.Run("an empty report is omitted rather than sent as an empty object", func(t *testing.T) {
		// {} would assert "the caller reported nothing", which is a different
		// statement from "no report arrived".
		claims := signAndDecode(t, nil, nil, ResponseEchoes{ReportedUsage: map[string]string{}})
		_, present := claims["reported_usage"]
		assert.False(t, present)
	})

	t.Run("absent is distinguishable from a reported zero", func(t *testing.T) {
		// A stripped param must never be readable as "this install reported 0
		// clients" - that is the exact tamper this claim exists to catch.
		absent := signAndDecode(t, nil, nil, ResponseEchoes{})
		zero := signAndDecode(t, nil, nil, ResponseEchoes{ReportedUsage: map[string]string{"clients": "0"}})

		_, absentPresent := absent["reported_usage"]
		assert.False(t, absentPresent)

		usage, present := usageClaim(t, zero)
		require.True(t, present)
		assert.Equal(t, "0", usage["clients"])
	})

	t.Run("a dropped field is distinguishable from a zeroed one", func(t *testing.T) {
		// Stripping one field out of a multi-field report must not quietly
		// reduce to a smaller report that still looks well formed.
		both := signAndDecode(t, nil, nil, ResponseEchoes{ReportedUsage: map[string]string{"clients": "200", "ips": "20"}})
		one := signAndDecode(t, nil, nil, ResponseEchoes{ReportedUsage: map[string]string{"clients": "200"}})

		bothUsage, _ := usageClaim(t, both)
		oneUsage, _ := usageClaim(t, one)

		assert.Contains(t, bothUsage, "ips")
		assert.NotContains(t, oneUsage, "ips")
	})

	t.Run("empty value is echoed as present and empty", func(t *testing.T) {
		// "?usage.clients=" did arrive, so the field is present; the client
		// compares it against what it sent and sees the mismatch.
		claims := signAndDecode(t, nil, nil, ResponseEchoes{ReportedUsage: map[string]string{"clients": ""}})
		usage, _ := usageClaim(t, claims)
		val, present := usage["clients"]
		assert.True(t, present)
		assert.Equal(t, "", val)
	})

	t.Run("caps each value at 64 chars", func(t *testing.T) {
		claims := signAndDecode(t, nil, nil, ResponseEchoes{ReportedUsage: map[string]string{
			"clients": strings.Repeat("9", 100),
			"ips":     "20",
		}})
		usage, _ := usageClaim(t, claims)
		assert.Equal(t, strings.Repeat("9", 64), usage["clients"])
		assert.Equal(t, "20", usage["ips"], "capping one field must not disturb another")
	})

	t.Run("a 64 char value is not truncated", func(t *testing.T) {
		exact := strings.Repeat("9", 64)
		claims := signAndDecode(t, nil, nil, ResponseEchoes{ReportedUsage: map[string]string{"clients": exact}})
		usage, _ := usageClaim(t, claims)
		assert.Equal(t, exact, usage["clients"])
	})

	t.Run("caps on a rune boundary", func(t *testing.T) {
		// Cutting mid-rune would leave invalid UTF-8 that JSON encoding would
		// silently rewrite, so the echo would no longer be verbatim.
		claims := signAndDecode(t, nil, nil, ResponseEchoes{ReportedUsage: map[string]string{"odd": strings.Repeat("é", 100)}})
		usage, _ := usageClaim(t, claims)
		assert.Equal(t, strings.Repeat("é", 64), usage["odd"])
	})

	t.Run("drops a field name longer than 64 chars", func(t *testing.T) {
		// Truncating the name would produce a field the caller never sent.
		claims := signAndDecode(t, nil, nil, ResponseEchoes{ReportedUsage: map[string]string{
			strings.Repeat("k", 65): "1",
			"clients":               "200",
		}})
		usage, _ := usageClaim(t, claims)
		assert.Equal(t, map[string]interface{}{"clients": "200"}, usage)
	})

	t.Run("keeps a field name of exactly 64 chars", func(t *testing.T) {
		exact := strings.Repeat("k", 64)
		claims := signAndDecode(t, nil, nil, ResponseEchoes{ReportedUsage: map[string]string{exact: "1"}})
		usage, _ := usageClaim(t, claims)
		assert.Equal(t, "1", usage[exact])
	})

	t.Run("caps the number of fields deterministically", func(t *testing.T) {
		// An intermediary can append params, so the report needs a ceiling. The
		// trim is by sorted key order, not Go's randomised map iteration.
		reported := make(map[string]string, 50)
		for i := range 50 {
			reported[fmt.Sprintf("field%02d", i)] = "1"
		}

		first, _ := usageClaim(t, signAndDecode(t, nil, nil, ResponseEchoes{ReportedUsage: reported}))
		second, _ := usageClaim(t, signAndDecode(t, nil, nil, ResponseEchoes{ReportedUsage: reported}))

		assert.Len(t, first, 32)
		assert.Equal(t, first, second, "the same report must cap to the same fields every time")
		assert.Contains(t, first, "field00")
		assert.Contains(t, first, "field31")
		assert.NotContains(t, first, "field32")
	})
}

func TestSignLicense_NonceEcho(t *testing.T) {
	t.Run("echoes the nonce verbatim", func(t *testing.T) {
		nonce := "a7F-3kZq_09xYtR2"
		claims := signAndDecode(t, nil, nil, ResponseEchoes{Nonce: strPtr(nonce)})
		assert.Equal(t, nonce, claims["nonce"])
	})

	t.Run("claim is absent when no nonce arrived", func(t *testing.T) {
		claims := signAndDecode(t, nil, nil, ResponseEchoes{})
		_, present := claims["nonce"]
		assert.False(t, present)
	})

	t.Run("ignores a nonce longer than 64 chars", func(t *testing.T) {
		// Truncating would produce a nonce that can never match the one the
		// client generated, so it is dropped instead.
		claims := signAndDecode(t, nil, nil, ResponseEchoes{Nonce: strPtr(strings.Repeat("n", 65))})
		_, present := claims["nonce"]
		assert.False(t, present)
	})

	t.Run("keeps a nonce of exactly 64 chars", func(t *testing.T) {
		exact := strings.Repeat("n", 64)
		claims := signAndDecode(t, nil, nil, ResponseEchoes{Nonce: strPtr(exact)})
		assert.Equal(t, exact, claims["nonce"])
	})

	t.Run("the two echoes are independent", func(t *testing.T) {
		claims := signAndDecode(t, nil, nil, ResponseEchoes{Nonce: strPtr("only-nonce")})
		assert.Equal(t, "only-nonce", claims["nonce"])
		_, present := claims["reported_usage"]
		assert.False(t, present)
	})
}

func TestSignLicense_IncludesIssuedAt(t *testing.T) {
	before := time.Now().Unix()
	claims := signAndDecode(t, nil, nil, ResponseEchoes{})
	after := time.Now().Unix()

	iat, ok := claims["iat"].(float64)
	require.True(t, ok, "token must carry iat so clients can enforce freshness")
	assert.GreaterOrEqual(t, int64(iat), before)
	assert.LessOrEqual(t, int64(iat), after)
}

// TestSignLicense_NoEchoesIsByteCompatible pins the promise that this change is
// safe to deploy before any client verifies the new claims: a request that sends
// neither param produces exactly the claim set clients already parse, plus the
// newly required iat and nothing else.
func TestSignLicense_NoEchoesIsByteCompatible(t *testing.T) {
	pubKey, privKey, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	expiresAt := time.Date(2030, 12, 31, 23, 59, 59, 0, time.UTC)
	features := []string{"sso", "audit"}

	tokenStr, err := SignLicense(base64.StdEncoding.EncodeToString(privKey), "TEST-KEY", &expiresAt, false, features, ResponseEchoes{})
	require.NoError(t, err)

	token, err := jwt.Parse(tokenStr, func(*jwt.Token) (interface{}, error) { return pubKey, nil })
	require.NoError(t, err)
	require.True(t, token.Valid)
	assert.Equal(t, "EdDSA", token.Method.Alg(), "signing method must stay EdDSA")

	claims, ok := token.Claims.(jwt.MapClaims)
	require.True(t, ok)

	// iat is the one addition; strip it and the payload must be byte-identical
	// to what today's claim set encodes to.
	_, hasIat := claims["iat"]
	assert.True(t, hasIat)
	delete(claims, "iat")

	legacy := map[string]interface{}{
		"sub":      "TEST-KEY",
		"iss":      "clortho",
		"valid":    false,
		"features": features,
		"exp":      expiresAt.Unix(),
	}

	// Both sides go through encoding/json, which sorts map keys, so this is a
	// true byte comparison of the payload rather than a field-by-field check.
	want, err := json.Marshal(legacy)
	require.NoError(t, err)
	got, err := json.Marshal(claims)
	require.NoError(t, err)
	assert.JSONEq(t, string(want), string(got))
	assert.Equal(t, string(want), string(got), "claims must encode to identical bytes")
}
