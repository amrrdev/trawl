package repository

import (
	"context"

	"github.com/amrrdev/trawl/services/auth/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository interface {
	CreateUser(ctx context.Context, arg db.CreateUserParams) (db.CreateUserRow, error)
	UpdateUserProfile(ctx context.Context, arg db.UpdateUserProfileParams) (db.UpdateUserProfileRow, error)
	UpdateUserEmail(ctx context.Context, arg db.UpdateUserEmailParams) (db.UpdateUserEmailRow, error)
	UpdateUserPassword(ctx context.Context, arg db.UpdateUserPasswordParams) error

	GetUserByEmail(ctx context.Context, email string) (db.User, error)
	GetUserByID(ctx context.Context, userID pgtype.UUID) (db.User, error)
	GetUserForValidation(ctx context.Context, userID pgtype.UUID) (db.GetUserForValidationRow, error)
	CheckUserExists(ctx context.Context, email string) (bool, error)

	DeactivateUser(ctx context.Context, userID pgtype.UUID) error
	ReactivateUser(ctx context.Context, userID pgtype.UUID) error
	BulkDeactivateUsers(ctx context.Context, userIDs []pgtype.UUID) error

	ListUsers(ctx context.Context, arg db.ListUsersParams) ([]db.ListUsersRow, error)
	ListActiveUsers(ctx context.Context, arg db.ListActiveUsersParams) ([]db.ListActiveUsersRow, error)
	CountUsers(ctx context.Context, arg db.CountUsersParams) (int64, error)
	GetUsersByDateRange(ctx context.Context, arg db.GetUsersByDateRangeParams) ([]db.GetUsersByDateRangeRow, error)

	GetUserStats(ctx context.Context) (db.GetUserStatsRow, error)

	GetDuplicateEmails(ctx context.Context) ([]db.GetDuplicateEmailsRow, error)

	AdminHardDeleteUser(ctx context.Context, userID pgtype.UUID) error
}

type userRepository struct {
	queries *db.Queries
}

func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &userRepository{
		queries: db.New(pool),
	}
}

func (r *userRepository) GetUserByEmail(ctx context.Context, email string) (db.User, error) {
	return r.queries.GetUserByEmail(ctx, email)
}

func (r *userRepository) GetUserByID(ctx context.Context, userID pgtype.UUID) (db.User, error) {
	return r.queries.GetUserByID(ctx, userID)
}

func (r *userRepository) GetUserForValidation(ctx context.Context, userID pgtype.UUID) (db.GetUserForValidationRow, error) {
	return r.queries.GetUserForValidation(ctx, userID)
}

func (r *userRepository) CheckUserExists(ctx context.Context, email string) (bool, error) {
	return r.queries.CheckUserExists(ctx, email)
}

func (r *userRepository) CreateUser(ctx context.Context, arg db.CreateUserParams) (db.CreateUserRow, error) {
	return r.queries.CreateUser(ctx, arg)
}

func (r *userRepository) UpdateUserProfile(ctx context.Context, arg db.UpdateUserProfileParams) (db.UpdateUserProfileRow, error) {
	return r.queries.UpdateUserProfile(ctx, arg)
}

func (r *userRepository) UpdateUserEmail(ctx context.Context, arg db.UpdateUserEmailParams) (db.UpdateUserEmailRow, error) {
	return r.queries.UpdateUserEmail(ctx, arg)
}

func (r *userRepository) UpdateUserPassword(ctx context.Context, arg db.UpdateUserPasswordParams) error {
	return r.queries.UpdateUserPassword(ctx, arg)
}

func (r *userRepository) DeactivateUser(ctx context.Context, userID pgtype.UUID) error {
	return r.queries.DeactivateUser(ctx, userID)
}

func (r *userRepository) ReactivateUser(ctx context.Context, userID pgtype.UUID) error {
	return r.queries.ReactivateUser(ctx, userID)
}

func (r *userRepository) BulkDeactivateUsers(ctx context.Context, userIDs []pgtype.UUID) error {
	return r.queries.BulkDeactivateUsers(ctx, userIDs)
}

func (r *userRepository) ListUsers(ctx context.Context, arg db.ListUsersParams) ([]db.ListUsersRow, error) {
	return r.queries.ListUsers(ctx, arg)
}

func (r *userRepository) ListActiveUsers(ctx context.Context, arg db.ListActiveUsersParams) ([]db.ListActiveUsersRow, error) {
	return r.queries.ListActiveUsers(ctx, arg)
}

func (r *userRepository) CountUsers(ctx context.Context, arg db.CountUsersParams) (int64, error) {
	return r.queries.CountUsers(ctx, arg)
}

func (r *userRepository) GetUsersByDateRange(ctx context.Context, arg db.GetUsersByDateRangeParams) ([]db.GetUsersByDateRangeRow, error) {
	return r.queries.GetUsersByDateRange(ctx, arg)
}

func (r *userRepository) GetUserStats(ctx context.Context) (db.GetUserStatsRow, error) {
	return r.queries.GetUserStats(ctx)
}

func (r *userRepository) GetDuplicateEmails(ctx context.Context) ([]db.GetDuplicateEmailsRow, error) {
	return r.queries.GetDuplicateEmails(ctx)
}

func (r *userRepository) AdminHardDeleteUser(ctx context.Context, userID pgtype.UUID) error {
	return r.queries.AdminHardDeleteUser(ctx, userID)
}
