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
	Accrual    uint              `json:"accrual"`
	UploadedAt string            `json:"uploaded_at"`
}

func ToOrderResponse(o *model.Order) OrderResponse {
	return OrderResponse{
		Number:     o.Number,
		Status:     o.Status,
		Accrual:    o.Accrual,
		UploadedAt: o.CreatedAt.Format(time.RFC3339),
	}
}
