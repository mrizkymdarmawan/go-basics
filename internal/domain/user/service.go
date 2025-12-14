package user

import (
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

var (
	// Authentication
	ErrInvalidCredentials = errors.New("invalid email or password")

	// User management
	ErrUserNotFound = errors.New("user not found")
	ErrEmailExists  = errors.New("email already exists")
)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Service) Login(ctx context.Context, email, password string) (*User, error) {
	u, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, ErrInvalidCredentials
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(u.PasswordHash),
		[]byte(password),
	)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	return u, nil
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, u *User) error {
	existing, _ := s.repo.FindByEmail(ctx, u.Email)
	if existing != nil {
		return ErrEmailExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(u.PasswordHash),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return err
	}

	u.PasswordHash = string(hashedPassword)

	return s.repo.Create(ctx, u)
}

func (s *Service) GetByID(ctx context.Context, id uint64) (*User, error) {
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (s *Service) Update(ctx context.Context, u *User) error {
	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(u.PasswordHash),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return err
	}

	u.PasswordHash = string(hashedPassword)

	return s.repo.Update(ctx, u)
}

func (s *Service) Delete(ctx context.Context, id uint64) error {
	return s.repo.Delete(ctx, id)
}
