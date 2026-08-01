package service

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
)

const defaultCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func ParseCharset(input string) (string, error) {
	if input == "" {
		return defaultCharset, nil
	}
	var builder strings.Builder
	parts := strings.Split(input, ",")
	for _, part := range parts {
		if len(part) == 3 && part[1] == '-' {
			start := part[0]
			end := part[2]
			if start > end {
				return "", fmt.Errorf("invalid range: %s", part)
			}
			for i := start; i <= end; i++ {
				builder.WriteByte(i)
			}
		} else {
			builder.WriteString(part)
		}
	}
	return builder.String(), nil
}

// MaxLicenseKeyLength bounds a caller-supplied key. license_check_logs.license_key
// is VARCHAR(255), so a longer key would validate fine and then fail to log.
const MaxLicenseKeyLength = 255

// ValidateLicenseKey checks a caller-supplied license key, as accepted by
// POST /admin/keys when importing keys that already exist elsewhere.
//
// Generated keys are drawn from a charset, but an imported key is whatever the
// system it came from produced, so this only rejects what would actually break:
// keys travel in the X-License-Key request header, where whitespace and control
// characters are a header-injection vector rather than an unusual key.
func ValidateLicenseKey(key string) error {
	if key == "" {
		return fmt.Errorf("license key must not be empty")
	}
	if len(key) > MaxLicenseKeyLength {
		return fmt.Errorf("license key must be at most %d characters, got %d", MaxLicenseKeyLength, len(key))
	}
	for i := 0; i < len(key); i++ {
		if c := key[i]; c < '!' || c > '~' {
			return fmt.Errorf("license key must contain only printable non-whitespace ASCII, found byte %#x at position %d", c, i)
		}
	}
	return nil
}

func GenerateLicenseKey(prefix string, length int, separator string, charset string) (string, error) {
	if length <= 0 {
		length = 12 // Default length
	}

	if separator == "" {
		separator = "-"
	}

	usedCharset := defaultCharset
	if charset != "" {
		usedCharset = charset
	}

	b := make([]byte, length)
	for i := range b {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(usedCharset))))
		if err != nil {
			return "", err
		}
		b[i] = usedCharset[num.Int64()]
	}

	randomString := string(b)
	if prefix != "" {
		return prefix + separator + randomString, nil
	}
	return randomString, nil
}
