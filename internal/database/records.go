package database

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var ErrRecordNotFound = errors.New("financial record not found")

// CreateRecord inserts a new financial transaction
func (s *service) CreateRecord(ctx context.Context, record *FinancialRecord) error {
	query := `
		INSERT INTO financial_records (amount, type, category, date, description, created_by)
		VALUES (?, ?, ?, ?, ?, ?)
		RETURNING id, created_at, updated_at
	`

	// Ensure the date is formatted correctly for SQLite
	return s.db.QueryRowContext(ctx, query,
		record.Amount,
		record.Type,
		record.Category,
		record.Date,
		record.Description,
		record.CreatedBy,
	).Scan(&record.ID, &record.CreatedAt, &record.UpdatedAt)
}

// GetRecordByID fetches a single record, ensuring it hasn't been soft-deleted
func (s *service) GetRecordByID(ctx context.Context, id int) (*FinancialRecord, error) {
	query := `
		SELECT id, amount, type, category, date, description, created_by, created_at, updated_at 
		FROM financial_records 
		WHERE id = ? AND deleted_at IS NULL
	`

	var record FinancialRecord
	var desc sql.NullString // Handle potential nulls in description

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&record.ID, &record.Amount, &record.Type, &record.Category,
		&record.Date, &desc, &record.CreatedBy, &record.CreatedAt, &record.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRecordNotFound
		}
		return nil, err
	}

	if desc.Valid {
		record.Description = desc.String
	}

	return &record, nil
}

// UpdateRecord updates an existing record
func (s *service) UpdateRecord(ctx context.Context, record *FinancialRecord) error {
	query := `
		UPDATE financial_records 
		SET amount = ?, type = ?, category = ?, date = ?, description = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND deleted_at IS NULL
	`

	result, err := s.db.ExecContext(ctx, query,
		record.Amount, record.Type, record.Category, record.Date, record.Description, record.ID,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrRecordNotFound
	}

	return nil
}

// DeleteRecord performs a SOFT DELETE by setting the deleted_at timestamp
func (s *service) DeleteRecord(ctx context.Context, id int) error {
	query := `
		UPDATE financial_records 
		SET deleted_at = ? 
		WHERE id = ? AND deleted_at IS NULL
	`

	result, err := s.db.ExecContext(ctx, query, time.Now(), id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrRecordNotFound
	}

	return nil
}

// ListRecords fetches records with basic optional filtering
// ListRecords fetches records with optional filtering and pagination
func (s *service) ListRecords(ctx context.Context, recordType, category string, limit int, offset int) ([]FinancialRecord, error) {
	query := `
		SELECT id, amount, type, category, date, description, created_by, created_at, updated_at 
		FROM financial_records 
		WHERE deleted_at IS NULL
	`

	var args []any

	if recordType != "" {
		query += " AND type = ?"
		args = append(args, recordType)
	}
	if category != "" {
		query += " AND category = ?"
		args = append(args, category)
	}

	// Add Order, Limit, and Offset
	query += " ORDER BY date DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []FinancialRecord
	for rows.Next() {
		var r FinancialRecord
		var desc sql.NullString

		if err := rows.Scan(&r.ID, &r.Amount, &r.Type, &r.Category, &r.Date, &desc, &r.CreatedBy, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		if desc.Valid {
			r.Description = desc.String
		}
		records = append(records, r)
	}

	return records, rows.Err()
}

// GetDashboardSummary aggregates financial data (Total Income, Total Expense, by Category)
func (s *service) GetDashboardSummary(ctx context.Context) (*DashboardSummary, error) {
	summary := &DashboardSummary{
		ByCategory: make(map[string]float64),
	}

	// 1. Get totals by type (Income vs Expense)
	typeQuery := `
		SELECT type, SUM(amount) 
		FROM financial_records 
		WHERE deleted_at IS NULL 
		GROUP BY type
	`

	rows, err := s.db.QueryContext(ctx, typeQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var recordType string
		var total float64
		if err := rows.Scan(&recordType, &total); err != nil {
			return nil, err
		}
		switch recordType {
		case "income":
			summary.TotalIncome = total
		case "expense":
			summary.TotalExpense = total
		}
	}
	summary.NetBalance = summary.TotalIncome - summary.TotalExpense

	// 2. Get totals by category
	categoryQuery := `
		SELECT category, SUM(amount) 
		FROM financial_records 
		WHERE deleted_at IS NULL 
		GROUP BY category
	`
	catRows, err := s.db.QueryContext(ctx, categoryQuery)
	if err != nil {
		return nil, err
	}
	defer catRows.Close()

	for catRows.Next() {
		var category string
		var total float64
		if err := catRows.Scan(&category, &total); err != nil {
			return nil, err
		}
		summary.ByCategory[category] = total
	}

	return summary, nil
}
