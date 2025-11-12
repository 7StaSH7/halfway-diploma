package dto

import (
	"time"

	"github.com/7StaSH7/halfway-diploma/internal/model"
)

type RegisterRequest struct {
	Username string `json:"login" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginRequest struct {
	Username string `json:"login" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type OrderResponse struct {
	Number     string            `json:"number"`
	Status     model.OrderStatus `json:"status"`
	Accrual    float64           `json:"accrual,omitempty"`
	UploadedAt string            `json:"uploaded_at"`
}

func ToOrderResponse(o *model.Order) OrderResponse {
	response := OrderResponse{
		Number:     o.Number,
		Status:     o.Status,
		UploadedAt: o.CreatedAt.Format(time.RFC3339),
	}

	if o.Accrual > 0 {
		response.Accrual = float64(o.Accrual) / 100
	}

	return response
}

type BalanceResponse struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

type WithdrawRequest struct {
	Order string  `json:"order" binding:"required"`
	Sum   float64 `json:"sum" binding:"required,gt=0"`
}

type WithdrawalResponse struct {
	Order       string  `json:"order"`
	Sum         float64 `json:"sum"`
	ProcessedAt string  `json:"processed_at"`
}

func ToWithdrawalResponse(w *model.Withdrawal) WithdrawalResponse {
	return WithdrawalResponse{
		Order:       w.OrderNumber,
		Sum:         float64(w.Sum) / 100,
		ProcessedAt: w.CreatedAt.Format(time.RFC3339),
	}
}
