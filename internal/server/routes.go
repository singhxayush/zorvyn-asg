package server

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/time/rate"

	"zorvyn-asg/internal/database"
)

func (s *Server) RegisterRoutes() http.Handler {
	r := gin.Default()

	// Rate Limit: 5 requests per second, Burst: 10 requests
	r.Use(RateLimitMiddleware(rate.Limit(5), 10))

	// Cors middleware
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	r.GET("/", s.HelloWorldHandler)
	r.GET("/health", s.healthHandler)

	v1 := r.Group("/api/v1")
	{
		// Public Auth Routes
		v1.POST("/login", s.loginHandler)     // NEW LOGIN ROUTE
		v1.POST("/refresh", s.refreshHandler) // NEW
		v1.POST("/logout", s.logoutHandler)   // NEW

		// Protected Routes (Requires a valid JWT)
		protected := v1.Group("/")
		protected.Use(RequireAuth())
		{
			// Dashboard: Viewers, Analysts, and Admins can view this
			dashboardGroup := protected.Group("/")
			{
				dashboardGroup.Use(RequireRole("viewer", "analyst", "admin"))
				dashboardGroup.GET("/dashboard", s.dashboardSummaryHandler)
			}

			// Read Records: Only Analysts and Admins can view individual records
			readRecordsGroup := protected.Group("/")
			{
				readRecordsGroup.Use(RequireRole("analyst", "admin"))
				readRecordsGroup.GET("/records", s.listRecordsHandler)
				readRecordsGroup.GET("/records/:id", s.getRecordHandler)
			}

			// Write Records: ONLY Admins can create, update, or delete records
			writeRecordsGroup := protected.Group("/")
			{
				writeRecordsGroup.Use(RequireRole("admin"))
				writeRecordsGroup.POST("/records", s.createRecordHandler)
				writeRecordsGroup.PUT("/records/:id", s.updateRecordHandler)
				writeRecordsGroup.DELETE("/records/:id", s.deleteRecordHandler)
			}

			// User Management: ONLY ADMIN can manage user role and status
			adminGroup := protected.Group("/users")
			adminGroup.Use(RequireRole("admin"))
			{
				adminGroup.POST("/", s.createUserHandler)
				adminGroup.GET("/", s.listUsersHandler)
				adminGroup.PUT("/:id/role", s.updateUserRoleHandler)
				adminGroup.PUT("/:id/status", s.updateUserStatusHandler)
				adminGroup.DELETE("/:id", s.deleteUserHandler)
			}
		}
	}

	return r
}

// Handlers (HelloWorld, Health, Register remain unchanged)
func (s *Server) HelloWorldHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Hello World"})
}

func (s *Server) healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, s.db.Health())
}

// DTOs
type RegisterUserRequest struct {
	Username string `json:"username" binding:"required,min=3,max=30"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Role     string `json:"role" binding:"required,oneof=viewer analyst admin"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type CreateRecordRequest struct {
	Amount      float64   `json:"amount" binding:"required,gt=0"`
	Type        string    `json:"type" binding:"required,oneof=income expense"`
	Category    string    `json:"category" binding:"required"`
	Date        time.Time `json:"date" binding:"required"`
	Description string    `json:"description"`
}

type UpdateRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=viewer analyst admin"`
}

type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=active inactive"`
}

func (s *Server) createUserHandler(c *gin.Context) {
	var req RegisterUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input", "details": err.Error()})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process password"})
		return
	}

	user := &database.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		Role:         req.Role,
		Status:       "active",
	}

	// EXECUTE DB CALL
	if err := s.db.CreateUser(c.Request.Context(), user); err != nil {
		// CHECK FOR OUR CUSTOM ERROR
		if errors.Is(err, database.ErrDuplicateEmail) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}

		// Catch-all for other DB errors
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "User created successfully", "user_id": user.ID})
}

func (s *Server) listUsersHandler(c *gin.Context) {
	users, err := s.db.ListUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}
	c.JSON(http.StatusOK, users)
}

func (s *Server) updateUserRoleHandler(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role"})
		return
	}

	if err := s.db.UpdateUserRole(c.Request.Context(), id, req.Role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update role"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User role updated"})
}

func (s *Server) updateUserStatusHandler(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status"})
		return
	}

	if err := s.db.UpdateUserStatus(c.Request.Context(), id, req.Status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User status updated"})
}

func (s *Server) deleteUserHandler(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := s.db.DeleteUser(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

// NEW: Login Handler
func (s *Server) loginHandler(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	user, err := s.db.GetUserByEmail(c.Request.Context(), req.Email)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	// Block inactive users
	if user.Status == "inactive" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Account is inactive. Contact an administrator."})
		return
	}

	accessToken, refreshToken, err := GenerateTokens(user.ID, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate tokens"})
		return
	}

	// Set HTTP-Only Cookies. (name, value, maxAge, path, domain, secure, httpOnly)
	// secure is false for local development over HTTP. Set to true in production (HTTPS).
	c.SetCookie("access_token", accessToken, 15*60, "/", "", false, true)
	c.SetCookie("refresh_token", refreshToken, 7*24*60*60, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{"message": "Login successful", "user_id": user.ID, "role": user.Role})
}

func (s *Server) refreshHandler(c *gin.Context) {
	refreshTokenString, err := c.Cookie("refresh_token")
	if err != nil || refreshTokenString == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No refresh token provided"})
		return
	}

	token, err := jwt.ParseWithClaims(refreshTokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired refresh token"})
		return
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse claims"})
		return
	}

	// Generate a new access token (and optionally a new refresh token for rolling sessions)
	newAccessToken, _, err := GenerateTokens(claims.UserID, claims.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate new tokens"})
		return
	}

	c.SetCookie("access_token", newAccessToken, 15*60, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "Token refreshed successfully"})
}

func (s *Server) logoutHandler(c *gin.Context) {
	// Clear cookies by setting maxAge to -1
	c.SetCookie("access_token", "", -1, "/", "", false, true)
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// Financial Record Handlers (Unchanged, they now securely use the JWT context)
func (s *Server) createRecordHandler(c *gin.Context) {
	var req CreateRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input", "details": err.Error()})
		return
	}

	userID := c.GetInt("userID") // Now securely comes from the JWT

	record := &database.FinancialRecord{
		Amount:      req.Amount,
		Type:        req.Type,
		Category:    req.Category,
		Date:        req.Date,
		Description: req.Description,
		CreatedBy:   userID,
	}

	if err := s.db.CreateRecord(c.Request.Context(), record); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create record"})
		return
	}

	c.JSON(http.StatusCreated, record)
}

func (s *Server) getRecordHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid record ID"})
		return
	}

	record, err := s.db.GetRecordByID(c.Request.Context(), id)
	if err != nil {
		if err == database.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, record)
}

func (s *Server) listRecordsHandler(c *gin.Context) {
	recordType := c.Query("type")
	category := c.Query("category")

	// Pagination logic
	pageStr := c.DefaultQuery("page", "1")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit := 50
	offset := (page - 1) * limit

	records, err := s.db.ListRecords(c.Request.Context(), recordType, category, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch records"})
		return
	}

	if records == nil {
		records = []database.FinancialRecord{}
	}

	// Return data along with pagination metadata
	c.JSON(http.StatusOK, gin.H{
		"data": records,
		"meta": gin.H{
			"page":  page,
			"limit": limit,
			"count": len(records),
		},
	})
}

func (s *Server) updateRecordHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid record ID"})
		return
	}

	var req CreateRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input", "details": err.Error()})
		return
	}

	record := &database.FinancialRecord{
		ID:          id,
		Amount:      req.Amount,
		Type:        req.Type,
		Category:    req.Category,
		Date:        req.Date,
		Description: req.Description,
	}

	if err := s.db.UpdateRecord(c.Request.Context(), record); err != nil {
		if err == database.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update record"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Record updated successfully"})
}

func (s *Server) deleteRecordHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid record ID"})
		return
	}

	if err := s.db.DeleteRecord(c.Request.Context(), id); err != nil {
		if err == database.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete record"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Record deleted successfully"})
}

func (s *Server) dashboardSummaryHandler(c *gin.Context) {
	summary, err := s.db.GetDashboardSummary(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch dashboard summary"})
		return
	}
	c.JSON(http.StatusOK, summary)
}
