package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"clortho/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		cfg            config.RateLimitConfig
		requests       int
		expectedStatus int
		expectFailures bool
	}{
		{
			name: "Allow within limit",
			cfg: config.RateLimitConfig{
				RequestsPerSecond: 10,
				Burst:             20,
				Enabled:           true,
			},
			requests:       10,
			expectedStatus: http.StatusOK,
			expectFailures: false,
		},
		{
			name: "Block after burst",
			cfg: config.RateLimitConfig{
				RequestsPerSecond: 1,
				Burst:             1,
				Enabled:           true,
			},
			requests:       5, // 1 allowed, 4 blocked (approx)
			expectedStatus: http.StatusTooManyRequests,
			expectFailures: true,
		},
		{
			name: "Disabled",
			cfg: config.RateLimitConfig{
				RequestsPerSecond: 1,
				Burst:             1,
				Enabled:           false,
			},
			requests:       10,
			expectedStatus: http.StatusOK,
			expectFailures: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(RateLimitMiddleware(tt.cfg))
			r.GET("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			failed := false
			for i := 0; i < tt.requests; i++ {
				w := httptest.NewRecorder()
				req, _ := http.NewRequest("GET", "/test", nil)
				r.ServeHTTP(w, req)

				if w.Code == http.StatusTooManyRequests {
					failed = true
				}
			}

			if tt.expectFailures {
				assert.True(t, failed, "Expected some requests to be rate limited")
			} else {
				assert.False(t, failed, "Expected no requests to be rate limited")
			}
		})
	}
}
