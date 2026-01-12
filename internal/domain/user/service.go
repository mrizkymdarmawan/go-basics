// Package user contains the user domain logic.
// This file implements the Service layer (also called Use Case layer).
//
// THE SERVICE LAYER:
// The service layer contains business logic and orchestrates operations.
// It sits between the handler (HTTP) and repository (database) layers.
//
// Responsibilities:
// - Implement business rules (e.g., "password must be 8+ characters")
// - Coordinate multiple operations (e.g., check email exists, then create user)
// - Handle transactions when needed
// - Transform data between layers
//
// What it should NOT do:
// - Know about HTTP (no http.Request, http.ResponseWriter)
// - Know about specific database implementations
// - Handle presentation logic (JSON encoding, etc.)
package user

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// Password constraints as constants.
// Using constants instead of magic numbers makes code self-documenting.
const (
	// MinPasswordLength is the minimum allowed password length.
	// NIST guidelines recommend at least 8 characters.
	MinPasswordLength = 8

	// MaxPasswordLength is the maximum allowed password length.
	// bcrypt truncates at 72 bytes, so we enforce this limit.
	MaxPasswordLength = 72

	// bcryptCost determines how computationally expensive hashing is.
	// Higher = more secure but slower. 10-12 is recommended for production.
	// Each increment doubles the computation time.
	bcryptCost = 12
)

// emailRegex is a simple regex for email validation.
// Note: Perfect email validation is complex. This catches obvious errors.
// For production, consider sending a verification email instead.
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// Service implements business logic for user operations.
// It depends on the Repository interface, NOT a concrete implementation.
//
// WHY INTERFACE DEPENDENCY?
// This is called "Dependency Inversion Principle" (the D in SOLID).
// Benefits:
// 1. Easy testing - you can use a mock repository
// 2. Flexibility - swap MySQL for PostgreSQL without changing this code
// 3. Decoupling - service doesn't know or care about database details
type Service struct {
	repo Repository // Interface, not concrete type
}

// NewService creates a new user service.
// This is a constructor function - a common Go pattern.
// We pass dependencies as parameters (Dependency Injection).
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Create registers a new user in the system.
// It validates input, hashes the password, and stores the user.
//
// Parameters:
//   - ctx: Context for cancellation and deadlines
//   - email: The user's email address
//   - password: The plain-text password (will be hashed)
//
// Returns:
//   - The created user (with ID populated)
//   - An error if validation fails or email exists
func (s *Service) Create(ctx context.Context, email, password string) (*User, error) {
	// Step 1: Validate input
	// Always validate at the service layer, even if the handler validates too.
	// This ensures business rules are enforced regardless of how the service is called.
	if err := validateEmail(email); err != nil {
		return nil, err
	}
	if err := validatePassword(password); err != nil {
		return nil, err
	}

	// Step 2: Check if email already exists
	// We do this BEFORE hashing to avoid wasting CPU on duplicate requests.
	existing, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		// Wrap errors with context using fmt.Errorf and %w.
		// This preserves the original error while adding context.
		return nil, fmt.Errorf("checking email existence: %w", err)
	}
	if existing != nil {
		return nil, ErrEmailExists
	}

	// Step 3: Hash the password
	// NEVER store plain-text passwords! Always hash them.
	hashedPassword, err := hashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	// Step 4: Create the user entity
	user := &User{
		Email:        strings.ToLower(email), // Normalize email to lowercase
		PasswordHash: hashedPassword,
	}

	// Step 5: Persist to database
	if err := s.repo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}

	return user, nil
}

// GetByID retrieves a user by their ID.
// Returns ErrNotFound if the user doesn't exist.
func (s *Service) GetByID(ctx context.Context, id uint64) (*User, error) {
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("finding user by id: %w", err)
	}
	if user == nil {
		// Return domain error instead of nil.
		// This makes error handling explicit for callers.
		return nil, ErrNotFound
	}
	return user, nil
}

// Update modifies an existing user's information.
// Currently supports email and password updates.
func (s *Service) Update(ctx context.Context, id uint64, email, password string) (*User, error) {
	// Step 1: Verify user exists
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("finding user: %w", err)
	}
	if user == nil {
		return nil, ErrNotFound
	}

	// Step 2: Validate and update email if provided
	if email != "" && email != user.Email {
		if err := validateEmail(email); err != nil {
			return nil, err
		}
		// Check if new email is taken by another user
		existing, err := s.repo.FindByEmail(ctx, email)
		if err != nil {
			return nil, fmt.Errorf("checking email: %w", err)
		}
		if existing != nil && existing.ID != id {
			return nil, ErrEmailExists
		}
		user.Email = strings.ToLower(email)
	}

	// Step 3: Validate and update password if provided
	if password != "" {
		if err := validatePassword(password); err != nil {
			return nil, err
		}
		hashedPassword, err := hashPassword(password)
		if err != nil {
			return nil, fmt.Errorf("hashing password: %w", err)
		}
		user.PasswordHash = hashedPassword
	}

	// Step 4: Persist changes
	if err := s.repo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("updating user: %w", err)
	}

	return user, nil
}

// Delete removes a user from the system.
// Uses soft delete - sets deleted_at instead of removing the row.
func (s *Service) Delete(ctx context.Context, id uint64) error {
	// Verify user exists before deleting
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("finding user: %w", err)
	}
	if user == nil {
		return ErrNotFound
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}
	return nil
}

// Authenticate verifies user credentials and returns the user if valid.
// This is used for login functionality.
//
// SECURITY NOTES:
// - We return the same error for "user not found" and "wrong password"
//   to prevent attackers from discovering valid emails.
// - We use constant-time comparison (bcrypt does this internally).
func (s *Service) Authenticate(ctx context.Context, email, password string) (*User, error) {
	// Find user by email
	user, err := s.repo.FindByEmail(ctx, strings.ToLower(email))
	if err != nil {
		return nil, fmt.Errorf("finding user: %w", err)
	}
	if user == nil {
		// User not found - return generic error
		return nil, ErrInvalidCredentials
	}

	// Compare password with hash
	// bcrypt.CompareHashAndPassword is constant-time to prevent timing attacks.
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		// Wrong password - return same generic error
		return nil, ErrInvalidCredentials
	}

	return user, nil
}

// validateEmail checks if the email format is valid.
func validateEmail(email string) error {
	if email == "" {
		return &ValidationError{Field: "email", Message: "email is required"}
	}
	if !emailRegex.MatchString(email) {
		return ErrInvalidEmail
	}
	return nil
}

// validatePassword checks if the password meets requirements.
func validatePassword(password string) error {
	if password == "" {
		return &ValidationError{Field: "password", Message: "password is required"}
	}
	if len(password) < MinPasswordLength {
		return ErrPasswordTooShort
	}
	if len(password) > MaxPasswordLength {
		return ErrPasswordTooLong
	}
	return nil
}

// hashPassword creates a bcrypt hash of the password.
//
// HOW BCRYPT WORKS:
// 1. Generates a random salt (no need to store separately)
// 2. Combines salt + password + cost factor
// 3. Runs the expensive Blowfish cipher multiple times (2^cost)
// 4. Returns a string containing: algorithm, cost, salt, and hash
//
// The result looks like: $2a$12$LQv3c1yqBw...
// Where $2a$ = algorithm, $12$ = cost, rest = salt+hash
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
