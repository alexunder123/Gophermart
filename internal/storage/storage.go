package storage

import (
	"gophermart/internal/config"
)

type Storager interface {
	AddNewUser(login, password, userID string) error
	LogInUser(login, password string) (string, error)
	CheckUser(userID string) error
	AddNewOrder(userID, orders string) error
	UserWithdraw(userID, order string, sum float32) error
	UserBalance(userID string) ([]byte, error)
	UserOrders(userID string) ([]byte, error)
	UserWithdrawals(userID string) ([]byte, error)
	GetProcessedOrders() ([]ProcessedOrders, error)
	UpdateOrderStatus(AccuralResult) error
	CloseDB()
}

func NewStorage(p *config.Config) Storager {
	return NewSQLStorager(p)
}

type currentBalance struct {
	Current   float32 `json:"current"`
	Withdrawn float32 `json:"withdrawn"`
}

type orders struct {
	Number     string  `json:"number"`
	Status     string  `json:"status"`
	Accrual    float32 `json:"accrual,omitempty"`
	UploadedAt string  `json:"uploaded_at"`
}

type withdraws struct {
	Order       string  `json:"order"`
	Sum         float32 `json:"sum"`
	ProcessedAt string  `json:"processed_at"`
}

type ProcessedOrders struct {
	UserID string
	Order  string
	Status string
}

type AccuralResult struct {
	UserID  string
	Order   string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float32 `json:"accrual"`
}
