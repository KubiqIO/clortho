package middleware

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestResponseSigningMiddleware(t *testing.T) {
	// Generate a valid key pair
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	assert.NoError(t, err)

	privBase64 := base64.StdEncoding.EncodeToString(priv)
	
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ResponseSigningMiddleware(privBase64))
	
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "hello world")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "hello world", w.Body.String())

	sigHeader := w.Header().Get("X-Clortho-Signature")
	tsHeader := w.Header().Get("X-Clortho-Timestamp")

	assert.NotEmpty(t, sigHeader, "Signature header should be present")
	assert.NotEmpty(t, tsHeader, "Timestamp header should be present")

	// Verify signature
	sigBytes, err := base64.StdEncoding.DecodeString(sigHeader)
	assert.NoError(t, err)

	payload := tsHeader + "." + w.Body.String()
	valid := ed25519.Verify(pub, []byte(payload), sigBytes)
	assert.True(t, valid, "Signature should be valid")
}

func TestResponseSigningMiddleware_NoKey(t *testing.T) {
	r := gin.New()
	r.Use(ResponseSigningMiddleware(""))
	
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Empty(t, w.Header().Get("X-Clortho-Signature"))
}

func TestResponseSigningMiddleware_InvalidKey(t *testing.T) {
	r := gin.New()
	r.Use(ResponseSigningMiddleware("invalid-base64"))
	
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	// Should not panic, should just pass through
	assert.Equal(t, 200, w.Code)
	assert.Empty(t, w.Header().Get("X-Clortho-Signature"))
}
