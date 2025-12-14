package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"go-basics/internal/domain/user"
)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Insert / update request
type userRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Response structure
type userResponse struct {
	ID    uint64 `json:"id"`
	Email string `json:"email"`
}

// Handler struct
type UserHandler struct {
	service *user.Service
}

func NewUserHandler(service *user.Service) *UserHandler {
	return &UserHandler{service: service}
}

// Routing registration
func (h *UserHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/users", h.users)
	mux.HandleFunc("/users/", h.userByID)
	mux.HandleFunc("/login", h.login)
}

// Post /users (insert)
func (h *UserHandler) users(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.create(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// Implementation of POST /users
func (h *UserHandler) create(w http.ResponseWriter, r *http.Request) {
	var req userRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		http.Error(w, "email and password required", http.StatusBadRequest)
		return
	}

	u := &user.User{
		Email:        req.Email,
		PasswordHash: req.Password, // hashing added later (auth step)
	}

	if err := h.service.Create(r.Context(), u); err != nil {
		if err == user.ErrEmailExists {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	resp := userResponse{ID: u.ID, Email: u.Email}
	writeJSON(w, http.StatusCreated, resp)
}

// GET / PUT / DELETE /users/{id}
func (h *UserHandler) userByID(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/users/")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.get(w, r, id)
	case http.MethodPut:
		h.update(w, r, id)
	case http.MethodDelete:
		h.delete(w, r, id)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// GET
func (h *UserHandler) get(w http.ResponseWriter, r *http.Request, id uint64) {
	u, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if err == user.ErrUserNotFound {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	resp := userResponse{ID: u.ID, Email: u.Email}
	writeJSON(w, http.StatusOK, resp)
}

// PUT
func (h *UserHandler) update(w http.ResponseWriter, r *http.Request, id uint64) {
	var req userRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		http.Error(w, "email and password required", http.StatusBadRequest)
		return
	}

	u := &user.User{
		ID:           id,
		Email:        req.Email,
		PasswordHash: req.Password,
	}

	if err := h.service.Update(r.Context(), u); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DELETE
func (h *UserHandler) delete(w http.ResponseWriter, r *http.Request, id uint64) {
	if err := h.service.Delete(r.Context(), id); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// JSON helper
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// POST /login
func (h *UserHandler) login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		http.Error(w, "email and password required", http.StatusBadRequest)
		return
	}

	u, err := h.service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if err == user.ErrInvalidCredentials {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	resp := userResponse{
		ID:    u.ID,
		Email: u.Email,
	}

	writeJSON(w, http.StatusOK, resp)
}
