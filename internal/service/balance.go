package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/7StaSH7/halfway-diploma/internal/model"
	"github.com/7StaSH7/halfway-diploma/internal/repository"
	"github.com/google/uuid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var BalanceModule = fx.Module("balance_service",
	fx.Provide(NewBalanceService),
)

type BalanceService interface {
	GetUserBalance(ctx context.Context, userID string) (current uint, withdrawn uint, err error)
	Withdraw(ctx context.Context, userID string, order string, sum uint) error
	GetUserWithdrawals(ctx context.Context, userID string) ([]*model.Withdrawal, error)
}

type balanceService struct {
	userRepo       repository.UserRepository
	withdrawalRepo repository.WithdrawalRepository
	logger         *zap.Logger
}

type BalanceServiceParams struct {
	fx.In
	UserRepo       repository.UserRepository
	WithdrawalRepo repository.WithdrawalRepository
	Logger         *zap.Logger
}

func NewBalanceService(p BalanceServiceParams) BalanceService {
	return &balanceService{
		userRepo:       p.UserRepo,
		withdrawalRepo: p.WithdrawalRepo,
		logger:         p.Logger,
	}
}

func (s *balanceService) GetUserBalance(ctx context.Context, userID string) (current uint, withdrawn uint, err error) {
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		s.logger.Error("failed to get user",
			zap.Error(err),
			zap.String("user_id", userID))
		return 0, 0, fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		s.logger.Warn("user not found",
			zap.String("user_id", userID))
		return 0, 0, errors.New("user not found")
	}

	withdrawn, err = s.withdrawalRepo.GetTotalWithdrawnByUser(ctx, userID)
	if err != nil {
		s.logger.Error("failed to get total withdrawn amount",
			zap.Error(err),
			zap.String("user_id", userID))
		return 0, 0, fmt.Errorf("failed to get withdrawn amount: %w", err)
	}

	if user.Balance > withdrawn {
		current = user.Balance - withdrawn
	} else {
		current = 0
	}

	s.logger.Debug("user balance retrieved",
		zap.String("user_id", userID),
		zap.Uint("balance", user.Balance),
		zap.Uint("withdrawn", withdrawn),
		zap.Uint("current", current))

	return current, withdrawn, nil
}

func (s *balanceService) Withdraw(ctx context.Context, userID string, order string, sum uint) error {
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		s.logger.Error("failed to get user",
			zap.Error(err),
			zap.String("user_id", userID))
		return fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		s.logger.Warn("user not found",
			zap.String("user_id", userID))
		return errors.New("user not found")
	}

	currentBalance, _, err := s.GetUserBalance(ctx, userID)
	if err != nil {
		s.logger.Error("failed to get user balance",
			zap.Error(err),
			zap.String("user_id", userID))
		return fmt.Errorf("failed to check balance: %w", err)
	}

	if currentBalance < sum {
		s.logger.Warn("insufficient funds",
			zap.String("user_id", userID),
			zap.Uint("current_balance", currentBalance),
			zap.Uint("requested_amount", sum))
		return errors.New("insufficient funds")
	}

	withdrawal := &model.Withdrawal{
		ID:          uuid.New().String(),
		UserID:      userID,
		OrderNumber: order,
		Sum:         sum,
	}

	err = s.withdrawalRepo.CreateWithdrawal(ctx, withdrawal)
	if err != nil {
		s.logger.Error("failed to create withdrawal",
			zap.Error(err),
			zap.String("user_id", userID),
			zap.String("order", order),
			zap.Uint("sum", sum))
		return fmt.Errorf("failed to create withdrawal: %w", err)
	}

	s.logger.Info("withdrawal created successfully",
		zap.String("withdrawal_id", withdrawal.ID),
		zap.String("user_id", userID),
		zap.String("order", order),
		zap.Uint("sum", sum))

	return nil
}

func (s *balanceService) GetUserWithdrawals(ctx context.Context, userID string) ([]*model.Withdrawal, error) {
	withdrawals, err := s.withdrawalRepo.GetUserWithdrawals(ctx, userID)
	if err != nil {
		s.logger.Error("failed to get user withdrawals",
			zap.Error(err),
			zap.String("user_id", userID))
		return nil, fmt.Errorf("failed to get withdrawals: %w", err)
	}

	s.logger.Debug("user withdrawals retrieved",
		zap.String("user_id", userID),
		zap.Int("count", len(withdrawals)))

	return withdrawals, nil
}
