package accrual

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/7StaSH7/halfway-diploma/internal/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var Module = fx.Module("accrual_client",
	fx.Provide(NewAccrualClient),
)

type AccrualClient interface {
	GetOrderAccrual(ctx context.Context, orderNumber string) (*AccrualResponse, error)
}

type accrualClient struct {
	httpClient *http.Client
	baseURL    string
	logger     *zap.Logger
}

type AccrualClientParams struct {
	fx.In
	Config *config.ServerConfig
	Logger *zap.Logger
}

func NewAccrualClient(p AccrualClientParams) AccrualClient {
	return &accrualClient{
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		baseURL: p.Config.AccuralSystemAddress,
		logger:  p.Logger,
	}
}

type AccrualResponse struct {
	Order   string `json:"order"`
	Status  string `json:"status"`
	Accrual *uint  `json:"accrual,omitempty"`
}

func (c *accrualClient) GetOrderAccrual(ctx context.Context, orderNumber string) (*AccrualResponse, error) {
	url := fmt.Sprintf("%s/api/orders/%s", c.baseURL, orderNumber)

	c.logger.Debug("making request to accrual service",
		zap.String("url", url),
		zap.String("order_number", orderNumber))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		c.logger.Error("failed to create request",
			zap.Error(err),
			zap.String("url", url))
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("failed to make request to accrual service",
			zap.Error(err),
			zap.String("url", url))
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNoContent:
		c.logger.Debug("no accrual data available yet",
			zap.String("order_number", orderNumber),
			zap.Int("status_code", resp.StatusCode))
		return nil, nil
	case http.StatusTooManyRequests:
		c.logger.Warn("rate limited by accrual service",
			zap.String("order_number", orderNumber),
			zap.Int("status_code", resp.StatusCode))
		return nil, fmt.Errorf("rate limited by accrual service")
	default:
		c.logger.Error("unexpected response from accrual service",
			zap.String("order_number", orderNumber),
			zap.Int("status_code", resp.StatusCode))
		return nil, fmt.Errorf("unexpected response status: %d", resp.StatusCode)
	}

	var accrualResp AccrualResponse
	if err := json.NewDecoder(resp.Body).Decode(&accrualResp); err != nil {
		c.logger.Error("failed to decode accrual response",
			zap.Error(err),
			zap.String("order_number", orderNumber))
		return nil, err
	}

	c.logger.Debug("received accrual data",
		zap.String("order_number", orderNumber),
		zap.Any("response", accrualResp))

	return &accrualResp, nil
}
