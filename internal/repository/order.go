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

var OrderModule = fx.Module("order_repository",
	fx.Provide(NewOrderRepository),
)

type OrderRepository interface {
	CreateOrder(ctx context.Context, order *model.Order) error
	GetOrderByNumber(ctx context.Context, number string) (*model.Order, error)
	GetUserOrders(ctx context.Context, userID string) ([]*model.Order, error)
	GetOrdersByStatus(ctx context.Context, status model.OrderStatus) ([]*model.Order, error)
	UpdateOrder(ctx context.Context, tx pgx.Tx, order *model.Order) error
}

type orderRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

type OrderRepositoryParams struct {
	fx.In
	DB     *pgxpool.Pool
	Logger *zap.Logger
}

func NewOrderRepository(p OrderRepositoryParams) OrderRepository {
	return &orderRepository{
		db:     p.DB,
		logger: p.Logger,
	}
}

func (r *orderRepository) CreateOrder(ctx context.Context, order *model.Order) error {
	query := `
		INSERT INTO orders (id, user_id, number, status, accrual)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at, updated_at
	`

	err := r.db.QueryRow(ctx, query, order.ID, order.UserID, order.Number, order.Status, order.Accrual).
		Scan(&order.CreatedAt, &order.UpdatedAt)

	if err != nil {
		r.logger.Error("failed to create order",
			zap.Error(err),
			zap.String("order_number", order.Number))
		return err
	}

	return nil
}

func (r *orderRepository) GetOrderByNumber(ctx context.Context, number string) (*model.Order, error) {
	query := `
		SELECT id, user_id, number, status, accrual, created_at, updated_at
		FROM orders
		WHERE number = $1
	`

	var order model.Order
	err := r.db.QueryRow(ctx, query, number).Scan(
		&order.ID,
		&order.UserID,
		&order.Number,
		&order.Status,
		&order.Accrual,
		&order.CreatedAt,
		&order.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.Debug("order not found", zap.String("order_number", number))
			return nil, nil
		}
		r.logger.Error("failed to get order by number",
			zap.Error(err),
			zap.String("order_number", number))
		return nil, err
	}

	return &order, nil
}

func (r *orderRepository) GetUserOrders(ctx context.Context, userID string) ([]*model.Order, error) {
	query := `
		SELECT id, user_id, number, status, accrual, created_at, updated_at
		FROM orders
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		r.logger.Error("failed to get user orders",
			zap.Error(err),
			zap.String("user_id", userID))
		return nil, err
	}
	defer rows.Close()

	var orders []*model.Order
	for rows.Next() {
		var order model.Order
		err := rows.Scan(
			&order.ID,
			&order.UserID,
			&order.Number,
			&order.Status,
			&order.Accrual,
			&order.CreatedAt,
			&order.UpdatedAt,
		)
		if err != nil {
			r.logger.Error("failed to scan order",
				zap.Error(err))
			return nil, err
		}
		orders = append(orders, &order)
	}

	return orders, nil
}

func (r *orderRepository) GetOrdersByStatus(ctx context.Context, status model.OrderStatus) ([]*model.Order, error) {
	query := `
		SELECT id, user_id, number, status, accrual, created_at, updated_at
		FROM orders
		WHERE status = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, status)
	if err != nil {
		r.logger.Error("failed to get orders by status",
			zap.Error(err),
			zap.String("status", string(status)))
		return nil, err
	}
	defer rows.Close()

	var orders []*model.Order
	for rows.Next() {
		var order model.Order
		err := rows.Scan(
			&order.ID,
			&order.UserID,
			&order.Number,
			&order.Status,
			&order.Accrual,
			&order.CreatedAt,
			&order.UpdatedAt,
		)
		if err != nil {
			r.logger.Error("failed to scan order",
				zap.Error(err))
			return nil, err
		}
		orders = append(orders, &order)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("error iterating over orders",
			zap.Error(err))
		return nil, err
	}

	return orders, nil
}

func (r *orderRepository) UpdateOrder(ctx context.Context, tx pgx.Tx, order *model.Order) error {
	query := `
		UPDATE orders
		SET user_id = $1, number = $2, status = $3, accrual = $4, updated_at = CURRENT_TIMESTAMP
		WHERE id = $5
		RETURNING created_at, updated_at
	`

	var err error
	if tx == nil {
		err = r.db.QueryRow(ctx, query, order.UserID, order.Number, order.Status, order.Accrual, order.ID).
			Scan(&order.CreatedAt, &order.UpdatedAt)
	} else {
		err = tx.QueryRow(ctx, query, order.UserID, order.Number, order.Status, order.Accrual, order.ID).
			Scan(&order.CreatedAt, &order.UpdatedAt)
	}

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.Warn("order to update not found", zap.String("order_id", order.ID))
			return errors.New("order not found")
		}
		r.logger.Error("failed to update order",
			zap.Error(err),
			zap.String("order_id", order.ID))
		return err
	}

	r.logger.Debug("order updated",
		zap.String("order_id", order.ID),
		zap.String("order_number", order.Number),
		zap.String("status", string(order.Status)),
		zap.Uint("accrual", order.Accrual))

	return nil
}
