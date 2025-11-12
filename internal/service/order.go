package service

import (
	"context"
	"errors"

	"github.com/7StaSH7/halfway-diploma/internal/model"
	"github.com/7StaSH7/halfway-diploma/internal/repository"
	"github.com/google/uuid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var OrderModule = fx.Module("order_service",
	fx.Provide(NewOrderService),
)

type OrderService interface {
	CreateOrder(ctx context.Context, userID, orderNumber string) (*model.Order, error)
	GetUserOrders(ctx context.Context, userID string) ([]*model.Order, error)
}

type orderService struct {
	orderRepo repository.OrderRepository
	logger    *zap.Logger
}

type OrderServiceParams struct {
	fx.In
	OrderRepo repository.OrderRepository
	Logger    *zap.Logger
}

func NewOrderService(p OrderServiceParams) OrderService {
	return &orderService{
		orderRepo: p.OrderRepo,
		logger:    p.Logger,
	}
}

var ErrOrderConflict = errors.New("user already loaded order")
var ErrDuplicateOrder = errors.New("user duplicate order")

func (s *orderService) CreateOrder(ctx context.Context, userID, orderNumber string) (*model.Order, error) {
	existingOrder, err := s.orderRepo.GetOrderByNumber(ctx, orderNumber)
	if err != nil {
		s.logger.Error("failed to check existing order",
			zap.Error(err),
			zap.String("order_number", orderNumber))
		return nil, errors.New("failed to process order")
	}

	if existingOrder != nil {
		if existingOrder.UserID == userID {
			return existingOrder, ErrDuplicateOrder
		}

		s.logger.Warn("order already exists",
			zap.String("order_number", orderNumber),
		)

		return nil, ErrOrderConflict
	}

	newOrder := &model.Order{
		ID:      uuid.New().String(),
		UserID:  userID,
		Number:  orderNumber,
		Status:  model.OrderStatusNew,
		Accrual: 0,
	}

	err = s.orderRepo.CreateOrder(ctx, newOrder)
	if err != nil {
		s.logger.Error("failed to create order",
			zap.Error(err),
			zap.String("order_number", orderNumber),
			zap.String("user_id", userID))
		return nil, errors.New("failed to create order")
	}

	s.logger.Debug("order created successfully",
		zap.String("order_id", newOrder.ID),
		zap.String("order_number", orderNumber),
		zap.String("user_id", userID))

	return newOrder, nil
}

func (s *orderService) GetUserOrders(ctx context.Context, userID string) ([]*model.Order, error) {
	orders, err := s.orderRepo.GetUserOrders(ctx, userID)
	if err != nil {
		s.logger.Error("failed to get user orders",
			zap.Error(err),
			zap.String("user_id", userID))
		return nil, errors.New("failed to get orders")
	}
	return orders, nil
}
