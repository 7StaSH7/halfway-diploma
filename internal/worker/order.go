package worker

import (
	"context"
	"time"

	"github.com/7StaSH7/halfway-diploma/internal/client/accrual"
	"github.com/7StaSH7/halfway-diploma/internal/model"
	"github.com/7StaSH7/halfway-diploma/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var OrderWorkerModule = fx.Module("order_worker",
	fx.Provide(NewOrderWorker),
	fx.Invoke(RegisterWorkerHooks),
)

type OrderWorker struct {
	orderRepo     repository.OrderRepository
	userRepo      repository.UserRepository
	accrualClient accrual.AccrualClient
	logger        *zap.Logger
	dbPool        *pgxpool.Pool
	interval      time.Duration
}

type OrderWorkerParams struct {
	fx.In
	OrderRepo     repository.OrderRepository
	UserRepo      repository.UserRepository
	AccrualClient accrual.AccrualClient
	Logger        *zap.Logger
	DB            *pgxpool.Pool
}

func NewOrderWorker(p OrderWorkerParams) *OrderWorker {
	return &OrderWorker{
		orderRepo:     p.OrderRepo,
		userRepo:      p.UserRepo,
		accrualClient: p.AccrualClient,
		logger:        p.Logger,
		dbPool:        p.DB,
		interval:      5 * time.Second,
	}
}

func RegisterWorkerHooks(lc fx.Lifecycle, worker *OrderWorker, logger *zap.Logger) {
	ctx, cancel := context.WithCancel(context.Background())

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			logger.Info("starting order worker")
			go worker.Start(ctx)
			return nil
		},
		OnStop: func(context.Context) error {
			logger.Info("stopping order worker")
			cancel()
			return nil
		},
	})
}

func (w *OrderWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.processNewOrders(ctx); err != nil {
				w.logger.Error("failed to process new orders", zap.Error(err))
			}
		}
	}
}

func (w *OrderWorker) processNewOrders(ctx context.Context) error {
	orders, err := w.orderRepo.GetOrdersByStatus(ctx, model.OrderStatusNew)
	if err != nil {
		return err
	}

	for _, order := range orders {
		if err := w.processOrder(ctx, order); err != nil {
			w.logger.Error("failed to process order",
				zap.String("order_id", order.ID),
				zap.String("order_number", order.Number),
				zap.Error(err))
			continue
		}
	}

	return nil
}

func (w *OrderWorker) processOrder(ctx context.Context, order *model.Order) error {
	accrualResp, err := w.accrualClient.GetOrderAccrual(ctx, order.Number)
	if err != nil {
		return err
	}

	if accrualResp == nil {
		return nil
	}

	switch accrualResp.Status {
	case "INVALID":
		order.Status = model.OrderStatusInvalid
		return w.orderRepo.UpdateOrder(ctx, nil, order)
	case "PROCESSING":
		order.Status = model.OrderStatusProcessing
		return w.orderRepo.UpdateOrder(ctx, nil, order)
	case "PROCESSED":
		tx, err := w.dbPool.Begin(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)

		order.Status = model.OrderStatusProcessed
		if accrualResp.Accrual != nil {
			if *accrualResp.Accrual >= 0 {
				order.Accrual = uint(*accrualResp.Accrual * 100)
			}
		}

		if err := w.orderRepo.UpdateOrder(ctx, tx, order); err != nil {
			return err
		}

		if accrualResp.Accrual != nil && *accrualResp.Accrual > 0 {
			user, err := w.userRepo.GetUserByID(ctx, order.UserID)
			if err != nil {
				return err
			}

			if user != nil {
				accrualValue := uint(*accrualResp.Accrual * 100)
				newBalance := user.Balance + accrualValue
				if err := w.userRepo.UpdateUserBalance(ctx, tx, user.ID, newBalance); err != nil {
					return err
				}
			}
		}

		return tx.Commit(ctx)
	case "REGISTERED":
		return nil
	default:
		w.logger.Warn("unknown accrual status",
			zap.String("order_id", order.ID),
			zap.String("status", accrualResp.Status))
		return nil
	}
}
