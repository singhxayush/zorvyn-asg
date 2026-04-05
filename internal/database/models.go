package database

import "time"

// User represents a user in the system
type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`      
	Role         string    `json:"role"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// FinancialRecord represents an income or expense entry
type FinancialRecord struct {
	ID          int        `json:"id"`
	Amount      float64    `json:"amount"`
	Type        string     `json:"type"`
	Category    string     `json:"category"`
	Date        time.Time  `json:"date"`
	Description string     `json:"description,omitempty"`
	CreatedBy   int        `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

// DashboardSummary represents the aggregated analytics data
type DashboardSummary struct {
	TotalIncome  float64            `json:"total_income"`
	TotalExpense float64            `json:"total_expense"`
	NetBalance   float64            `json:"net_balance"`
	ByCategory   map[string]float64 `json:"by_category"`
}
