package model

import (
	"time"
)

type OrderStatus string

const (
	OrderStatusNew        OrderStatus = "NEW"
	OrderStatusProcessing OrderStatus = "PROCESSING"
	OrderStatusInvalid    OrderStatus = "INVALID"
	OrderStatusProcessed  OrderStatus = "PROCESSED"
)

type Order struct {
	ID        string      `json:"id" db:"id"`
	UserID    string      `json:"user_id" db:"user_id"`
	Number    string      `json:"number" db:"number"`
	Status    OrderStatus `json:"status" db:"status"`
	Accrual   uint        `json:"accrual" db:"accrual"`
	CreatedAt time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt time.Time   `json:"updated_at" db:"updated_at"`
}
