package storage

import (
	"context"
	"strings"

	"github.com/Dorrrke/loyality-system.git/pkg/models"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
)

type Storage interface {
	InsertUser(ctx context.Context, login string, passHash string) (string, error)
	CheckUser(ctx context.Context, login string, passHash string) (bool, error)
	InsertOrder(ctx context.Context, orderNumber string) error
	GetAllOrders(ctx context.Context, userID string) ([]models.Order, error)
	GetUserBalance(ctx context.Context, userID string) (models.Balance, error)
	GetUsersWithdrawls(ctx context.Context, userID string) ([]models.Withdraw, error)
	InsertWriteOffBonuces(ctx context.Context, orderNumber string, sum string) error
	GetUserForLoginAndPass(ctx context.Context, login string, password string) (int, error)
}

type DataBaseStorage struct {
	DB *pgxpool.Pool
}

func (db *DataBaseStorage) InsertUser(ctx context.Context, login string, passHash string) (string, error) {
	row := db.DB.QueryRow(ctx, "insert into users (login, password) values ($1, $2) RETURNING uid;", login, passHash)
	var uuid string
	if err := row.Scan(&uuid); err != nil {
		return "", err
	}
	return uuid, nil
}
func (db *DataBaseStorage) CheckUser(ctx context.Context, login string, passHash string) (bool, error) {
	row := db.DB.QueryRow(ctx, "Select Exists(select * from users where login = $1 and password = $2)", login, passHash)
	var exists bool
	if err := row.Scan(&exists); err != nil {
		return false, errors.Wrap(err, "Error parsing db info")
	}
	return exists, nil
}
func (db *DataBaseStorage) GetUserForLoginAndPass(ctx context.Context, login string, password string) (int, error) {
	row := db.DB.QueryRow(ctx, "Select uid FROM users where login = $1 and password = $2", login, password)
	var uid int
	if err := row.Scan(&uid); err != nil {
		return -1, err
	}
	return uid, nil
}
func (db *DataBaseStorage) InsertOrder(ctx context.Context, uuid string, orderNumber string) error {
	_, err := db.DB.Exec(ctx, "insert into orders (uid, number) values ($1, $2)", uuid, orderNumber)
	if err != nil {
		return errors.Wrap(err, "Insert order error")
	}
	return nil

}
func (db *DataBaseStorage) GetAllOrders(ctx context.Context, userID string) ([]models.Order, error) {
	rows, err := db.DB.Query(ctx, "select number, status, accrual, date from orders where uid = $1 order by date", userID)
	if err != nil {
		return nil, errors.Wrap(err, "Get orders error")
	}
	defer rows.Close()
	var orders []models.Order

	for rows.Next() {
		var order models.Order
		if err := rows.Scan(&order.Number, &order.Status, &order.Accrual, &order.UploadedAt); err != nil {
			return nil, errors.Wrap(err, "Parsing get order db info error")
		}
		order.Number = strings.TrimSpace(order.Number)
		order.Status = strings.TrimSpace(order.Status)
		orders = append(orders, order)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return orders, nil

}
func (db *DataBaseStorage) GetUserBalance(ctx context.Context, userID string) (models.Balance, error) {
	row := db.DB.QueryRow(ctx, "select current, withdrawn from user_balance where uid = $1", userID)
	var balance models.Balance
	if err := row.Scan(&balance.Current, &balance.Withdraw); err != nil {
		return models.Balance{
			Current:  0,
			Withdraw: 0,
		}, err
	}
	return balance, nil

}
func (db *DataBaseStorage) GetUsersWithdrawls(ctx context.Context, userID string) ([]models.Withdraw, error) {
	rows, err := db.DB.Query(ctx, "select number, sum, processed_at from withdrawals LEFT JOIN orders on withdrawals.order_id = orders.id whereuid = $1 order by processed_at", userID)
	if err != nil {
		return nil, errors.Wrap(err, "Get withdrawls history error")
	}
	defer rows.Close()
	var withdrawls []models.Withdraw

	for rows.Next() {
		var withdraw models.Withdraw
		if err := rows.Scan(&withdraw.Order, &withdraw.Sum, &withdraw.ProcessedAt); err != nil {
			return nil, errors.Wrap(err, "Parsing withdrawls info error")
		}
		withdraw.Order = strings.TrimSpace(withdraw.Order)
		withdrawls = append(withdrawls, withdraw)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return withdrawls, nil
}
func (db *DataBaseStorage) InsertWriteOffBonuces(ctx context.Context, orderNumber string, sum string) error {
	_, err := db.DB.Exec(ctx, "insert into withdrawals (order_id, sum) values ((select id from orders where number = $1), $2)", orderNumber, sum)
	if err != nil {
		return errors.Wrap(err, "Error insert withdrawals")
	}
	return nil
}
