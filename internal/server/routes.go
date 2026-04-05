package server

import (
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func (s *Server) RegisterRoutes() http.Handler {
	r := gin.Default()

	// Rate Limit middleware: 5 requests per second, Burst: 10 requests
	r.Use(RateLimitMiddleware(rate.Limit(5), 10))

	// Cors middleware
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	// Route grouping
	v1 := r.Group("/api/v1")
	{
		// Public Auth Routes
		v1.POST("/login", s.loginHandler)
		v1.POST("/refresh", s.refreshHandler)
		v1.POST("/logout", s.logoutHandler)

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
