package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/7StaSH7/halfway-diploma/internal/model"
	"github.com/7StaSH7/halfway-diploma/internal/repository"
	"github.com/7StaSH7/halfway-diploma/internal/utils"
	"github.com/google/uuid"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

var AuthModule = fx.Module("auth_service",
	fx.Provide(NewAuthService),
)

type AuthService interface {
	Register(ctx context.Context, username, password string) (*model.User, string, error)
	Login(ctx context.Context, username, password string) (*model.User, string, error)
	ValidateToken(tokenString string) (string, error)
}

type authService struct {
	userRepo   repository.UserRepository
	jwtService utils.JWTService
	logger     *zap.Logger
}

type AuthServiceParams struct {
	fx.In
	UserRepo   repository.UserRepository
	JWTService utils.JWTService
	Logger     *zap.Logger
}

func NewAuthService(p AuthServiceParams) AuthService {
	return &authService{
		userRepo:   p.UserRepo,
		jwtService: p.JWTService,
		logger:     p.Logger,
	}
}

func (s *authService) Register(ctx context.Context, username, password string) (*model.User, string, error) {
	exists, err := s.userRepo.UserExists(ctx, username)
	if err != nil {
		s.logger.Error("failed to check user existence", zap.String("username", username))
		return nil, "", fmt.Errorf("failed to check user existence: %w", err)
	}

	if exists {
		s.logger.Warn("user already exists", zap.String("username", username))
		return nil, "", errors.New("user already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("failed to hash password", zap.Error(err))
		return nil, "", errors.New("failed to hash password")
	}

	newUser := &model.User{
		ID:       uuid.New().String(),
		Username: username,
		Password: string(hashedPassword),
		Balance:  0,
	}

	err = s.userRepo.CreateUser(ctx, newUser)
	if err != nil {
		s.logger.Error("failed to create user", zap.Error(err))
		return nil, "", errors.New("failed to create user")
	}

	token, err := s.jwtService.GenerateToken(newUser.ID)
	if err != nil {
		s.logger.Error("failed to generate authentication token", zap.Error(err))
		return nil, "", errors.New("failed to generate authentication token")
	}

	s.logger.Info("user registered successfully",
		zap.String("user_id", newUser.ID),
		zap.String("username", username))

	return newUser, token, nil
}

func (s *authService) Login(ctx context.Context, username, password string) (*model.User, string, error) {
	user, err := s.userRepo.GetUserByUsername(ctx, username)
	if err != nil {
		s.logger.Error("failed to get user", zap.Error(err))
		return nil, "", errors.New("invalid credentials")
	}

	if user == nil {
		s.logger.Warn("user not found", zap.String("username", username))
		return nil, "", errors.New("invalid credentials")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		s.logger.Warn("invalid password", zap.String("username", username))
		return nil, "", errors.New("invalid credentials")
	}

	token, err := s.jwtService.GenerateToken(user.ID)
	if err != nil {
		s.logger.Error("failed to generate token", zap.Error(err))
		return nil, "", errors.New("invalid credentials")
	}

	s.logger.Info("user logged in successfully",
		zap.String("user_id", user.ID),
		zap.String("username", username))

	return user, token, nil
}

func (s *authService) ValidateToken(tokenString string) (string, error) {
	claims, err := s.jwtService.ValidateToken(tokenString)
	if err != nil {
		s.logger.Debug("token validation failed", zap.Error(err))
		return "", errors.New("invalid token")
	}

	if claims.UserID == "" {
		s.logger.Warn("empty user ID in claims")
		return "", errors.New("invalid token")
	}

	return claims.UserID, nil
}
