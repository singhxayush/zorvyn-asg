package server

import (
	"context"
	"time"
	"zorvyn-asg/internal/database"
)

// MockDB implements database.Service interface for testing
type MockDB struct {
	users     map[int]*database.User
	records   map[int]*database.FinancialRecord
	nextID    int
	nextRecID int
}

// NewMockDB creates a new mock database
func NewMockDB() *MockDB {
	return &MockDB{
		users:     make(map[int]*database.User),
		records:   make(map[int]*database.FinancialRecord),
		nextID:    1,
		nextRecID: 1,
	}
}

// CreateUser creates a test user
func (m *MockDB) CreateUser(ctx context.Context, user *database.User) error {
	// Check for duplicate email
	for _, u := range m.users {
		if u.Email == user.Email {
			return database.ErrDuplicateEmail
		}
	}
	user.ID = m.nextID
	m.nextID++
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	m.users[user.ID] = user
	return nil
}

// GetUserByEmail retrieves a user by email
func (m *MockDB) GetUserByEmail(ctx context.Context, email string) (*database.User, error) {
	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, database.ErrUserNotFound
}

// ListUsers lists all users
func (m *MockDB) ListUsers(ctx context.Context) ([]database.User, error) {
	var users []database.User
	for _, user := range m.users {
		users = append(users, *user)
	}
	return users, nil
}

// UpdateUserRole updates a user's role
func (m *MockDB) UpdateUserRole(ctx context.Context, id int, role string) error {
	user, ok := m.users[id]
	if !ok {
		return database.ErrUserNotFound
	}
	user.Role = role
	user.UpdatedAt = time.Now()
	return nil
}

// UpdateUserStatus updates a user's status
func (m *MockDB) UpdateUserStatus(ctx context.Context, id int, status string) error {
	user, ok := m.users[id]
	if !ok {
		return database.ErrUserNotFound
	}
	user.Status = status
	user.UpdatedAt = time.Now()
	return nil
}

// DeleteUser deletes a user
func (m *MockDB) DeleteUser(ctx context.Context, id int) error {
	if _, ok := m.users[id]; !ok {
		return database.ErrUserNotFound
	}
	delete(m.users, id)
	return nil
}

// CreateRecord creates a financial record
func (m *MockDB) CreateRecord(ctx context.Context, record *database.FinancialRecord) error {
	record.ID = m.nextRecID
	m.nextRecID++
	record.CreatedAt = time.Now()
	record.UpdatedAt = time.Now()
	m.records[record.ID] = record
	return nil
}

// GetRecordByID retrieves a record by ID
func (m *MockDB) GetRecordByID(ctx context.Context, id int) (*database.FinancialRecord, error) {
	record, ok := m.records[id]
	if !ok {
		return nil, database.ErrRecordNotFound
	}
	return record, nil
}

// ListRecords lists financial records
func (m *MockDB) ListRecords(ctx context.Context, recordType, category string, limit, offset int) ([]database.FinancialRecord, error) {
	var records []database.FinancialRecord
	count := 0
	for _, record := range m.records {
		if record.DeletedAt != nil {
			continue
		}
		if recordType != "" && record.Type != recordType {
			continue
		}
		if category != "" && record.Category != category {
			continue
		}
		if count >= offset && len(records) < limit {
			records = append(records, *record)
		}
		count++
	}
	return records, nil
}

// UpdateRecord updates a financial record
func (m *MockDB) UpdateRecord(ctx context.Context, record *database.FinancialRecord) error {
	existing, ok := m.records[record.ID]
	if !ok {
		return database.ErrRecordNotFound
	}
	existing.Amount = record.Amount
	existing.Type = record.Type
	existing.Category = record.Category
	existing.Date = record.Date
	existing.Description = record.Description
	existing.UpdatedAt = time.Now()
	return nil
}

// DeleteRecord soft-deletes a record
func (m *MockDB) DeleteRecord(ctx context.Context, id int) error {
	record, ok := m.records[id]
	if !ok {
		return database.ErrRecordNotFound
	}
	now := time.Now()
	record.DeletedAt = &now
	return nil
}

// GetDashboardSummary returns dashboard summary
func (m *MockDB) GetDashboardSummary(ctx context.Context) (*database.DashboardSummary, error) {
	summary := &database.DashboardSummary{
		ByCategory: make(map[string]float64),
	}
	for _, record := range m.records {
		if record.DeletedAt != nil {
			continue
		}
		if record.Type == "income" {
			summary.TotalIncome += record.Amount
		} else {
			summary.TotalExpense += record.Amount
		}
		summary.ByCategory[record.Category] += record.Amount
	}
	summary.NetBalance = summary.TotalIncome - summary.TotalExpense
	return summary, nil
}

// Health returns health status
func (m *MockDB) Health() map[string]string {
	return map[string]string{
		"status": "up",
	}
}

// Close closes the database connection (no-op for mock)
func (m *MockDB) Close() error {
	return nil
}
