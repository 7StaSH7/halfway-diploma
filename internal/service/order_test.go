package service

import (
	"context"
	"errors"
	"testing"

	"github.com/7StaSH7/halfway-diploma/internal/model"
	mock_repository "github.com/7StaSH7/halfway-diploma/internal/service/mocks/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestOrderService_CreateOrder(t *testing.T) {
	testCases := []struct {
		name           string
		setupMocks     func(*mock_repository.MockOrderRepository, string, string)
		userID         string
		orderNumber    string
		expectError    bool
		expectedError  string
		validateResult func(*testing.T, *model.Order, error)
	}{
		{
			name: "successful order creation",
			setupMocks: func(mockRepo *mock_repository.MockOrderRepository, userID, orderNumber string) {
				mockRepo.EXPECT().
					GetOrderByNumber(gomock.Any(), orderNumber).
					Return(nil, nil)
				mockRepo.EXPECT().
					CreateOrder(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, order *model.Order) error {
						assert.Equal(t, userID, order.UserID)
						assert.Equal(t, orderNumber, order.Number)
						assert.Equal(t, model.OrderStatusNew, order.Status)
						assert.Equal(t, uint(0), order.Accrual)
						return nil
					})
			},
			userID:        uuid.New().String(),
			orderNumber:   "12345678903",
			expectError:   false,
			validateResult: func(t *testing.T, order *model.Order, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, order)
				assert.Equal(t, "12345678903", order.Number)
				assert.Equal(t, model.OrderStatusNew, order.Status)
			},
		},
		{
			name: "order already exists for same user",
			setupMocks: func(mockRepo *mock_repository.MockOrderRepository, userID, orderNumber string) {
				existingOrder := &model.Order{
					ID:      uuid.New().String(),
					UserID:  userID,
					Number:  orderNumber,
					Status:  model.OrderStatusNew,
					Accrual: 0,
				}
				mockRepo.EXPECT().
					GetOrderByNumber(gomock.Any(), orderNumber).
					Return(existingOrder, nil)
			},
			userID:        uuid.New().String(),
			orderNumber:   "12345678903",
			expectError:   false,
			validateResult: func(t *testing.T, order *model.Order, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "12345678903", order.Number)
			},
		},
		{
			name: "order already exists for different user",
			setupMocks: func(mockRepo *mock_repository.MockOrderRepository, userID, orderNumber string) {
				existingOrder := &model.Order{
					ID:      uuid.New().String(),
					UserID:  uuid.New().String(),
					Number:  orderNumber,
					Status:  model.OrderStatusNew,
					Accrual: 0,
				}
				mockRepo.EXPECT().
					GetOrderByNumber(gomock.Any(), orderNumber).
					Return(existingOrder, nil)
			},
			userID:        uuid.New().String(),
			orderNumber:   "12345678903",
			expectError:   true,
			expectedError: "order already exists",
		},
		{
			name: "repository error when checking existing order",
			setupMocks: func(mockRepo *mock_repository.MockOrderRepository, userID, orderNumber string) {
				mockRepo.EXPECT().
					GetOrderByNumber(gomock.Any(), orderNumber).
					Return(nil, errors.New("database error"))
			},
			userID:        uuid.New().String(),
			orderNumber:   "12345678903",
			expectError:   true,
			expectedError: "failed to process order",
		},
		{
			name: "repository error when creating order",
			setupMocks: func(mockRepo *mock_repository.MockOrderRepository, userID, orderNumber string) {
				mockRepo.EXPECT().
					GetOrderByNumber(gomock.Any(), orderNumber).
					Return(nil, nil)
				mockRepo.EXPECT().
					CreateOrder(gomock.Any(), gomock.Any()).
					Return(errors.New("database error"))
			},
			userID:        uuid.New().String(),
			orderNumber:   "12345678903",
			expectError:   true,
			expectedError: "failed to create order",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mock_repository.NewMockOrderRepository(ctrl)
			logger, _ := zap.NewDevelopment()

			orderService := NewOrderService(OrderServiceParams{
				OrderRepo: mockRepo,
				Logger:    logger,
			})

			ctx := context.Background()

			tc.setupMocks(mockRepo, tc.userID, tc.orderNumber)

			order, err := orderService.CreateOrder(ctx, tc.userID, tc.orderNumber)

			if tc.validateResult != nil {
				tc.validateResult(t, order, err)
			} else if tc.expectError {
				assert.Error(t, err)
				if tc.expectedError != "" {
					assert.Equal(t, tc.expectedError, err.Error())
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestOrderService_GetUserOrders(t *testing.T) {
	testCases := []struct {
		name           string
		setupMocks     func(*mock_repository.MockOrderRepository, string)
		userID         string
		expectError    bool
		expectedError  string
		validateResult func(*testing.T, []*model.Order, error)
	}{
		{
			name: "successful retrieval of user orders",
			setupMocks: func(mockRepo *mock_repository.MockOrderRepository, userID string) {
				expectedOrders := []*model.Order{
					{
						ID:      uuid.New().String(),
						UserID:  userID,
						Number:  "12345678903",
						Status:  model.OrderStatusNew,
						Accrual: 0,
					},
					{
						ID:      uuid.New().String(),
						UserID:  userID,
						Number:  "09876543214",
						Status:  model.OrderStatusProcessed,
						Accrual: 100,
					},
				}
				mockRepo.EXPECT().
					GetUserOrders(gomock.Any(), userID).
					Return(expectedOrders, nil)
			},
			userID:      uuid.New().String(),
			expectError: false,
			validateResult: func(t *testing.T, orders []*model.Order, err error) {
				assert.NoError(t, err)
				assert.Len(t, orders, 2)
				assert.Equal(t, "12345678903", orders[0].Number)
				assert.Equal(t, "09876543214", orders[1].Number)
			},
		},
		{
			name: "no orders found for user",
			setupMocks: func(mockRepo *mock_repository.MockOrderRepository, userID string) {
				mockRepo.EXPECT().
					GetUserOrders(gomock.Any(), userID).
					Return([]*model.Order{}, nil)
			},
			userID:      uuid.New().String(),
			expectError: false,
			validateResult: func(t *testing.T, orders []*model.Order, err error) {
				assert.NoError(t, err)
				assert.Empty(t, orders)
			},
		},
		{
			name: "repository error when getting user orders",
			setupMocks: func(mockRepo *mock_repository.MockOrderRepository, userID string) {
				mockRepo.EXPECT().
					GetUserOrders(gomock.Any(), userID).
					Return(nil, errors.New("database error"))
			},
			userID:        uuid.New().String(),
			expectError:   true,
			expectedError: "failed to get orders",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mock_repository.NewMockOrderRepository(ctrl)
			logger, _ := zap.NewDevelopment()

			orderService := NewOrderService(OrderServiceParams{
				OrderRepo: mockRepo,
				Logger:    logger,
			})

			ctx := context.Background()

			tc.setupMocks(mockRepo, tc.userID)

			orders, err := orderService.GetUserOrders(ctx, tc.userID)

			if tc.validateResult != nil {
				tc.validateResult(t, orders, err)
			} else if tc.expectError {
				assert.Error(t, err)
				if tc.expectedError != "" {
					assert.Equal(t, tc.expectedError, err.Error())
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}