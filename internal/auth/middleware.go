package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

// contextKey is a custom type for context keys.
// We use a custom type to avoid collisions with other packages.
//
// WHY NOT USE STRING DIRECTLY?
// If two packages both use "user" as a context key, they would collide.
// Using a custom type ensures our keys are unique to this package.
type contextKey string

// ClaimsKey is the context key for storing JWT claims.
// We export this so handlers can retrieve claims from the context.
const ClaimsKey contextKey = "claims"

// Middleware is an HTTP middleware that validates JWT tokens.
//
// WHAT IS MIDDLEWARE?
// Middleware is code that runs BEFORE your handler.
// It's like a security guard checking IDs before letting people into a building.
//
// Pattern: func(next http.Handler) http.Handler
// The middleware wraps around the next handler in the chain.
//
// Request flow:
// Client -> Middleware (check token) -> Handler (if token valid)
//                                    -> 401 response (if token invalid)
type Middleware struct {
	jwtManager *JWTManager
}

// NewMiddleware creates a new authentication middleware.
func NewMiddleware(jwtManager *JWTManager) *Middleware {
	return &Middleware{jwtManager: jwtManager}
}

// Authenticate is the middleware function that validates JWT tokens.
// It returns an http.Handler that wraps the next handler.
//
// Usage in routes:
//
//	mux.Handle("GET /protected", authMiddleware.Authenticate(protectedHandler))
func (m *Middleware) Authenticate(next http.Handler) http.Handler {
	// Return a new handler that wraps the original
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Step 1: Extract the token from the Authorization header
		// Expected format: "Bearer <token>"
		token, err := extractBearerToken(r)
		if err != nil {
			// No token provided - return 401 Unauthorized
			http.Error(w, "missing or invalid authorization header", http.StatusUnauthorized)
			return
		}

		// Step 2: Validate the token and extract claims
		claims, err := m.jwtManager.ValidateToken(token)
		if err != nil {
			// Token is invalid or expired
			if errors.Is(err, ErrExpiredToken) {
				http.Error(w, "token has expired", http.StatusUnauthorized)
				return
			}
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		// Step 3: Store claims in context for the handler to use
		// Context is how we pass request-scoped data through the handler chain.
		ctx := context.WithValue(r.Context(), ClaimsKey, claims)

		// Step 4: Call the next handler with the updated context
		// r.WithContext creates a new request with the modified context.
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// AuthenticateFunc is a convenience wrapper for http.HandlerFunc.
// Use this when your handler is a function, not an http.Handler.
//
// Usage:
//
//	mux.HandleFunc("GET /protected", authMiddleware.AuthenticateFunc(myHandlerFunc))
func (m *Middleware) AuthenticateFunc(next http.HandlerFunc) http.HandlerFunc {
	// Convert HandlerFunc to Handler, apply middleware, then convert back
	return m.Authenticate(next).ServeHTTP
}

// extractBearerToken extracts the JWT token from the Authorization header.
//
// Expected header format: "Authorization: Bearer <token>"
//
// WHY "BEARER"?
// "Bearer" is part of the OAuth 2.0 specification. It means
// "whoever bears (carries) this token is authorized".
// Other types exist (Basic, Digest) but Bearer is standard for JWT.
func extractBearerToken(r *http.Request) (string, error) {
	// Get the Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("authorization header is required")
	}

	// Split "Bearer <token>" into parts
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return "", errors.New("authorization header format must be 'Bearer <token>'")
	}

	return parts[1], nil
}

// GetClaimsFromContext retrieves JWT claims from the request context.
// Call this in your handlers to get information about the authenticated user.
//
// Usage in handler:
//
//	claims, ok := auth.GetClaimsFromContext(r.Context())
//	if !ok {
//	    // Handle error - should not happen if middleware is applied
//	}
//	userID := claims.UserID
func GetClaimsFromContext(ctx context.Context) (*Claims, bool) {
	// Type assertion: get the value and convert to *Claims
	claims, ok := ctx.Value(ClaimsKey).(*Claims)
	return claims, ok
}
