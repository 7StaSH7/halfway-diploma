package worker

import (
	"context"
	"time"

	"github.com/7StaSH7/halfway-diploma/internal/client/accrual"
	"github.com/7StaSH7/halfway-diploma/internal/model"
	"github.com/7StaSH7/halfway-diploma/internal/repository"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var OrderWorkerModule = fx.Module("order_worker",
	fx.Provide(NewOrderWorker),
	fx.Invoke(RegisterWorkerHooks),
)

type OrderWorker struct {
	orderRepo     repository.OrderRepository
	accrualClient accrual.AccrualClient
	logger        *zap.Logger
	interval      time.Duration
}

type OrderWorkerParams struct {
	fx.In
	OrderRepo     repository.OrderRepository
	AccrualClient accrual.AccrualClient
	Logger        *zap.Logger
}

func NewOrderWorker(p OrderWorkerParams) *OrderWorker {
	return &OrderWorker{
		orderRepo:     p.OrderRepo,
		accrualClient: p.AccrualClient,
		logger:        p.Logger,
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
		return w.orderRepo.UpdateOrderStatus(ctx, order.ID, model.OrderStatusInvalid)
	case "PROCESSING":
		return w.orderRepo.UpdateOrderStatus(ctx, order.ID, model.OrderStatusProcessing)
	case "PROCESSED":
		if accrualResp.Accrual != nil {
			if err := w.orderRepo.UpdateOrderAccrual(ctx, order.ID, *accrualResp.Accrual); err != nil {
				return err
			}
		}
		return w.orderRepo.UpdateOrderStatus(ctx, order.ID, model.OrderStatusProcessed)
	case "REGISTERED":
		return nil
	default:
		w.logger.Warn("unknown accrual status",
			zap.String("order_id", order.ID),
			zap.String("status", accrualResp.Status))
		return nil
	}
}
