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
)

func setupUserTestServer(t *testing.T) (*Server, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	mockDB := NewMockDB()
	server := &Server{port: 8080, db: mockDB}
	router := gin.New()

	// Add auth middleware that sets userID and userRole for protected routes
	authMiddleware := func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("userRole", "admin")
	}

	router.Use(authMiddleware)

	return server, router
}

func TestCreateUserHandler_Success(t *testing.T) {
	server, router := setupUserTestServer(t)
	router.POST("/users", server.createUserHandler)

	requestBody := RegisterUserRequest{
		Username: "newuser",
		Email:    "newuser@example.com",
		Password: "password123",
		Role:     "analyst",
	}
	bodyBytes, _ := json.Marshal(requestBody)

	req, _ := http.NewRequest("POST", "/users", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if _, ok := response["user_id"]; !ok {
		t.Error("Expected user_id in response")
	}
}

func TestCreateUserHandler_InvalidEmail(t *testing.T) {
	server, router := setupUserTestServer(t)
	router.POST("/users", server.createUserHandler)

	requestBody := RegisterUserRequest{
		Username: "newuser",
		Email:    "invalid-email",
		Password: "password123",
		Role:     "analyst",
	}
	bodyBytes, _ := json.Marshal(requestBody)

	req, _ := http.NewRequest("POST", "/users", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestCreateUserHandler_ShortPassword(t *testing.T) {
	server, router := setupUserTestServer(t)
	router.POST("/users", server.createUserHandler)

	requestBody := RegisterUserRequest{
		Username: "newuser",
		Email:    "newuser@example.com",
		Password: "1234",
		Role:     "analyst",
	}
	bodyBytes, _ := json.Marshal(requestBody)

	req, _ := http.NewRequest("POST", "/users", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestCreateUserHandler_DuplicateEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockDB := NewMockDB()

	// Create an existing user
	existingUser := &database.User{
		Username:     "existing",
		Email:        "existing@example.com",
		PasswordHash: "hashed",
		Role:         "viewer",
		Status:       "active",
	}
	mockDB.CreateUser(context.Background(), existingUser)

	server := &Server{port: 8080, db: mockDB}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("userRole", "admin")
	})
	router.POST("/users", server.createUserHandler)

	requestBody := RegisterUserRequest{
		Username: "newuser",
		Email:    "existing@example.com", // Duplicate email
		Password: "password123",
		Role:     "analyst",
	}
	bodyBytes, _ := json.Marshal(requestBody)

	req, _ := http.NewRequest("POST", "/users", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status %d, got %d", http.StatusConflict, w.Code)
	}
}

func TestListUsersHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockDB := NewMockDB()

	// Create test users
	for i := 0; i < 3; i++ {
		user := &database.User{
			Username: "user" + string(rune(i)),
			Email:    "user" + string(rune(i)) + "@example.com",
			Role:     "viewer",
			Status:   "active",
		}
		mockDB.CreateUser(context.Background(), user)
	}

	server := &Server{port: 8080, db: mockDB}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("userRole", "admin")
	})
	router.GET("/users", server.listUsersHandler)

	req, _ := http.NewRequest("GET", "/users", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var users []database.User
	json.Unmarshal(w.Body.Bytes(), &users)

	if len(users) != 3 {
		t.Errorf("Expected 3 users, got %d", len(users))
	}
}

func TestListUsersHandler_Empty(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockDB := NewMockDB()
	server := &Server{port: 8080, db: mockDB}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("userRole", "admin")
	})
	router.GET("/users", server.listUsersHandler)

	req, _ := http.NewRequest("GET", "/users", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var users []database.User
	json.Unmarshal(w.Body.Bytes(), &users)

	if len(users) != 0 {
		t.Errorf("Expected 0 users, got %d", len(users))
	}
}

func TestUpdateUserRoleHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockDB := NewMockDB()

	// Create a test user
	user := &database.User{
		Username: "testuser",
		Email:    "test@example.com",
		Role:     "viewer",
		Status:   "active",
	}
	mockDB.CreateUser(context.Background(), user)

	server := &Server{port: 8080, db: mockDB}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("userRole", "admin")
	})
	router.PUT("/users/:id/role", server.updateUserRoleHandler)

	requestBody := UpdateRoleRequest{Role: "admin"}
	bodyBytes, _ := json.Marshal(requestBody)

	req, _ := http.NewRequest("PUT", "/users/1/role", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestUpdateUserRoleHandler_InvalidRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockDB := NewMockDB()
	server := &Server{port: 8080, db: mockDB}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("userRole", "admin")
	})
	router.PUT("/users/:id/role", server.updateUserRoleHandler)

	requestBody := UpdateRoleRequest{Role: "invalid_role"}
	bodyBytes, _ := json.Marshal(requestBody)

	req, _ := http.NewRequest("PUT", "/users/1/role", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestUpdateUserRoleHandler_UserNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockDB := NewMockDB()
	server := &Server{port: 8080, db: mockDB}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("userRole", "admin")
	})
	router.PUT("/users/:id/role", server.updateUserRoleHandler)

	requestBody := UpdateRoleRequest{Role: "admin"}
	bodyBytes, _ := json.Marshal(requestBody)

	req, _ := http.NewRequest("PUT", "/users/999/role", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestUpdateUserStatusHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockDB := NewMockDB()

	// Create a test user
	user := &database.User{
		Username: "testuser",
		Email:    "test@example.com",
		Role:     "viewer",
		Status:   "active",
	}
	mockDB.CreateUser(context.Background(), user)

	server := &Server{port: 8080, db: mockDB}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("userRole", "admin")
	})
	router.PUT("/users/:id/status", server.updateUserStatusHandler)

	requestBody := UpdateStatusRequest{Status: "inactive"}
	bodyBytes, _ := json.Marshal(requestBody)

	req, _ := http.NewRequest("PUT", "/users/1/status", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestUpdateUserStatusHandler_InvalidStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockDB := NewMockDB()
	server := &Server{port: 8080, db: mockDB}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("userRole", "admin")
	})
	router.PUT("/users/:id/status", server.updateUserStatusHandler)

	requestBody := UpdateStatusRequest{Status: "invalid_status"}
	bodyBytes, _ := json.Marshal(requestBody)

	req, _ := http.NewRequest("PUT", "/users/1/status", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestDeleteUserHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockDB := NewMockDB()

	// Create a test user
	user := &database.User{
		Username: "testuser",
		Email:    "test@example.com",
		Role:     "viewer",
		Status:   "active",
	}
	mockDB.CreateUser(context.Background(), user)

	server := &Server{port: 8080, db: mockDB}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userID", 2)
		c.Set("userRole", "admin")
	})
	router.DELETE("/users/:id", server.deleteUserHandler)

	req, _ := http.NewRequest("DELETE", "/users/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestDeleteUserHandler_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockDB := NewMockDB()
	server := &Server{port: 8080, db: mockDB}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("userRole", "admin")
	})
	router.DELETE("/users/:id", server.deleteUserHandler)

	req, _ := http.NewRequest("DELETE", "/users/999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}
