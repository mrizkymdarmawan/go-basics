// Package auth provides authentication functionality using JWT (JSON Web Tokens).
//
// JWT BASICS:
// A JWT consists of three parts separated by dots: header.payload.signature
// 1. Header: Contains the token type (JWT) and signing algorithm (HS256)
// 2. Payload: Contains claims (data) like user ID, email, expiration time
// 3. Signature: Ensures the token hasn't been tampered with
//
// Example JWT: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Sentinel errors for JWT operations.
// Using sentinel errors allows callers to check error types with errors.Is().
var (
	// ErrInvalidToken is returned when the token is malformed or signature is invalid.
	ErrInvalidToken = errors.New("invalid token")

	// ErrExpiredToken is returned when the token has expired.
	ErrExpiredToken = errors.New("token has expired")
)

// Claims represents the JWT payload (the data stored in the token).
// It embeds jwt.RegisteredClaims which provides standard JWT fields.
//
// IMPORTANT: Only store non-sensitive data in claims!
// JWTs are encoded (base64), NOT encrypted. Anyone can decode and read them.
// Never put passwords, credit cards, or sensitive data in claims.
type Claims struct {
	// UserID is the unique identifier of the authenticated user.
	// We store this to identify the user on subsequent requests.
	UserID uint64 `json:"user_id"`

	// Email is included for convenience so we don't need a database
	// lookup for every request that needs the user's email.
	Email string `json:"email"`

	// RegisteredClaims contains standard JWT fields like:
	// - ExpiresAt: When the token expires
	// - IssuedAt: When the token was created
	// - Issuer: Who created the token
	jwt.RegisteredClaims
}

// JWTManager handles JWT token operations.
// We use a struct instead of package-level functions because:
// 1. It allows dependency injection (easier testing)
// 2. Configuration is explicit, not hidden in global variables
// 3. You could have multiple JWTManagers with different settings
type JWTManager struct {
	secret   []byte        // The secret key used for signing tokens
	duration time.Duration // How long tokens are valid
	issuer   string        // Identifies who created the token
}

// NewJWTManager creates a new JWT manager.
// Parameters:
//   - secret: The signing key. Should be at least 32 bytes for HS256.
//   - duration: How long tokens should be valid (e.g., 15*time.Minute)
//   - issuer: A string identifying your application
func NewJWTManager(secret string, duration time.Duration, issuer string) *JWTManager {
	return &JWTManager{
		secret:   []byte(secret), // Convert string to bytes for signing
		duration: duration,
		issuer:   issuer,
	}
}

// GenerateToken creates a new JWT token for a user.
// This is called after successful login to give the user a token
// they can use for subsequent authenticated requests.
//
// Returns:
//   - The signed JWT token string
//   - An error if signing fails
func (m *JWTManager) GenerateToken(userID uint64, email string) (string, error) {
	// Create the claims (payload data)
	now := time.Now()
	claims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			// ExpiresAt: After this time, the token is invalid.
			// Short expiration (15-30 min) limits damage if token is stolen.
			ExpiresAt: jwt.NewNumericDate(now.Add(m.duration)),

			// IssuedAt: When the token was created.
			// Useful for debugging and audit logs.
			IssuedAt: jwt.NewNumericDate(now),

			// NotBefore: Token is not valid before this time.
			// We set it to now, but you could delay activation if needed.
			NotBefore: jwt.NewNumericDate(now),

			// Issuer: Identifies who created the token.
			// Useful when multiple services issue tokens.
			Issuer: m.issuer,
		},
	}

	// Create the token with our claims
	// jwt.SigningMethodHS256 uses HMAC-SHA256 for signing.
	// This is symmetric encryption - same key for signing and verifying.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token with our secret key.
	// This creates the third part of the JWT (the signature).
	tokenString, err := token.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken verifies a JWT token and extracts the claims.
// This is called on every authenticated request to verify the user.
//
// Parameters:
//   - tokenString: The JWT token from the Authorization header
//
// Returns:
//   - The claims if the token is valid
//   - An error if the token is invalid or expired
func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	// Parse and validate the token
	token, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{}, // Empty claims struct to be populated
		func(token *jwt.Token) (interface{}, error) {
			// This function is called during parsing to provide the key.
			// We also verify the signing method is what we expect.

			// SECURITY: Always check the signing algorithm!
			// Attackers might try to change "alg" to "none" or "HS256" when
			// you expect "RS256". This is a common JWT vulnerability.
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}

			return m.secret, nil
		},
	)

	// Handle parsing errors
	if err != nil {
		// Check if it's an expiration error specifically
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	// Extract and return the claims
	// Type assertion: convert interface{} to our Claims type
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
