package service

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// SignLicense generates a JWT containing license claims for offline verification.
func SignLicense(privateKeyBase64 string, key string, expiresAt *time.Time, valid bool, features []string) (string, error) {
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
	}

	if expiresAt != nil {
		claims["exp"] = expiresAt.Unix()
	}

	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	return token.SignedString(privateKey)
}
