package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"us.ztechai.zsms/backend/internal/database"
	"us.ztechai.zsms/backend/internal/domain"
)

type UserRepository struct {
	db *database.Client
}

func NewUserRepository(db *database.Client) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, email, passwordHash, fullName string) (*domain.User, error) {
	query := `
		INSERT INTO users (email, password_hash, full_name, status, created_at, updated_at)
		VALUES ($1, $2, $3, 'active', NOW(), NOW())
		RETURNING id, email, full_name, status, created_at, updated_at
	`

	user := &domain.User{}
	err := r.db.DB.QueryRowContext(ctx, query, email, passwordHash, fullName).Scan(
		&user.ID,
		&user.Email,
		&user.FullName,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, full_name, status, created_at, updated_at
		FROM users
		WHERE email = $1 AND deleted_at IS NULL
		LIMIT 1
	`

	user := &domain.User{}
	err := r.db.DB.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return user, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `
		SELECT id, email, full_name, status, created_at, updated_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
		LIMIT 1
	`

	user := &domain.User{}
	err := r.db.DB.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.FullName,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	return user, nil
}

// Count returns total active users.
func (r *UserRepository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.db.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE deleted_at IS NULL").Scan(&count)
	return count, err
}
