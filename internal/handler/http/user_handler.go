// Package http contains HTTP handlers for the API.
// This is the Delivery layer (also called Presentation or Interface layer).
//
// THE HANDLER LAYER:
// Handlers are responsible for:
// 1. Parsing HTTP requests (JSON, query params, path params)
// 2. Validating input format (not business rules)
// 3. Calling the appropriate service methods
// 4. Formatting HTTP responses (JSON, status codes)
//
// Handlers should NOT contain business logic!
// Business logic belongs in the service layer.
package http

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"go-basics/internal/auth"
	"go-basics/internal/domain/user"
)

// Request DTOs (Data Transfer Objects)
// DTOs are structures used to transfer data between layers.
// They decouple the API format from the domain model.

// registerRequest is the expected JSON body for user registration.
// struct tags like `json:"email"` map JSON keys to struct fields.
type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// loginRequest is the expected JSON body for user login.
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// updateRequest is the expected JSON body for user updates.
// Both fields are optional - only non-empty fields are updated.
type updateRequest struct {
	Email    string `json:"email,omitempty"`
	Password string `json:"password,omitempty"`
}

// Response DTOs
// We use separate response types to control what data is exposed.
// NEVER expose password hashes or internal fields in responses!

// userResponse is returned for single user operations.
type userResponse struct {
	ID    uint64 `json:"id"`
	Email string `json:"email"`
}

// loginResponse includes the JWT token for authentication.
type loginResponse struct {
	Token string       `json:"token"`
	User  userResponse `json:"user"`
}

// errorResponse provides consistent error formatting.
type errorResponse struct {
	Error string `json:"error"`
}

// UserHandler handles HTTP requests for user operations.
// It depends on the user service and JWT manager for authentication.
type UserHandler struct {
	service    *user.Service  // Business logic layer
	jwtManager *auth.JWTManager // For generating tokens on login
}

// NewUserHandler creates a new user handler.
// This is dependency injection - we pass dependencies as parameters.
func NewUserHandler(service *user.Service, jwtManager *auth.JWTManager) *UserHandler {
	return &UserHandler{
		service:    service,
		jwtManager: jwtManager,
	}
}

// RegisterRoutes sets up HTTP routes for user operations.
//
// GO 1.22+ ROUTING ENHANCEMENTS:
// Before Go 1.22, we had to check r.Method manually:
//
//	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
//	    switch r.Method {
//	    case http.MethodGet: ...
//	    case http.MethodPost: ...
//	    }
//	})
//
// Now we can specify the method in the pattern:
//
//	mux.HandleFunc("GET /users", handler.list)
//	mux.HandleFunc("POST /users", handler.create)
//
// Path parameters are also supported:
//
//	mux.HandleFunc("GET /users/{id}", handler.get)
//
// Access path params with r.PathValue("id")
func (h *UserHandler) RegisterRoutes(mux *http.ServeMux, authMiddleware *auth.Middleware) {
	// Public routes - no authentication required
	mux.HandleFunc("POST /register", h.register)
	mux.HandleFunc("POST /login", h.login)

	// Protected routes - require valid JWT token
	// We wrap handlers with authMiddleware.AuthenticateFunc()
	mux.HandleFunc("GET /users/{id}", authMiddleware.AuthenticateFunc(h.get))
	mux.HandleFunc("PUT /users/{id}", authMiddleware.AuthenticateFunc(h.update))
	mux.HandleFunc("DELETE /users/{id}", authMiddleware.AuthenticateFunc(h.delete))

	// Example of a protected route that gets current user info
	mux.HandleFunc("GET /me", authMiddleware.AuthenticateFunc(h.me))
}

// register handles POST /register
// Creates a new user account.
func (h *UserHandler) register(w http.ResponseWriter, r *http.Request) {
	// Step 1: Parse JSON request body
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Client sent invalid JSON
		writeError(w, http.StatusBadRequest, "invalid JSON format")
		return
	}

	// Step 2: Call service to create user
	// The service handles validation and business logic
	newUser, err := h.service.Create(r.Context(), req.Email, req.Password)
	if err != nil {
		// Map domain errors to HTTP status codes
		handleServiceError(w, err)
		return
	}

	// Step 3: Return success response
	// 201 Created is the correct status for successful resource creation
	writeJSON(w, http.StatusCreated, userResponse{
		ID:    newUser.ID,
		Email: newUser.Email,
	})
}

// login handles POST /login
// Authenticates a user and returns a JWT token.
func (h *UserHandler) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON format")
		return
	}

	// Authenticate user (verify email and password)
	authenticatedUser, err := h.service.Authenticate(r.Context(), req.Email, req.Password)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	// Generate JWT token for the authenticated user
	token, err := h.jwtManager.GenerateToken(authenticatedUser.ID, authenticatedUser.Email)
	if err != nil {
		// Token generation shouldn't fail normally - log for debugging
		log.Printf("failed to generate token: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	// Return token and user info
	writeJSON(w, http.StatusOK, loginResponse{
		Token: token,
		User: userResponse{
			ID:    authenticatedUser.ID,
			Email: authenticatedUser.Email,
		},
	})
}

// get handles GET /users/{id}
// Retrieves a user by ID. Requires authentication.
func (h *UserHandler) get(w http.ResponseWriter, r *http.Request) {
	// GO 1.22+: Extract path parameter using PathValue
	// Before 1.22, you'd have to manually parse the URL path
	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	// Get user from service
	foundUser, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, userResponse{
		ID:    foundUser.ID,
		Email: foundUser.Email,
	})
}

// update handles PUT /users/{id}
// Updates a user's information. Requires authentication.
func (h *UserHandler) update(w http.ResponseWriter, r *http.Request) {
	// Parse path parameter
	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	// AUTHORIZATION CHECK:
	// Users should only be able to update their own profile.
	// Get the authenticated user's ID from the JWT claims in context.
	claims, ok := auth.GetClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if claims.UserID != id {
		// User is trying to update someone else's profile
		writeError(w, http.StatusForbidden, "you can only update your own profile")
		return
	}

	// Parse request body
	var req updateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON format")
		return
	}

	// Update user
	updatedUser, err := h.service.Update(r.Context(), id, req.Email, req.Password)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	// 200 OK for successful update
	writeJSON(w, http.StatusOK, userResponse{
		ID:    updatedUser.ID,
		Email: updatedUser.Email,
	})
}

// delete handles DELETE /users/{id}
// Soft-deletes a user. Requires authentication.
func (h *UserHandler) delete(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	// Authorization: users can only delete themselves
	claims, ok := auth.GetClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if claims.UserID != id {
		writeError(w, http.StatusForbidden, "you can only delete your own account")
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		handleServiceError(w, err)
		return
	}

	// 204 No Content is standard for successful DELETE
	w.WriteHeader(http.StatusNoContent)
}

// me handles GET /me
// Returns the currently authenticated user's information.
// This is a convenience endpoint so users don't need to know their ID.
func (h *UserHandler) me(w http.ResponseWriter, r *http.Request) {
	// Get user ID from JWT claims
	claims, ok := auth.GetClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Fetch full user data
	currentUser, err := h.service.GetByID(r.Context(), claims.UserID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, userResponse{
		ID:    currentUser.ID,
		Email: currentUser.Email,
	})
}

// handleServiceError maps domain errors to HTTP responses.
// This centralizes error handling and ensures consistent responses.
//
// WHY USE errors.Is()?
// errors.Is() checks if an error IS or WRAPS a specific error.
// This works even if the service wrapped the error with context:
//
//	return fmt.Errorf("finding user: %w", user.ErrNotFound)
//
// errors.Is(err, user.ErrNotFound) will still return true.
func handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, user.ErrNotFound):
		writeError(w, http.StatusNotFound, "user not found")
	case errors.Is(err, user.ErrEmailExists):
		writeError(w, http.StatusConflict, "email already exists")
	case errors.Is(err, user.ErrInvalidCredentials):
		writeError(w, http.StatusUnauthorized, "invalid email or password")
	case errors.Is(err, user.ErrInvalidEmail):
		writeError(w, http.StatusBadRequest, "invalid email format")
	case errors.Is(err, user.ErrPasswordTooShort):
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
	case errors.Is(err, user.ErrPasswordTooLong):
		writeError(w, http.StatusBadRequest, "password must be at most 72 characters")
	default:
		// Check if it's a validation error
		var validationErr *user.ValidationError
		if errors.As(err, &validationErr) {
			writeError(w, http.StatusBadRequest, validationErr.Error())
			return
		}
		// Unknown error - log it but don't expose details to client
		log.Printf("internal error: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

// writeJSON writes a JSON response with the given status code.
// This is a helper function to reduce code duplication.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	// Set Content-Type header BEFORE WriteHeader
	// Headers must be set before writing the body
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	// Encode data to JSON and write to response
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// This shouldn't happen with valid data, but log it if it does
		log.Printf("failed to encode JSON response: %v", err)
	}
}

// writeError writes an error response in JSON format.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}
