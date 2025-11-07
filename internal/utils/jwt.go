package utils

import (
	"errors"
	"time"

	"github.com/7StaSH7/halfway-diploma/internal/config"
	"github.com/dgrijalva/jwt-go/v4"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var JWTModule = fx.Module("jwt",
	fx.Provide(NewJWTService),
)

type JWTService interface {
	GenerateToken(userID string) (string, error)
	ValidateToken(tokenString string) (*Claims, error)
	GetTokenExpiry() time.Duration
}

type Claims struct {
	UserID string `json:"user_id"`
	jwt.StandardClaims
}

type jwtService struct {
	secretKey   string
	tokenExpiry time.Duration
	logger      *zap.Logger
}

type JWTServiceParams struct {
	fx.In
	Config *config.ServerConfig
	Logger *zap.Logger
}

func NewJWTService(p JWTServiceParams) JWTService {
	tokenExpiry := 24 * time.Hour

	return &jwtService{
		secretKey:   p.Config.JWTSecret,
		tokenExpiry: tokenExpiry,
		logger:      p.Logger,
	}
}

func (s *jwtService) GenerateToken(userID string) (string, error) {
	expirationTime := time.Now().Add(s.tokenExpiry)

	claims := &Claims{
		UserID: userID,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: jwt.At(expirationTime),
			IssuedAt:  jwt.At(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.secretKey))
	if err != nil {
		s.logger.Error("Failed to generate JWT token", zap.Error(err))
		return "", err
	}

	s.logger.Debug("JWT token generated successfully", zap.String("user_id", userID))
	return tokenString, nil
}

func (s *jwtService) ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			s.logger.Warn("Unexpected signing method", zap.Any("method", token.Header["alg"]))
			return nil, errors.New("unexpected signing method")
		}
		return []byte(s.secretKey), nil
	})

	if err != nil {
		s.logger.Debug("Failed to parse JWT token", zap.Error(err))
		return nil, err
	}

	if !token.Valid {
		s.logger.Warn("Invalid JWT token")
		return nil, errors.New("invalid token")
	}

	s.logger.Debug("JWT token validated successfully", zap.String("user_id", claims.UserID))
	return claims, nil
}

func (s *jwtService) GetTokenExpiry() time.Duration {
	return s.tokenExpiry
}
