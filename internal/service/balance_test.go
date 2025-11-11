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

func TestBalanceService_GetUserBalance(t *testing.T) {
	testCases := []struct {
		name           string
		setupMocks     func(*mock_repository.MockUserRepository, *mock_repository.MockWithdrawalRepository, string)
		userID         string
		expectError    bool
		expectedError  string
		validateResult func(*testing.T, uint, uint, error)
	}{
		{
			name: "successful balance retrieval",
			setupMocks: func(mockUserRepo *mock_repository.MockUserRepository, mockWithdrawalRepo *mock_repository.MockWithdrawalRepository, userID string) {
				user := &model.User{
					ID:       userID,
					Username: "testuser",
					Balance:  10000,
				}
				mockUserRepo.EXPECT().
					GetUserByID(gomock.Any(), userID).
					Return(user, nil)

				mockWithdrawalRepo.EXPECT().
					GetTotalWithdrawnByUser(gomock.Any(), userID).
					Return(uint(3000), nil)
			},
			userID:      uuid.New().String(),
			expectError: false,
			validateResult: func(t *testing.T, current uint, withdrawn uint, err error) {
				assert.NoError(t, err)
				assert.Equal(t, uint(7000), current)
				assert.Equal(t, uint(3000), withdrawn)
			},
		},
		{
			name: "user not found",
			setupMocks: func(mockUserRepo *mock_repository.MockUserRepository, mockWithdrawalRepo *mock_repository.MockWithdrawalRepository, userID string) {
				mockUserRepo.EXPECT().
					GetUserByID(gomock.Any(), userID).
					Return(nil, nil)
			},
			userID:        uuid.New().String(),
			expectError:   true,
			expectedError: "user not found",
		},
		{
			name: "error getting user",
			setupMocks: func(mockUserRepo *mock_repository.MockUserRepository, mockWithdrawalRepo *mock_repository.MockWithdrawalRepository, userID string) {
				mockUserRepo.EXPECT().
					GetUserByID(gomock.Any(), userID).
					Return(nil, errors.New("database error"))
			},
			userID:        uuid.New().String(),
			expectError:   true,
			expectedError: "failed to get user",
		},
		{
			name: "error getting total withdrawn",
			setupMocks: func(mockUserRepo *mock_repository.MockUserRepository, mockWithdrawalRepo *mock_repository.MockWithdrawalRepository, userID string) {
				user := &model.User{
					ID:       userID,
					Username: "testuser",
					Balance:  10000,
				}
				mockUserRepo.EXPECT().
					GetUserByID(gomock.Any(), userID).
					Return(user, nil)

				mockWithdrawalRepo.EXPECT().
					GetTotalWithdrawnByUser(gomock.Any(), userID).
					Return(uint(0), errors.New("database error"))
			},
			userID:        uuid.New().String(),
			expectError:   true,
			expectedError: "failed to get withdrawn amount",
		},
		{
			name: "negative balance handled correctly",
			setupMocks: func(mockUserRepo *mock_repository.MockUserRepository, mockWithdrawalRepo *mock_repository.MockWithdrawalRepository, userID string) {
				user := &model.User{
					ID:       userID,
					Username: "testuser",
					Balance:  1000,
				}
				mockUserRepo.EXPECT().
					GetUserByID(gomock.Any(), userID).
					Return(user, nil)

				mockWithdrawalRepo.EXPECT().
					GetTotalWithdrawnByUser(gomock.Any(), userID).
					Return(uint(2000), nil)
			},
			userID:      uuid.New().String(),
			expectError: false,
			validateResult: func(t *testing.T, current uint, withdrawn uint, err error) {
				assert.NoError(t, err)
				assert.Equal(t, uint(0), current)
				assert.Equal(t, uint(2000), withdrawn)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockUserRepo := mock_repository.NewMockUserRepository(ctrl)
			mockWithdrawalRepo := mock_repository.NewMockWithdrawalRepository(ctrl)
			logger, _ := zap.NewDevelopment()

			balanceService := NewBalanceService(BalanceServiceParams{
				UserRepo:       mockUserRepo,
				WithdrawalRepo: mockWithdrawalRepo,
				Logger:         logger,
			})

			ctx := context.Background()

			tc.setupMocks(mockUserRepo, mockWithdrawalRepo, tc.userID)

			current, withdrawn, err := balanceService.GetUserBalance(ctx, tc.userID)

			if tc.validateResult != nil {
				tc.validateResult(t, current, withdrawn, err)
			} else if tc.expectError {
				assert.Error(t, err)
				if tc.expectedError != "" {
					assert.Contains(t, err.Error(), tc.expectedError)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBalanceService_Withdraw(t *testing.T) {
	testCases := []struct {
		name          string
		setupMocks    func(*mock_repository.MockUserRepository, *mock_repository.MockWithdrawalRepository, string, string, uint)
		userID        string
		order         string
		sum           uint
		expectError   bool
		expectedError string
	}{
		{
			name: "successful withdrawal",
			setupMocks: func(mockUserRepo *mock_repository.MockUserRepository, mockWithdrawalRepo *mock_repository.MockWithdrawalRepository, userID, order string, sum uint) {
				user := &model.User{
					ID:       userID,
					Username: "testuser",
					Balance:  10000,
				}
				mockUserRepo.EXPECT().
					GetUserByID(gomock.Any(), userID).
					Return(user, nil)

				mockUserRepo.EXPECT().
					GetUserByID(gomock.Any(), userID).
					Return(user, nil)
				mockWithdrawalRepo.EXPECT().
					GetTotalWithdrawnByUser(gomock.Any(), userID).
					Return(uint(2000), nil)

				mockWithdrawalRepo.EXPECT().
					CreateWithdrawal(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, withdrawal *model.Withdrawal) error {
						assert.Equal(t, userID, withdrawal.UserID)
						assert.Equal(t, order, withdrawal.OrderNumber)
						assert.Equal(t, sum, withdrawal.Sum)
						assert.NotEmpty(t, withdrawal.ID)
						return nil
					})
			},
			userID:      uuid.New().String(),
			order:       "2377225624",
			sum:         5000,
			expectError: false,
		},
		{
			name: "user not found",
			setupMocks: func(mockUserRepo *mock_repository.MockUserRepository, mockWithdrawalRepo *mock_repository.MockWithdrawalRepository, userID, order string, sum uint) {
				mockUserRepo.EXPECT().
					GetUserByID(gomock.Any(), userID).
					Return(nil, nil)
			},
			userID:        uuid.New().String(),
			order:         "2377225624",
			sum:           5000,
			expectError:   true,
			expectedError: "user not found",
		},
		{
			name: "error getting user",
			setupMocks: func(mockUserRepo *mock_repository.MockUserRepository, mockWithdrawalRepo *mock_repository.MockWithdrawalRepository, userID, order string, sum uint) {
				mockUserRepo.EXPECT().
					GetUserByID(gomock.Any(), userID).
					Return(nil, errors.New("database error"))
			},
			userID:        uuid.New().String(),
			order:         "2377225624",
			sum:           5000,
			expectError:   true,
			expectedError: "failed to get user",
		},
		{
			name: "insufficient funds",
			setupMocks: func(mockUserRepo *mock_repository.MockUserRepository, mockWithdrawalRepo *mock_repository.MockWithdrawalRepository, userID, order string, sum uint) {
				user := &model.User{
					ID:       userID,
					Username: "testuser",
					Balance:  10000,
				}
				mockUserRepo.EXPECT().
					GetUserByID(gomock.Any(), userID).
					Return(user, nil)

				mockUserRepo.EXPECT().
					GetUserByID(gomock.Any(), userID).
					Return(user, nil)
				mockWithdrawalRepo.EXPECT().
					GetTotalWithdrawnByUser(gomock.Any(), userID).
					Return(uint(8000), nil)
			},
			userID:        uuid.New().String(),
			order:         "2377225624",
			sum:           5000,
			expectError:   true,
			expectedError: "insufficient funds",
		},
		{
			name: "error creating withdrawal",
			setupMocks: func(mockUserRepo *mock_repository.MockUserRepository, mockWithdrawalRepo *mock_repository.MockWithdrawalRepository, userID, order string, sum uint) {
				user := &model.User{
					ID:       userID,
					Username: "testuser",
					Balance:  10000,
				}
				mockUserRepo.EXPECT().
					GetUserByID(gomock.Any(), userID).
					Return(user, nil)

				mockUserRepo.EXPECT().
					GetUserByID(gomock.Any(), userID).
					Return(user, nil)
				mockWithdrawalRepo.EXPECT().
					GetTotalWithdrawnByUser(gomock.Any(), userID).
					Return(uint(2000), nil)

				mockWithdrawalRepo.EXPECT().
					CreateWithdrawal(gomock.Any(), gomock.Any()).
					Return(errors.New("database error"))
			},
			userID:        uuid.New().String(),
			order:         "2377225624",
			sum:           5000,
			expectError:   true,
			expectedError: "failed to create withdrawal",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockUserRepo := mock_repository.NewMockUserRepository(ctrl)
			mockWithdrawalRepo := mock_repository.NewMockWithdrawalRepository(ctrl)
			logger, _ := zap.NewDevelopment()

			balanceService := NewBalanceService(BalanceServiceParams{
				UserRepo:       mockUserRepo,
				WithdrawalRepo: mockWithdrawalRepo,
				Logger:         logger,
			})

			ctx := context.Background()

			tc.setupMocks(mockUserRepo, mockWithdrawalRepo, tc.userID, tc.order, tc.sum)

			err := balanceService.Withdraw(ctx, tc.userID, tc.order, tc.sum)

			if tc.expectError {
				assert.Error(t, err)
				if tc.expectedError != "" {
					assert.Contains(t, err.Error(), tc.expectedError)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBalanceService_GetUserWithdrawals(t *testing.T) {
	testCases := []struct {
		name           string
		setupMocks     func(*mock_repository.MockUserRepository, *mock_repository.MockWithdrawalRepository, string)
		userID         string
		expectError    bool
		expectedError  string
		validateResult func(*testing.T, []*model.Withdrawal, error)
	}{
		{
			name: "successful retrieval of user withdrawals",
			setupMocks: func(mockUserRepo *mock_repository.MockUserRepository, mockWithdrawalRepo *mock_repository.MockWithdrawalRepository, userID string) {
				expectedWithdrawals := []*model.Withdrawal{
					{
						ID:          uuid.New().String(),
						UserID:      userID,
						OrderNumber: "2377225624",
						Sum:         5000,
					},
					{
						ID:          uuid.New().String(),
						UserID:      userID,
						OrderNumber: "1234567890",
						Sum:         3000,
					},
				}
				mockWithdrawalRepo.EXPECT().
					GetUserWithdrawals(gomock.Any(), userID).
					Return(expectedWithdrawals, nil)
			},
			userID:      uuid.New().String(),
			expectError: false,
			validateResult: func(t *testing.T, withdrawals []*model.Withdrawal, err error) {
				assert.NoError(t, err)
				assert.Len(t, withdrawals, 2)
				assert.Equal(t, "2377225624", withdrawals[0].OrderNumber)
				assert.Equal(t, "1234567890", withdrawals[1].OrderNumber)
			},
		},
		{
			name: "no withdrawals found for user",
			setupMocks: func(mockUserRepo *mock_repository.MockUserRepository, mockWithdrawalRepo *mock_repository.MockWithdrawalRepository, userID string) {
				mockWithdrawalRepo.EXPECT().
					GetUserWithdrawals(gomock.Any(), userID).
					Return([]*model.Withdrawal{}, nil)
			},
			userID:      uuid.New().String(),
			expectError: false,
			validateResult: func(t *testing.T, withdrawals []*model.Withdrawal, err error) {
				assert.NoError(t, err)
				assert.Empty(t, withdrawals)
			},
		},
		{
			name: "repository error when getting user withdrawals",
			setupMocks: func(mockUserRepo *mock_repository.MockUserRepository, mockWithdrawalRepo *mock_repository.MockWithdrawalRepository, userID string) {
				mockWithdrawalRepo.EXPECT().
					GetUserWithdrawals(gomock.Any(), userID).
					Return(nil, errors.New("database error"))
			},
			userID:        uuid.New().String(),
			expectError:   true,
			expectedError: "failed to get withdrawals",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockUserRepo := mock_repository.NewMockUserRepository(ctrl)
			mockWithdrawalRepo := mock_repository.NewMockWithdrawalRepository(ctrl)
			logger, _ := zap.NewDevelopment()

			balanceService := NewBalanceService(BalanceServiceParams{
				UserRepo:       mockUserRepo,
				WithdrawalRepo: mockWithdrawalRepo,
				Logger:         logger,
			})

			ctx := context.Background()

			tc.setupMocks(mockUserRepo, mockWithdrawalRepo, tc.userID)

			withdrawals, err := balanceService.GetUserWithdrawals(ctx, tc.userID)

			if tc.validateResult != nil {
				tc.validateResult(t, withdrawals, err)
			} else if tc.expectError {
				assert.Error(t, err)
				if tc.expectedError != "" {
					assert.Contains(t, err.Error(), tc.expectedError)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
