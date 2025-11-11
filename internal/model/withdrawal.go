package model

import (
	"time"
)

type Withdrawal struct {
	ID          string    `json:"id" db:"id"`
	UserID      string    `json:"user_id" db:"user_id"`
	OrderNumber string    `json:"order_number" db:"order_number"`
	Sum         uint     `json:"sum" db:"sum"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}
