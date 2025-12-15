package services

import (
	"context"
	"fmt"
	"strings"

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

func NewAuthService(repo repository.UserRepository, hashingService *HashingService, jwtService *JWTService) *AuthService {
	return &AuthService{
		repo:           repo,
		hashingService: hashingService,
		jwtService:     jwtService,
	}
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*LoginResponse, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	if !user.IsActive.Bool {
		return nil, fmt.Errorf("account is deactivated")
	}

	isValid := s.hashingService.ComparePassword(user.Password, password)
	if !isValid {
		return nil, fmt.Errorf("invalid credentials")
	}

	accessToken, err := s.jwtService.GenerateAccessToken(user.UserID.String(), email)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	name := ""
	if user.Name.Valid {
		name = user.Name.String
	}

	return &LoginResponse{
		UserID:      user.UserID.String(),
		Email:       user.Email,
		Name:        name,
		AccessToken: accessToken,
	}, nil
}

func (s *AuthService) Register(ctx context.Context, name, email, password string) (*RegisterResponse, error) {
	name = strings.TrimSpace(name)
	email = strings.ToLower(strings.TrimSpace(email))

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

	userName := ""
	if newUser.Name.Valid {
		userName = newUser.Name.String
	}

	return &RegisterResponse{
		UserID:      newUser.UserID.String(),
		Email:       newUser.Email,
		Name:        userName,
		AccessToken: accessToken,
	}, nil
}
