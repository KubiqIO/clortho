package service

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"sort"
	"time"
	"unicode/utf8"

	"github.com/golang-jwt/jwt/v5"
)

// maxEchoRunes bounds every echoed key and value, so a caller cannot inflate
// the signed token with arbitrary data.
const maxEchoRunes = 64

// maxUsageKeys bounds how many entries one usage report can carry. Callers
// choose their own field names, so without a ceiling an intermediary could
// append params until the token is unreasonable to sign or parse.
const maxUsageKeys = 32

// ResponseEchoes holds request values that /check reflects back verbatim inside
// the signed token. The reporter holds the shared secret, so signing the request
// cannot prove what it sent; echoing what the server received inside a response
// no intermediary can forge lets the client detect a rewrite in flight.
//
// An empty field means the caller sent no such param and the claim is omitted.
// This is load-bearing: an absent usage report must never be readable as a
// reported zero.
//
// ReportedUsage keys are whatever the caller chose to count. Clortho attaches no
// meaning to them and never parses the values.
type ResponseEchoes struct {
	ReportedUsage map[string]string
	Nonce         *string
}

// SignLicense generates a JWT containing license claims for offline verification.
func SignLicense(privateKeyBase64 string, key string, expiresAt *time.Time, valid bool, features []string, echoes ResponseEchoes) (string, error) {
	if privateKeyBase64 == "" {
		return "", fmt.Errorf("private key is empty")
	}

	privateKeyBytes, err := base64.StdEncoding.DecodeString(privateKeyBase64)
	if err != nil {
		return "", fmt.Errorf("failed to decode private key: %w", err)
	}

	if len(privateKeyBytes) != ed25519.PrivateKeySize {
		return "", fmt.Errorf("invalid private key size: %d", len(privateKeyBytes))
	}

	privateKey := ed25519.PrivateKey(privateKeyBytes)

	claims := jwt.MapClaims{
		"sub":      key,
		"iss":      "clortho",
		"valid":    valid,
		"features": features,
		// iat lets a client reject a stale response a proxy captured on a
		// genuinely low-usage day and replayed later. It pairs with the nonce
		// echo below, which pins a response to one specific request.
		"iat": time.Now().Unix(),
	}

	if expiresAt != nil {
		claims["exp"] = expiresAt.Unix()
	}

	// Echoes are reflected exactly as received: the client compares them against
	// what it sent, so a rewritten or stripped value shows up as a mismatch.
	// Nothing here is parsed, metered or stored - Keyva owns metering.
	//
	// An empty report is omitted rather than sent as {}, which would assert the
	// caller reported nothing rather than that no report arrived.
	if usage := cappedUsageEcho(echoes.ReportedUsage); len(usage) > 0 {
		claims["reported_usage"] = usage
	}

	// An over-long nonce is dropped rather than truncated: the client matches it
	// against the value it generated, so a truncated nonce carries no signal.
	if echoes.Nonce != nil && utf8.RuneCountInString(*echoes.Nonce) <= maxEchoRunes {
		claims["nonce"] = *echoes.Nonce
	}

	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	return token.SignedString(privateKey)
}

// cappedUsageEcho applies the size limits to a usage report. Values are kept
// verbatim up to the cap; keys are dropped rather than truncated, since a
// shortened field name could never match the one the caller sent.
func cappedUsageEcho(reported map[string]string) map[string]string {
	keys := make([]string, 0, len(reported))
	for k := range reported {
		if utf8.RuneCountInString(k) <= maxEchoRunes {
			keys = append(keys, k)
		}
	}

	// Sort before capping so an over-full report is trimmed the same way every
	// time, rather than by Go's randomised map iteration order.
	sort.Strings(keys)
	if len(keys) > maxUsageKeys {
		keys = keys[:maxUsageKeys]
	}

	echo := make(map[string]string, len(keys))
	for _, k := range keys {
		echo[k] = truncateRunes(reported[k], maxEchoRunes)
	}
	return echo
}

// truncateRunes caps s at limit runes, cutting on a rune boundary so the result
// stays valid UTF-8 and survives JSON encoding unaltered.
func truncateRunes(s string, limit int) string {
	if utf8.RuneCountInString(s) <= limit {
		return s
	}
	count := 0
	for i := range s {
		if count == limit {
			return s[:i]
		}
		count++
	}
	return s
}
