// Package mysql implements data persistence using MySQL database.
// This is the Repository layer (also called Infrastructure or Data Access layer).
//
// THE REPOSITORY PATTERN:
// The repository pattern abstracts data storage from the domain.
// It provides a collection-like interface for accessing domain objects.
//
// Benefits:
// 1. Domain logic doesn't know about database details
// 2. Easy to swap databases (MySQL → PostgreSQL)
// 3. Easy to mock for testing
// 4. Centralizes query logic
//
// This package implements the interface defined in domain/user/repository.go
package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"go-basics/internal/domain/user"
)

// UserRepository implements user.Repository interface for MySQL.
// It wraps a *sql.DB connection pool.
//
// WHY USE *sql.DB?
// *sql.DB is a connection pool, not a single connection. It:
// - Manages multiple connections automatically
// - Handles connection reuse and cleanup
// - Is safe for concurrent use from multiple goroutines
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new repository instance.
// This is a constructor - it returns the interface type, not the struct.
// Returning the interface makes it clear what methods are available.
func NewUserRepository(db *sql.DB) user.Repository {
	return &UserRepository{db: db}
}

// Create inserts a new user into the database.
// It sets the user's ID to the auto-generated value after insert.
//
// IMPORTANT: The password should already be hashed by the service layer!
// The repository should never see plain-text passwords.
func (r *UserRepository) Create(ctx context.Context, u *user.User) error {
	// SQL query with placeholders (?)
	// MySQL uses ? for placeholders; PostgreSQL uses $1, $2, etc.
	//
	// WHY PLACEHOLDERS?
	// Never concatenate user input into SQL strings!
	// That causes SQL injection vulnerabilities.
	// Placeholders (parameterized queries) prevent SQL injection.
	query := `
		INSERT INTO users (email, password_hash, created_at, updated_at)
		VALUES (?, ?, NOW(), NOW())
	`

	// ExecContext executes a query that doesn't return rows (INSERT, UPDATE, DELETE).
	// We pass ctx to support cancellation and timeouts.
	result, err := r.db.ExecContext(ctx, query, u.Email, u.PasswordHash)
	if err != nil {
		return fmt.Errorf("executing insert: %w", err)
	}

	// Get the auto-generated ID from MySQL.
	// This only works with AUTO_INCREMENT columns.
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting last insert id: %w", err)
	}

	// Update the user struct with the new ID.
	// This is a common pattern - the caller gets the ID without another query.
	u.ID = uint64(id)
	return nil
}

// FindByID retrieves a user by their primary key.
// Returns nil, nil if the user doesn't exist (not an error).
//
// This pattern (nil, nil for not found) is debatable.
// Alternative: return a domain error like user.ErrNotFound.
// We use nil, nil here so the service layer decides how to handle "not found".
func (r *UserRepository) FindByID(ctx context.Context, id uint64) (*user.User, error) {
	// Query with soft-delete filter.
	// "deleted_at IS NULL" excludes soft-deleted records.
	query := `
		SELECT id, email, password_hash, created_at, updated_at, deleted_at
		FROM users
		WHERE id = ? AND deleted_at IS NULL
	`

	// QueryRowContext returns a single row.
	// Use QueryContext (without "Row") for multiple rows.
	row := r.db.QueryRowContext(ctx, query, id)

	// Scan the row into a user struct.
	// The order of arguments must match the SELECT column order.
	var u user.User
	err := row.Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.CreatedAt,
		&u.UpdatedAt,
		&u.DeletedAt, // Nullable column - use *time.Time
	)

	// Handle "not found" case.
	// sql.ErrNoRows is returned when the query returns zero rows.
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil // Not found is not an error
	}
	if err != nil {
		return nil, fmt.Errorf("scanning user: %w", err)
	}

	return &u, nil
}

// FindByEmail retrieves a user by their email address.
// Used for login and checking if email already exists.
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	query := `
		SELECT id, email, password_hash, created_at, updated_at, deleted_at
		FROM users
		WHERE email = ? AND deleted_at IS NULL
	`

	row := r.db.QueryRowContext(ctx, query, email)

	var u user.User
	err := row.Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.CreatedAt,
		&u.UpdatedAt,
		&u.DeletedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scanning user: %w", err)
	}

	return &u, nil
}

// Update modifies an existing user's data.
// Only updates email and password_hash; created_at stays unchanged.
//
// NOTE: This updates all fields every time.
// For partial updates, you'd need a different approach (e.g., update map).
func (r *UserRepository) Update(ctx context.Context, u *user.User) error {
	query := `
		UPDATE users
		SET email = ?, password_hash = ?, updated_at = NOW()
		WHERE id = ? AND deleted_at IS NULL
	`

	// ExecContext returns a sql.Result with RowsAffected().
	// We could check if any rows were updated to detect "not found".
	result, err := r.db.ExecContext(ctx, query, u.Email, u.PasswordHash, u.ID)
	if err != nil {
		return fmt.Errorf("executing update: %w", err)
	}

	// Optional: Check if any rows were affected.
	// If no rows affected, the user might not exist.
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("getting rows affected: %w", err)
	}
	if rowsAffected == 0 {
		// Could return an error here, but we let the service handle this
		// by calling FindByID first.
		return nil
	}

	return nil
}

// Delete performs a soft delete by setting deleted_at.
//
// SOFT DELETE vs HARD DELETE:
// - Hard delete: DELETE FROM users WHERE id = ?
//   * Data is gone forever
//   * Faster, saves space
//
// - Soft delete: UPDATE users SET deleted_at = NOW() WHERE id = ?
//   * Data is preserved but hidden
//   * Can be "undeleted" if needed
//   * Required for audit trails and compliance
//   * All queries must include "deleted_at IS NULL"
func (r *UserRepository) Delete(ctx context.Context, id uint64) error {
	query := `
		UPDATE users
		SET deleted_at = NOW()
		WHERE id = ? AND deleted_at IS NULL
	`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("executing soft delete: %w", err)
	}
	return nil
}
