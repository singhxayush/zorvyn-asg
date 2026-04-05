package server

import (
	"net/http"
	"strconv"
	"zorvyn-asg/internal/database"

	"github.com/gin-gonic/gin"
)

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
