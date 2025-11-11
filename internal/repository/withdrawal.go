package repository

import (
	"context"

	"github.com/7StaSH7/halfway-diploma/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var WithdrawalModule = fx.Module("withdrawal_repository",
	fx.Provide(NewWithdrawalRepository),
)

type WithdrawalRepository interface {
	CreateWithdrawal(ctx context.Context, withdrawal *model.Withdrawal) error
	GetUserWithdrawals(ctx context.Context, userID string) ([]*model.Withdrawal, error)
	GetTotalWithdrawnByUser(ctx context.Context, userID string) (uint, error)
}

type withdrawalRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

type WithdrawalRepositoryParams struct {
	fx.In
	DB     *pgxpool.Pool
	Logger *zap.Logger
}

func NewWithdrawalRepository(p WithdrawalRepositoryParams) WithdrawalRepository {
	return &withdrawalRepository{
		db:     p.DB,
		logger: p.Logger,
	}
}

func (r *withdrawalRepository) CreateWithdrawal(ctx context.Context, withdrawal *model.Withdrawal) error {
	query := `
		INSERT INTO withdrawals (id, user_id, order_number, sum)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at
	`

	err := r.db.QueryRow(ctx, query, withdrawal.ID, withdrawal.UserID, withdrawal.OrderNumber, withdrawal.Sum).
		Scan(&withdrawal.CreatedAt)

	if err != nil {
		r.logger.Error("failed to create withdrawal",
			zap.Error(err),
			zap.String("user_id", withdrawal.UserID),
			zap.String("order_number", withdrawal.OrderNumber),
			zap.Uint("sum", withdrawal.Sum))
		return err
	}

	return nil
}

func (r *withdrawalRepository) GetUserWithdrawals(ctx context.Context, userID string) ([]*model.Withdrawal, error) {
	query := `
		SELECT id, user_id, order_number, sum, created_at
		FROM withdrawals
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		r.logger.Error("failed to get user withdrawals",
			zap.Error(err),
			zap.String("user_id", userID))
		return nil, err
	}
	defer rows.Close()

	var withdrawals []*model.Withdrawal
	for rows.Next() {
		var withdrawal model.Withdrawal
		err := rows.Scan(
			&withdrawal.ID,
			&withdrawal.UserID,
			&withdrawal.OrderNumber,
			&withdrawal.Sum,
			&withdrawal.CreatedAt,
		)
		if err != nil {
			r.logger.Error("failed to scan withdrawal",
				zap.Error(err))
			return nil, err
		}
		withdrawals = append(withdrawals, &withdrawal)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("error iterating over withdrawals",
			zap.Error(err))
		return nil, err
	}

	return withdrawals, nil
}

func (r *withdrawalRepository) GetTotalWithdrawnByUser(ctx context.Context, userID string) (uint, error) {
	query := `
		SELECT COALESCE(SUM(sum), 0)
		FROM withdrawals
		WHERE user_id = $1
	`

	var total uint
	err := r.db.QueryRow(ctx, query, userID).Scan(&total)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, nil
		}
		r.logger.Error("failed to get total withdrawn by user",
			zap.Error(err),
			zap.String("user_id", userID))
		return 0, err
	}

	return total, nil
}
