package database

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

// ErrUserNotFound is a custom error for missing users
var ErrUserNotFound = errors.New("user not found")

// ErrDuplicateEmail is a custom error for duplicate constraints
var ErrDuplicateEmail = errors.New("email or username already exists")

// CreateUser inserts a new user into the database
func (s *service) CreateUser(ctx context.Context, user *User) error {
	query := `
		INSERT INTO users (username, email, password_hash, role, status)
		VALUES (?, ?, ?, ?, ?)
		RETURNING id, created_at, updated_at
	`

	err := s.db.QueryRowContext(ctx, query,
		user.Username,
		user.Email,
		user.PasswordHash,
		user.Role,
		user.Status,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		// Catch SQLite UNIQUE constraint violations
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return ErrDuplicateEmail
		}
		return err
	}

	return nil
}

// GetUserByEmail fetches a user by their email address
func (s *service) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, username, email, password_hash, role, status, created_at, updated_at
		FROM users
		WHERE email = ?
	`

	var user User
	err := s.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}

func (s *service) ListUsers(ctx context.Context) ([]User, error) {
	query := `SELECT id, username, email, role, status, created_at, updated_at FROM users`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Role, &u.Status, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (s *service) UpdateUserRole(ctx context.Context, id int, role string) error {
	query := `UPDATE users SET role = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := s.db.ExecContext(ctx, query, role, id)
	return err
}

func (s *service) UpdateUserStatus(ctx context.Context, id int, status string) error {
	query := `UPDATE users SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := s.db.ExecContext(ctx, query, status, id)
	return err
}

func (s *service) DeleteUser(ctx context.Context, id int) error {
	// Hard delete for users, cascading will remove their records (based on our DB schema)
	query := `DELETE FROM users WHERE id = ?`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}
