package repository

import (
	"context"
	"errors"

	"github.com/7StaSH7/halfway-diploma/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var UserModule = fx.Module("user_repository",
	fx.Provide(NewUserRepository),
)

type UserRepository interface {
	CreateUser(ctx context.Context, user *model.User) error
	GetUserByUsername(ctx context.Context, username string) (*model.User, error)
	GetUserByID(ctx context.Context, id string) (*model.User, error)
	UpdateUserBalance(ctx context.Context, id string, balance int64) error
	UserExists(ctx context.Context, username string) (bool, error)
}

type userRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

type UserRepositoryParams struct {
	fx.In
	DB     *pgxpool.Pool
	Logger *zap.Logger
}

func NewUserRepository(p UserRepositoryParams) UserRepository {
	return &userRepository{
		db:     p.DB,
		logger: p.Logger,
	}
}

func (r *userRepository) CreateUser(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO users (id, username, password, balance)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at, updated_at
	`

	err := r.db.QueryRow(ctx, query, user.ID, user.Username, user.Password, user.Balance).
		Scan(&user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return err
	}

	return nil
}

func (r *userRepository) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	query := `
		SELECT id, username, password, balance, created_at, updated_at
		FROM users
		WHERE username = $1
	`

	var user model.User
	err := r.db.QueryRow(ctx, query, username).Scan(
		&user.ID,
		&user.Username,
		&user.Password,
		&user.Balance,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.Debug("user not found", zap.String("username", username))
			return nil, nil
		}
		r.logger.Error("failed to get user by username",
			zap.Error(err),
			zap.String("username", username))
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) GetUserByID(ctx context.Context, id string) (*model.User, error) {
	query := `
		SELECT id, username, password, balance, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user model.User
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Password,
		&user.Balance,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.Debug("user not found", zap.String("user_id", id))
			return nil, nil
		}
		r.logger.Error("failed to get user by ID",
			zap.Error(err),
			zap.String("user_id", id))
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) UpdateUserBalance(ctx context.Context, id string, balance int64) error {
	query := `
		UPDATE users
		SET balance = $1
		WHERE id = $2
	`

	result, err := r.db.Exec(ctx, query, balance, id)
	if err != nil {
		r.logger.Error("failed to update user's balance",
			zap.Error(err),
			zap.String("user_id", id),
			zap.Int64("balance", balance))
		return err
	}

	if result.RowsAffected() == 0 {
		r.logger.Warn("user to update balance not found", zap.String("user_id", id))
		return errors.New("user not found")
	}

	return nil
}

func (r *userRepository) UserExists(ctx context.Context, username string) (bool, error) {
	query := `
		SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)
	`

	var exists bool
	err := r.db.QueryRow(ctx, query, username).Scan(&exists)
	if err != nil {
		r.logger.Error("failed to check user existnce",
			zap.Error(err),
			zap.String("username", username))
		return false, err
	}

	return exists, nil
}
