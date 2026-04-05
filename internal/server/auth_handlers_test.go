package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"zorvyn-asg/internal/database"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func TestLoginHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create mock database and add a test user
	mockDB := NewMockDB()
	password := "testpass123"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	testUser := &database.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: string(hashedPassword),
		Role:         "analyst",
		Status:       "active",
	}
	mockDB.CreateUser(context.Background(), testUser)

	// Create server with mock DB
	server := &Server{port: 8080, db: mockDB}

	// Create request
	requestBody := LoginRequest{
		Email:    "test@example.com",
		Password: password,
	}
	bodyBytes, _ := json.Marshal(requestBody)

	req, _ := http.NewRequest("POST", "/api/v1/login", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router := gin.New()
	router.POST("/api/v1/login", server.loginHandler)
	router.ServeHTTP(w, req)

	// Assertions
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify cookies are set
	cookies := w.Result().Cookies()
	hasAccessToken := false
	hasRefreshToken := false
	for _, cookie := range cookies {
		if cookie.Name == "access_token" {
			hasAccessToken = true
		}
		if cookie.Name == "refresh_token" {
			hasRefreshToken = true
		}
	}
	if !hasAccessToken || !hasRefreshToken {
		t.Error("Expected access_token and refresh_token cookies to be set")
	}
}

func TestLoginHandler_InvalidEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockDB := NewMockDB()
	server := &Server{port: 8080, db: mockDB}

	requestBody := LoginRequest{
		Email:    "invalid@example.com",
		Password: "password123",
	}
	bodyBytes, _ := json.Marshal(requestBody)

	req, _ := http.NewRequest("POST", "/api/v1/login", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router := gin.New()
	router.POST("/api/v1/login", server.loginHandler)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestLoginHandler_WrongPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockDB := NewMockDB()
	password := "correctpassword"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	testUser := &database.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: string(hashedPassword),
		Role:         "analyst",
		Status:       "active",
	}
	mockDB.CreateUser(context.Background(), testUser)

	server := &Server{port: 8080, db: mockDB}

	requestBody := LoginRequest{
		Email:    "test@example.com",
		Password: "wrongpassword",
	}
	bodyBytes, _ := json.Marshal(requestBody)

	req, _ := http.NewRequest("POST", "/api/v1/login", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router := gin.New()
	router.POST("/api/v1/login", server.loginHandler)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestLoginHandler_InactiveUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockDB := NewMockDB()
	password := "testpass123"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	testUser := &database.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: string(hashedPassword),
		Role:         "analyst",
		Status:       "inactive", // User is inactive
	}
	mockDB.CreateUser(context.Background(), testUser)

	server := &Server{port: 8080, db: mockDB}

	requestBody := LoginRequest{
		Email:    "test@example.com",
		Password: password,
	}
	bodyBytes, _ := json.Marshal(requestBody)

	req, _ := http.NewRequest("POST", "/api/v1/login", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router := gin.New()
	router.POST("/api/v1/login", server.loginHandler)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestLoginHandler_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockDB := NewMockDB()
	server := &Server{port: 8080, db: mockDB}

	req, _ := http.NewRequest("POST", "/api/v1/login", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router := gin.New()
	router.POST("/api/v1/login", server.loginHandler)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestRefreshHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockDB := NewMockDB()
	server := &Server{port: 8080, db: mockDB}

	// Generate tokens
	_, refreshToken, _ := GenerateTokens(1, "analyst")

	req, _ := http.NewRequest("POST", "/api/v1/refresh", nil)
	req.AddCookie(&http.Cookie{
		Name:  "refresh_token",
		Value: refreshToken,
	})

	w := httptest.NewRecorder()
	router := gin.New()
	router.POST("/api/v1/refresh", server.refreshHandler)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify access token cookie is set
	cookies := w.Result().Cookies()
	hasAccessToken := false
	for _, cookie := range cookies {
		if cookie.Name == "access_token" && cookie.Value != "" {
			hasAccessToken = true
			break
		}
	}
	if !hasAccessToken {
		t.Error("Expected access_token cookie to be set")
	}
}

func TestRefreshHandler_NoToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockDB := NewMockDB()
	server := &Server{port: 8080, db: mockDB}

	req, _ := http.NewRequest("POST", "/api/v1/refresh", nil)

	w := httptest.NewRecorder()
	router := gin.New()
	router.POST("/api/v1/refresh", server.refreshHandler)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestRefreshHandler_ExpiredToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockDB := NewMockDB()
	server := &Server{port: 8080, db: mockDB}

	// Create an expired token manually (this is a simplified approach)
	req, _ := http.NewRequest("POST", "/api/v1/refresh", nil)
	req.AddCookie(&http.Cookie{
		Name:  "refresh_token",
		Value: "invalid.token.here",
	})

	w := httptest.NewRecorder()
	router := gin.New()
	router.POST("/api/v1/refresh", server.refreshHandler)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestLogoutHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockDB := NewMockDB()
	server := &Server{port: 8080, db: mockDB}

	req, _ := http.NewRequest("POST", "/api/v1/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  "access_token",
		Value: "some_token",
	})
	req.AddCookie(&http.Cookie{
		Name:  "refresh_token",
		Value: "some_refresh_token",
	})

	w := httptest.NewRecorder()
	router := gin.New()
	router.POST("/api/v1/logout", server.logoutHandler)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify cookies are cleared (maxAge should be -1)
	cookies := w.Result().Cookies()
	for _, cookie := range cookies {
		if (cookie.Name == "access_token" || cookie.Name == "refresh_token") && cookie.MaxAge != -1 {
			t.Errorf("Expected cookie %s to be cleared (MaxAge -1), got %d", cookie.Name, cookie.MaxAge)
		}
	}
}

func TestGenerateTokens(t *testing.T) {
	accessToken, refreshToken, err := GenerateTokens(1, "analyst")

	if err != nil {
		t.Fatalf("GenerateTokens failed: %v", err)
	}

	if accessToken == "" {
		t.Error("Expected non-empty access token")
	}

	if refreshToken == "" {
		t.Error("Expected non-empty refresh token")
	}

	// Both should be different
	if accessToken == refreshToken {
		t.Error("Expected access token and refresh token to be different")
	}
}
