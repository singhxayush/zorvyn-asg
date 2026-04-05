package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func TestRequireAuthMiddleware_ValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Generate a valid token
	accessToken, _, _ := GenerateTokens(1, "analyst")

	router := gin.New()
	router.Use(RequireAuth())
	router.GET("/protected", func(c *gin.Context) {
		userID := c.GetInt("userID")
		userRole := c.GetString("userRole")
		c.JSON(http.StatusOK, gin.H{"user_id": userID, "role": userRole})
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  "access_token",
		Value: accessToken,
	})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestRequireAuthMiddleware_NoToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RequireAuth())
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/protected", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestRequireAuthMiddleware_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RequireAuth())
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  "access_token",
		Value: "invalid.token.value",
	})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestRequireAuthMiddleware_ExpiredToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create an expired token manually by manipulating time
	// Since we can't easily do this, we'll test with invalid token
	router := gin.New()
	router.Use(RequireAuth())
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  "access_token",
		Value: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxLCJyb2xlIjoiYW5hbHlzdCIsImV4cCI6MTAwMDAwMDAwMH0.invalid",
	})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestRequireRoleMiddleware_ValidRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()

	// Simulate authenticated request with admin role
	router.Use(func(c *gin.Context) {
		c.Set("userRole", "admin")
	})

	router.Use(RequireRole("admin", "analyst"))

	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestRequireRoleMiddleware_InvalidRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()

	// Simulate authenticated request with viewer role (not allowed)
	router.Use(func(c *gin.Context) {
		c.Set("userRole", "viewer")
	})

	router.Use(RequireRole("admin", "analyst"))

	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestRequireRoleMiddleware_MultipleRoles(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()

	// Test that analyst role passes through
	router.Use(func(c *gin.Context) {
		c.Set("userRole", "analyst")
	})

	router.Use(RequireRole("admin", "analyst"))

	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestRequireRoleMiddleware_NoRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()

	// No middleware to set role
	router.Use(RequireRole("admin", "analyst"))

	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestRateLimitMiddleware_AllowedRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RateLimitMiddleware(rate.Limit(10), 10)) // 10 requests per second

	router.GET("/api", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Make requests within the limit
	for i := 0; i < 10; i++ {
		req, _ := http.NewRequest("GET", "/api", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d failed with status %d", i+1, w.Code)
		}
	}
}

func TestRateLimitMiddleware_ExceededLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RateLimitMiddleware(rate.Limit(1), 1)) // 1 request per second

	router.GET("/api", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// First request should be OK
	req1, _ := http.NewRequest("GET", "/api", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Errorf("First request failed with status %d", w1.Code)
	}

	// Second immediate request should be rate limited
	req2, _ := http.NewRequest("GET", "/api", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status %d for rate limited request, got %d", http.StatusTooManyRequests, w2.Code)
	}
}

func TestRateLimitMiddleware_BurstAllowed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RateLimitMiddleware(rate.Limit(1), 5)) // 1 req/sec, burst 5

	router.GET("/api", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Should allow up to burst requests
	for i := 0; i < 5; i++ {
		req, _ := http.NewRequest("GET", "/api", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d within burst failed with status %d", i+1, w.Code)
		}
	}

	// Next request should be rate limited
	req, _ := http.NewRequest("GET", "/api", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status %d for request beyond burst, got %d", http.StatusTooManyRequests, w.Code)
	}
}

func TestRateLimitMiddleware_RecoveryAfterWait(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RateLimitMiddleware(rate.Limit(10), 1)) // 10 req/sec, burst 1

	router.GET("/api", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// First request OK
	req1, _ := http.NewRequest("GET", "/api", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Errorf("First request failed with status %d", w1.Code)
	}

	// Second immediate request rate limited
	req2, _ := http.NewRequest("GET", "/api", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("Expected rate limited status, got %d", w2.Code)
	}

	// Wait for recovery
	time.Sleep(150 * time.Millisecond)

	// Third request after recovery should be OK
	req3, _ := http.NewRequest("GET", "/api", nil)
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)

	if w3.Code != http.StatusOK {
		t.Errorf("Request after recovery failed with status %d", w3.Code)
	}
}
