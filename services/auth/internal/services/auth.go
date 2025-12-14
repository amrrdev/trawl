package services

import (
	"context"
	"fmt"

	"github.com/amrrdev/trawl/services/auth/internal/db"
	"github.com/amrrdev/trawl/services/auth/internal/repository"
	"github.com/jackc/pgx/v5/pgtype"
)

type AuthService struct {
	repo           repository.UserRepository
	hashingService *HashingService
	jwtService     *JWTService
}

type RegisterResponse struct {
	UserID      string `json:"user_id"`
	Email       string `json:"email"`
	Name        string `json:"name"`
	AccessToken string `json:"access_token"`
}

type LoginResponse struct {
	UserID      string `json:"user_id"`
	Email       string `json:"email"`
	Name        string `json:"name"`
	AccessToken string `json:"access_token"`
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*LoginResponse, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		// Don't reveal if user exists or not for security
		return nil, fmt.Errorf("invalid email or password")
	}

	// Check if user is active
	if !user.IsActive.Bool {
		return nil, fmt.Errorf("account is deactivated")
	}

	isValid := s.hashingService.ComparePassword(user.Password, password)
	if !isValid {
		return nil, fmt.Errorf("invalid email or password")
	}

	accessToken, err := s.jwtService.GenerateAccessToken(user.UserID.String(), email)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	return &LoginResponse{
		UserID:      user.UserID.String(),
		Email:       user.Email,
		Name:        user.Name.String,
		AccessToken: accessToken,
	}, nil
}

func (s *AuthService) Register(ctx context.Context, name, email, password string) (*RegisterResponse, error) {
	isExists, err := s.repo.CheckUserExists(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to check user existence: %w", err)
	}
	if isExists {
		return nil, fmt.Errorf("user already exists")
	}

	hashedPassword, err := s.hashingService.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	newUser, err := s.repo.CreateUser(ctx, db.CreateUserParams{
		Name:     pgtype.Text{String: name, Valid: name != ""},
		Email:    email,
		Password: hashedPassword,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	accessToken, err := s.jwtService.GenerateAccessToken(newUser.UserID.String(), newUser.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	return &RegisterResponse{
		UserID:      newUser.UserID.String(),
		Email:       newUser.Email,
		Name:        newUser.Name.String,
		AccessToken: accessToken,
	}, nil
}
