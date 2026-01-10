package middleware

import (
	"bytes"
	"crypto/ed25519"

	"encoding/base64"
	"fmt"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

type responseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (r responseBodyWriter) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

func ResponseSigningMiddleware(privateKeyBase64 string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if privateKeyBase64 == "" {
			c.Next()
			return
		}

		// Decode private key
		privateKeyBytes, err := base64.StdEncoding.DecodeString(privateKeyBase64)
		if err != nil {

			slog.Error("Invalid response signing key", "error", err)
			c.Next()
			return
		}

		if len(privateKeyBytes) != ed25519.PrivateKeySize {
			slog.Error("Invalid response signing key size", "size", len(privateKeyBytes), "expected", ed25519.PrivateKeySize)
			c.Next()
			return
		}

		privateKey := ed25519.PrivateKey(privateKeyBytes)

		w := &responseBodyWriter{body: &bytes.Buffer{}, ResponseWriter: c.Writer}
		c.Writer = w

		c.Next()

		timestamp := time.Now().UTC().Format(time.RFC3339)
		body := w.body.Bytes()
		
		// Signature payload: timestamp + body
		// Prevents replay of body with old timestamp
		payload := fmt.Sprintf("%s.%s", timestamp, string(body))
		
		signature := ed25519.Sign(privateKey, []byte(payload))
		signatureBase64 := base64.StdEncoding.EncodeToString(signature)

		c.Header("X-Clortho-Signature", signatureBase64)
		c.Header("X-Clortho-Timestamp", timestamp)
	}
}
