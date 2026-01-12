// Package user contains the user domain logic.
// This file defines all error types for the user domain.
//
// WHY SEPARATE ERRORS FILE?
// 1. Single source of truth for all user-related errors
// 2. Easy to find and maintain error definitions
// 3. Prevents circular imports when other packages need these errors
// 4. Makes error handling consistent across the application
package user

import "errors"

// Sentinel errors are predefined errors that can be checked with errors.Is().
// They are called "sentinel" because they stand guard - you compare against them.
//
// Usage:
//
//	if errors.Is(err, user.ErrNotFound) {
//	    // handle not found case
//	}
var (
	// ErrNotFound is returned when a user cannot be found in the database.
	// Use this instead of returning nil to make error handling explicit.
	ErrNotFound = errors.New("user not found")

	// ErrEmailExists is returned when trying to create a user with an email
	// that already exists in the database.
	ErrEmailExists = errors.New("email already exists")

	// ErrInvalidCredentials is returned when login fails due to wrong
	// email or password. We use a single error for both cases to prevent
	// attackers from knowing which field was wrong (security best practice).
	ErrInvalidCredentials = errors.New("invalid email or password")

	// ErrInvalidEmail is returned when the email format is invalid.
	ErrInvalidEmail = errors.New("invalid email format")

	// ErrPasswordTooShort is returned when the password doesn't meet
	// minimum length requirements.
	ErrPasswordTooShort = errors.New("password must be at least 8 characters")

	// ErrPasswordTooLong is returned when the password exceeds bcrypt's limit.
	// bcrypt truncates passwords longer than 72 bytes, so we reject them.
	ErrPasswordTooLong = errors.New("password must be at most 72 characters")
)

// ValidationError represents a validation error with field-specific information.
// This is useful for returning detailed error messages to API clients.
//
// Example usage:
//
//	return &ValidationError{Field: "email", Message: "invalid format"}
type ValidationError struct {
	Field   string // The field that failed validation
	Message string // Human-readable error message
}

// Error implements the error interface.
// This allows ValidationError to be used anywhere an error is expected.
func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}
