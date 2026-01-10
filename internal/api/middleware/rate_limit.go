package middleware

import (
	"net/http"
	"sync"


	"clortho/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"golang.org/x/time/rate"
)

type RateLimiter struct {
	ips     *expirable.LRU[string, *rate.Limiter]
	mu      sync.Mutex
	r       rate.Limit
	b       int
	enabled bool
}

func NewRateLimiter(cfg config.RateLimitConfig) *RateLimiter {
	// Use config values or fall back to defaults if 0/nil (though Load() handles defaults)
	size := cfg.CacheSize
	if size <= 0 {
		size = 5000
	}
	ttl := cfg.CacheTTL
	if ttl <= 0 {
		ttl = 0 // 0 means no expiry in some libs, but here we want it. Load() sets 1h.
	}

	return &RateLimiter{
		ips:     expirable.NewLRU[string, *rate.Limiter](size, nil, ttl),
		r:       rate.Limit(cfg.RequestsPerSecond),
		b:       cfg.Burst,
		enabled: cfg.Enabled,
	}
}

func (rl *RateLimiter) GetLimiter(ip string) *rate.Limiter {
	if limiter, ok := rl.ips.Get(ip); ok {
		return limiter
	}

	limiter := rate.NewLimiter(rl.r, rl.b)
	rl.ips.Add(ip, limiter)
	return limiter
}

func RateLimitMiddleware(cfg config.RateLimitConfig) gin.HandlerFunc {
	if !cfg.Enabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	rl := NewRateLimiter(cfg)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		limiter := rl.GetLimiter(ip)

		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
