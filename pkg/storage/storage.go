package storage

import (
	"context"

	"github.com/Dorrrke/loyality-system.git/pkg/models"
)

type Storage interface {
	InsertUser(ctx context.Context, login string, passHash string) error
	CheckUser(ctx context.Context, login string, passHash string) (bool, error)
	InsertOrder(ctx context.Context, orderNumber string) error
	GetAllOrders(ctx context.Context, userID string) ([]models.Order, error)
	GetUserBalance(ctx context.Context, userID string) (models.Balance, error)
	GetUsersWithdrawls(ctx context.Context, userID string) ([]models.Withdraw, error)
}
