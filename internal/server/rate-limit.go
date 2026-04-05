package server

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// IPlimiter holds a map of IP addresses and their respective rate limiters
type IPlimiter struct {
	ips map[string]*rate.Limiter
	mu  sync.Mutex
	r   rate.Limit
	b   int
}

// NewIPRateLimiter defines the rate (r) and burst (b)
func NewIPRateLimiter(r rate.Limit, b int) *IPlimiter {
	return &IPlimiter{
		ips: make(map[string]*rate.Limiter),
		r:   r,
		b:   b,
	}
}

// GetLimiter returns the rate limiter for the provided IP address
func (i *IPlimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	limiter, exists := i.ips[ip]
	if !exists {
		limiter = rate.NewLimiter(i.r, i.b)
		i.ips[ip] = limiter
	}

	return limiter
}

// RateLimitMiddleware intercepts requests and checks the rate limit
func RateLimitMiddleware(limit rate.Limit, burst int) gin.HandlerFunc {
	i := NewIPRateLimiter(limit, burst)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		limiter := i.GetLimiter(ip)

		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests. Please try again later.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
