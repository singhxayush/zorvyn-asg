package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"zorvyn-asg/internal/database"

	"github.com/gin-gonic/gin"
)

func setupRecordTestServer(t *testing.T) (*Server, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	mockDB := NewMockDB()
	server := &Server{port: 8080, db: mockDB}
	router := gin.New()

	// Add auth middleware
	authMiddleware := func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("userRole", "admin")
	}
	router.Use(authMiddleware)

	return server, router
}

func TestCreateRecordHandler_Success(t *testing.T) {
	server, router := setupRecordTestServer(t)
	router.POST("/records", server.createRecordHandler)

	requestBody := CreateRecordRequest{
		Amount:      1000.00,
		Type:        "income",
		Category:    "salary",
		Date:        time.Now(),
		Description: "Monthly salary",
	}
	bodyBytes, _ := json.Marshal(requestBody)

	req, _ := http.NewRequest("POST", "/records", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var record database.FinancialRecord
	json.Unmarshal(w.Body.Bytes(), &record)

	if record.ID == 0 {
		t.Error("Expected record ID to be set")
	}
	if record.Amount != 1000.00 {
		t.Errorf("Expected amount 1000.00, got %f", record.Amount)
	}
	if record.CreatedBy != 1 {
		t.Errorf("Expected created_by to be 1, got %d", record.CreatedBy)
	}
}

func TestCreateRecordHandler_InvalidAmount(t *testing.T) {
	server, router := setupRecordTestServer(t)
	router.POST("/records", server.createRecordHandler)

	requestBody := CreateRecordRequest{
		Amount:   -100.00, // Negative amount
		Type:     "income",
		Category: "salary",
		Date:     time.Now(),
	}
	bodyBytes, _ := json.Marshal(requestBody)

	req, _ := http.NewRequest("POST", "/records", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestCreateRecordHandler_InvalidType(t *testing.T) {
	server, router := setupRecordTestServer(t)
	router.POST("/records", server.createRecordHandler)

	requestBody := CreateRecordRequest{
		Amount:   100.00,
		Type:     "invalid_type",
		Category: "salary",
		Date:     time.Now(),
	}
	bodyBytes, _ := json.Marshal(requestBody)

	req, _ := http.NewRequest("POST", "/records", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestGetRecordHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockDB := NewMockDB()

	// Create a test record
	record := &database.FinancialRecord{
		Amount:      500.00,
		Type:        "expense",
		Category:    "groceries",
		Date:        time.Now(),
		Description: "Weekly groceries",
		CreatedBy:   1,
	}
	mockDB.CreateRecord(context.Background(), record)

	server := &Server{port: 8080, db: mockDB}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("userRole", "analyst")
	})
	router.GET("/records/:id", server.getRecordHandler)

	req, _ := http.NewRequest("GET", "/records/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var retrievedRecord database.FinancialRecord
	json.Unmarshal(w.Body.Bytes(), &retrievedRecord)

	if retrievedRecord.ID != 1 {
		t.Errorf("Expected record ID 1, got %d", retrievedRecord.ID)
	}
	if retrievedRecord.Amount != 500.00 {
		t.Errorf("Expected amount 500.00, got %f", retrievedRecord.Amount)
	}
}

func TestGetRecordHandler_NotFound(t *testing.T) {
	server, router := setupRecordTestServer(t)
	router.GET("/records/:id", server.getRecordHandler)

	req, _ := http.NewRequest("GET", "/records/999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestGetRecordHandler_InvalidID(t *testing.T) {
	server, router := setupRecordTestServer(t)
	router.GET("/records/:id", server.getRecordHandler)

	req, _ := http.NewRequest("GET", "/records/invalid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestListRecordsHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockDB := NewMockDB()

	// Create test records
	for i := 0; i < 5; i++ {
		record := &database.FinancialRecord{
			Amount:    float64(100 * (i + 1)),
			Type:      "income",
			Category:  "salary",
			Date:      time.Now(),
			CreatedBy: 1,
		}
		mockDB.CreateRecord(context.Background(), record)
	}

	server := &Server{port: 8080, db: mockDB}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("userRole", "analyst")
	})
	router.GET("/records", server.listRecordsHandler)

	req, _ := http.NewRequest("GET", "/records", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	data, ok := response["data"].([]interface{})
	if !ok {
		t.Error("Expected 'data' field in response")
	}
	if len(data) != 5 {
		t.Errorf("Expected 5 records, got %d", len(data))
	}
}

func TestListRecordsHandler_FilterByType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockDB := NewMockDB()

	// Create records of different types
	for i := 0; i < 3; i++ {
		record := &database.FinancialRecord{
			Amount:    100.00,
			Type:      "income",
			Category:  "salary",
			Date:      time.Now(),
			CreatedBy: 1,
		}
		mockDB.CreateRecord(context.Background(), record)
	}

	for i := 0; i < 2; i++ {
		record := &database.FinancialRecord{
			Amount:    50.00,
			Type:      "expense",
			Category:  "food",
			Date:      time.Now(),
			CreatedBy: 1,
		}
		mockDB.CreateRecord(context.Background(), record)
	}

	server := &Server{port: 8080, db: mockDB}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("userRole", "analyst")
	})
	router.GET("/records", server.listRecordsHandler)

	req, _ := http.NewRequest("GET", "/records?type=income", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].([]interface{})
	if len(data) != 3 {
		t.Errorf("Expected 3 income records, got %d", len(data))
	}
}

func TestListRecordsHandler_Pagination(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockDB := NewMockDB()

	// Create 100 records
	for i := 0; i < 100; i++ {
		record := &database.FinancialRecord{
			Amount:    float64(100),
			Type:      "income",
			Category:  "salary",
			Date:      time.Now(),
			CreatedBy: 1,
		}
		mockDB.CreateRecord(context.Background(), record)
	}

	server := &Server{port: 8080, db: mockDB}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("userRole", "analyst")
	})
	router.GET("/records", server.listRecordsHandler)

	req, _ := http.NewRequest("GET", "/records?page=2", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	meta := response["meta"].(map[string]interface{})
	if meta["page"] != float64(2) {
		t.Errorf("Expected page 2, got %v", meta["page"])
	}
}

func TestUpdateRecordHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockDB := NewMockDB()

	// Create a test record
	record := &database.FinancialRecord{
		Amount:    500.00,
		Type:      "expense",
		Category:  "groceries",
		Date:      time.Now(),
		CreatedBy: 1,
	}
	mockDB.CreateRecord(context.Background(), record)

	server := &Server{port: 8080, db: mockDB}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("userRole", "admin")
	})
	router.PUT("/records/:id", server.updateRecordHandler)

	requestBody := CreateRecordRequest{
		Amount:      750.00,
		Type:        "expense",
		Category:    "utilities",
		Date:        time.Now(),
		Description: "Updated description",
	}
	bodyBytes, _ := json.Marshal(requestBody)

	req, _ := http.NewRequest("PUT", "/records/1", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestUpdateRecordHandler_NotFound(t *testing.T) {
	server, router := setupRecordTestServer(t)
	router.PUT("/records/:id", server.updateRecordHandler)

	requestBody := CreateRecordRequest{
		Amount:   100.00,
		Type:     "income",
		Category: "salary",
		Date:     time.Now(),
	}
	bodyBytes, _ := json.Marshal(requestBody)

	req, _ := http.NewRequest("PUT", "/records/999", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestDeleteRecordHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockDB := NewMockDB()

	// Create a test record
	record := &database.FinancialRecord{
		Amount:    500.00,
		Type:      "expense",
		Category:  "groceries",
		Date:      time.Now(),
		CreatedBy: 1,
	}
	mockDB.CreateRecord(context.Background(), record)

	server := &Server{port: 8080, db: mockDB}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("userRole", "admin")
	})
	router.DELETE("/records/:id", server.deleteRecordHandler)

	req, _ := http.NewRequest("DELETE", "/records/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestDeleteRecordHandler_NotFound(t *testing.T) {
	server, router := setupRecordTestServer(t)
	router.DELETE("/records/:id", server.deleteRecordHandler)

	req, _ := http.NewRequest("DELETE", "/records/999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestDashboardSummaryHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockDB := NewMockDB()

	// Create income records
	incomeRecord := &database.FinancialRecord{
		Amount:    1000.00,
		Type:      "income",
		Category:  "salary",
		Date:      time.Now(),
		CreatedBy: 1,
	}
	mockDB.CreateRecord(context.Background(), incomeRecord)

	// Create expense records
	expenseRecord := &database.FinancialRecord{
		Amount:    200.00,
		Type:      "expense",
		Category:  "food",
		Date:      time.Now(),
		CreatedBy: 1,
	}
	mockDB.CreateRecord(context.Background(), expenseRecord)

	server := &Server{port: 8080, db: mockDB}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("userRole", "viewer")
	})
	router.GET("/dashboard", server.dashboardSummaryHandler)

	req, _ := http.NewRequest("GET", "/dashboard", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var summary database.DashboardSummary
	json.Unmarshal(w.Body.Bytes(), &summary)

	if summary.TotalIncome != 1000.00 {
		t.Errorf("Expected total income 1000.00, got %f", summary.TotalIncome)
	}
	if summary.TotalExpense != 200.00 {
		t.Errorf("Expected total expense 200.00, got %f", summary.TotalExpense)
	}
	if summary.NetBalance != 800.00 {
		t.Errorf("Expected net balance 800.00, got %f", summary.NetBalance)
	}
}
