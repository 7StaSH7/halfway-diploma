package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/7StaSH7/halfway-diploma/internal/model"
	mock_repository "github.com/7StaSH7/halfway-diploma/internal/service/mocks/repository"
	mock_utils "github.com/7StaSH7/halfway-diploma/internal/service/mocks/utils"
	"github.com/7StaSH7/halfway-diploma/internal/utils"
	jwt "github.com/dgrijalva/jwt-go/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

func TestAuthService_Register(t *testing.T) {
	testCases := []struct {
		name           string
		setupMocks     func(*mock_repository.MockUserRepository, *mock_utils.MockJWTService, string, string)
		username       string
		password       string
		expectError    bool
		expectedError  string
		validateResult func(*testing.T, *model.User, string, error)
	}{
		{
			name: "successful registration",
			setupMocks: func(mockUserRepo *mock_repository.MockUserRepository, mockJWTService *mock_utils.MockJWTService, username, password string) {
				mockUserRepo.EXPECT().
					UserExists(gomock.Any(), username).
					Return(false, nil)

				mockUserRepo.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, user *model.User) error {
						assert.Equal(t, username, user.Username)
						assert.NotEmpty(t, user.Password)
						assert.Equal(t, uint(0), user.Balance)
						err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
						assert.NoError(t, err)
						return nil
					})

				token := "test-token"
				mockJWTService.EXPECT().
					GenerateToken(gomock.Any()).
					DoAndReturn(func(userID string) (string, error) {
						assert.NotEmpty(t, userID)
						return token, nil
					})
			},
			username:    "testuser",
			password:    "testpassword",
			expectError: false,
			validateResult: func(t *testing.T, user *model.User, token string, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, "testuser", user.Username)
				assert.Equal(t, uint(0), user.Balance)
				assert.Equal(t, "test-token", token)
			},
		},
		{
			name: "user already exists",
			setupMocks: func(mockUserRepo *mock_repository.MockUserRepository, mockJWTService *mock_utils.MockJWTService, username, password string) {
				mockUserRepo.EXPECT().
					UserExists(gomock.Any(), username).
					Return(true, nil)
			},
			username:      "testuser",
			password:      "testpassword",
			expectError:   true,
			expectedError: "user already exists",
		},
		{
			name: "error checking user existence",
			setupMocks: func(mockUserRepo *mock_repository.MockUserRepository, mockJWTService *mock_utils.MockJWTService, username, password string) {
				mockUserRepo.EXPECT().
					UserExists(gomock.Any(), username).
					Return(false, errors.New("database error"))
			},
			username:      "testuser",
			password:      "testpassword",
			expectError:   true,
			expectedError: "failed to check user existence: database error",
		},
		{
			name: "error creating user",
			setupMocks: func(mockUserRepo *mock_repository.MockUserRepository, mockJWTService *mock_utils.MockJWTService, username, password string) {
				mockUserRepo.EXPECT().
					UserExists(gomock.Any(), username).
					Return(false, nil)

				mockUserRepo.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(errors.New("database error"))
			},
			username:      "testuser",
			password:      "testpassword",
			expectError:   true,
			expectedError: "failed to create user",
		},
		{
			name: "error generating token",
			setupMocks: func(mockUserRepo *mock_repository.MockUserRepository, mockJWTService *mock_utils.MockJWTService, username, password string) {
				mockUserRepo.EXPECT().
					UserExists(gomock.Any(), username).
					Return(false, nil)

				mockUserRepo.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(nil)

				mockJWTService.EXPECT().
					GenerateToken(gomock.Any()).
					Return("", errors.New("token error"))
			},
			username:      "testuser",
			password:      "testpassword",
			expectError:   true,
			expectedError: "failed to generate authentication token",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockUserRepo := mock_repository.NewMockUserRepository(ctrl)
			mockJWTService := mock_utils.NewMockJWTService(ctrl)
			logger, _ := zap.NewDevelopment()

			authService := NewAuthService(AuthServiceParams{
				UserRepo:   mockUserRepo,
				JWTService: mockJWTService,
				Logger:     logger,
			})

			ctx := context.Background()

			tc.setupMocks(mockUserRepo, mockJWTService, tc.username, tc.password)

			user, token, err := authService.Register(ctx, tc.username, tc.password)

			if tc.validateResult != nil {
				tc.validateResult(t, user, token, err)
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

func TestAuthService_Login(t *testing.T) {
	testCases := []struct {
		name           string
		setupMocks     func(*mock_repository.MockUserRepository, *mock_utils.MockJWTService, string, string, *model.User)
		username       string
		password       string
		setupUser      func() *model.User
		expectError    bool
		expectedError  string
		validateResult func(*testing.T, *model.User, string, error)
	}{
		{
			name: "successful login",
			setupMocks: func(mockUserRepo *mock_repository.MockUserRepository, mockJWTService *mock_utils.MockJWTService, username, password string, user *model.User) {
				mockUserRepo.EXPECT().
					GetUserByUsername(gomock.Any(), username).
					Return(user, nil)

				token := "test-token"
				mockJWTService.EXPECT().
					GenerateToken(user.ID).
					Return(token, nil)
			},
			username: "testuser",
			password: "testpassword",
			setupUser: func() *model.User {
				userID := uuid.New().String()
				hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("testpassword"), bcrypt.DefaultCost)
				return &model.User{
					ID:       userID,
					Username: "testuser",
					Password: string(hashedPassword),
					Balance:  100,
				}
			},
			expectError: false,
			validateResult: func(t *testing.T, user *model.User, token string, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, "test-token", token)
			},
		},
		{
			name: "user not found",
			setupMocks: func(mockUserRepo *mock_repository.MockUserRepository, mockJWTService *mock_utils.MockJWTService, username, password string, user *model.User) {
				mockUserRepo.EXPECT().
					GetUserByUsername(gomock.Any(), username).
					Return(nil, nil)
			},
			username:      "testuser",
			password:      "testpassword",
			setupUser:     func() *model.User { return nil },
			expectError:   true,
			expectedError: "invalid credentials",
		},
		{
			name: "error getting user",
			setupMocks: func(mockUserRepo *mock_repository.MockUserRepository, mockJWTService *mock_utils.MockJWTService, username, password string, user *model.User) {
				mockUserRepo.EXPECT().
					GetUserByUsername(gomock.Any(), username).
					Return(nil, errors.New("database error"))
			},
			username:      "testuser",
			password:      "testpassword",
			setupUser:     func() *model.User { return nil },
			expectError:   true,
			expectedError: "invalid credentials",
		},
		{
			name: "invalid password",
			setupMocks: func(mockUserRepo *mock_repository.MockUserRepository, mockJWTService *mock_utils.MockJWTService, username, password string, user *model.User) {
				mockUserRepo.EXPECT().
					GetUserByUsername(gomock.Any(), username).
					Return(user, nil)
			},
			username: "testuser",
			password: "wrongpassword",
			setupUser: func() *model.User {
				userID := uuid.New().String()
				hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("testpassword"), bcrypt.DefaultCost)
				return &model.User{
					ID:       userID,
					Username: "testuser",
					Password: string(hashedPassword),
					Balance:  100,
				}
			},
			expectError:   true,
			expectedError: "invalid credentials",
		},
		{
			name: "error generating token",
			setupMocks: func(mockUserRepo *mock_repository.MockUserRepository, mockJWTService *mock_utils.MockJWTService, username, password string, user *model.User) {
				mockUserRepo.EXPECT().
					GetUserByUsername(gomock.Any(), username).
					Return(user, nil)

				mockJWTService.EXPECT().
					GenerateToken(user.ID).
					Return("", errors.New("token error"))
			},
			username: "testuser",
			password: "testpassword",
			setupUser: func() *model.User {
				userID := uuid.New().String()
				hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("testpassword"), bcrypt.DefaultCost)
				return &model.User{
					ID:       userID,
					Username: "testuser",
					Password: string(hashedPassword),
					Balance:  100,
				}
			},
			expectError:   true,
			expectedError: "invalid credentials",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockUserRepo := mock_repository.NewMockUserRepository(ctrl)
			mockJWTService := mock_utils.NewMockJWTService(ctrl)
			logger, _ := zap.NewDevelopment()

			authService := NewAuthService(AuthServiceParams{
				UserRepo:   mockUserRepo,
				JWTService: mockJWTService,
				Logger:     logger,
			})

			ctx := context.Background()
			user := tc.setupUser()

			tc.setupMocks(mockUserRepo, mockJWTService, tc.username, tc.password, user)

			resultUser, token, err := authService.Login(ctx, tc.username, tc.password)

			if tc.validateResult != nil {
				tc.validateResult(t, resultUser, token, err)
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

func TestAuthService_ValidateToken(t *testing.T) {
	testCases := []struct {
		name           string
		setupMocks     func(*mock_repository.MockUserRepository, *mock_utils.MockJWTService, string)
		tokenString    string
		expectError    bool
		expectedError  string
		validateResult func(*testing.T, string, error)
	}{
		{
			name: "successful token validation",
			setupMocks: func(mockUserRepo *mock_repository.MockUserRepository, mockJWTService *mock_utils.MockJWTService, tokenString string) {
				userID := uuid.New().String()
				claims := &utils.Claims{
					UserID: userID,
					StandardClaims: jwt.StandardClaims{
						ExpiresAt: jwt.At(time.Now().Add(time.Hour)),
						IssuedAt:  jwt.At(time.Now()),
					},
				}

				mockJWTService.EXPECT().
					ValidateToken(tokenString).
					Return(claims, nil)
			},
			tokenString: "test-token",
			expectError: false,
			validateResult: func(t *testing.T, userID string, err error) {
				assert.NoError(t, err)
				assert.NotEmpty(t, userID)
			},
		},
		{
			name: "invalid token",
			setupMocks: func(mockUserRepo *mock_repository.MockUserRepository, mockJWTService *mock_utils.MockJWTService, tokenString string) {
				mockJWTService.EXPECT().
					ValidateToken(tokenString).
					Return(nil, errors.New("invalid token"))
			},
			tokenString:   "test-token",
			expectError:   true,
			expectedError: "invalid token",
		},
		{
			name: "empty user ID in claims",
			setupMocks: func(mockUserRepo *mock_repository.MockUserRepository, mockJWTService *mock_utils.MockJWTService, tokenString string) {
				claims := &utils.Claims{
					UserID: "",
					StandardClaims: jwt.StandardClaims{
						ExpiresAt: jwt.At(time.Now().Add(time.Hour)),
						IssuedAt:  jwt.At(time.Now()),
					},
				}

				mockJWTService.EXPECT().
					ValidateToken(tokenString).
					Return(claims, nil)
			},
			tokenString:   "test-token",
			expectError:   true,
			expectedError: "invalid token",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockUserRepo := mock_repository.NewMockUserRepository(ctrl)
			mockJWTService := mock_utils.NewMockJWTService(ctrl)
			logger, _ := zap.NewDevelopment()

			authService := NewAuthService(AuthServiceParams{
				UserRepo:   mockUserRepo,
				JWTService: mockJWTService,
				Logger:     logger,
			})

			tc.setupMocks(mockUserRepo, mockJWTService, tc.tokenString)

			resultUserID, err := authService.ValidateToken(tc.tokenString)

			if tc.validateResult != nil {
				tc.validateResult(t, resultUserID, err)
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
