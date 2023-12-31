package storage

import (
	"context"
	"embed"
	"strconv"
	"strings"
	"time"

	"github.com/Dorrrke/loyality-system.git/internal/logger"
	"github.com/Dorrrke/loyality-system.git/pkg/models"
	"github.com/Dorrrke/loyality-system.git/pkg/storage/errorsstorage"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type Storage interface {
	InsertUser(ctx context.Context, login string, passHash string) (string, error)
	CheckUser(ctx context.Context, login string, passHash string) (bool, error)
	InsertOrder(ctx context.Context, uuid string, orderNumber string) error
	GetAllOrders(ctx context.Context, userID string) ([]models.Order, error)
	GetUserBalance(ctx context.Context, userID int) (models.Balance, error)
	GetUsersWithdrawls(ctx context.Context, userID int) ([]models.WithdrawInfo, error)
	InsertWriteOffBonuces(ctx context.Context, withdraw models.Withdraw, current float32, userID int) error
	GetUserByLogin(ctx context.Context, login string, password string) (int, string, error)
	CheckOrder(ctx context.Context, order string) (string, error)
	UpdateByAccrual(ctx context.Context, accrual models.AccrualModel, userID string) error
	CreateTables(ctx context.Context) error
	ClearTables(ctx context.Context) error
	GetNoTerminateOrders(ctx context.Context) ([]string, error)
	MigrationUp(migrationPath string, dbURL string) error
}

type DataBaseStorage struct {
	DB *pgxpool.Pool
}

func (db *DataBaseStorage) InsertUser(ctx context.Context, login string, passHash string) (string, error) {
	row := db.DB.QueryRow(ctx, "insert into users (login, password) values ($1, $2) RETURNING uid;", login, passHash)
	var uuid string
	if err := row.Scan(&uuid); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgerrcode.IsIntegrityConstraintViolation(pgErr.Code) {
				logger.Log.Error("Register error", zap.Error(errorsstorage.ErrLoginCOnflict))
				return ``, errorsstorage.ErrLoginCOnflict
			}
			return "", err
		}
		return "", err
	}
	userID, err := strconv.Atoi(uuid)
	if err != nil {
		logger.Log.Error("str to int err", zap.Error(err))
	}
	_, err = db.DB.Exec(ctx, "insert into user_balance (uid, current, withdrawn) values ($1, 0, 0)", userID)
	if err != nil {
		return ``, errors.Wrap(err, "Insert user balance error")
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
func (db *DataBaseStorage) GetUserByLogin(ctx context.Context, login string, password string) (int, string, error) {
	row := db.DB.QueryRow(ctx, "Select uid, password FROM users where login = $1", login)
	var (
		uid  int
		pass string
	)
	if err := row.Scan(&uid, &pass); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return -1, ``, errorsstorage.ErrUserNotExists
		}
		return -1, ``, errors.Wrap(err, "Error parsing db info")
	}

	return uid, pass, nil
}
func (db *DataBaseStorage) InsertOrder(ctx context.Context, uuid string, orderNumber string) error {
	_, err := db.DB.Exec(ctx, "insert into orders (uid, number, status, accrual) values ($1, $2, $3, 0)", uuid, orderNumber, "NEW")
	if err != nil {
		return errors.Wrap(err, "Insert order error")
	}
	return nil
}
func (db *DataBaseStorage) CheckOrder(ctx context.Context, order string) (string, error) {
	row := db.DB.QueryRow(ctx, "select uid from orders where number = $1", order)
	var uid string
	if err := row.Scan(&uid); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", errorsstorage.ErrOrderNotExist
		}
		return "", errors.Wrap(err, "Scan row error")
	}
	return uid, nil
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
		var date time.Time
		if err := rows.Scan(&order.Number, &order.Status, &order.Accrual, &date); err != nil {
			return nil, errors.Wrap(err, "Parsing get order db info error")
		}
		order.Number = strings.TrimSpace(order.Number)
		order.Status = strings.TrimSpace(order.Status)
		order.UploadedAt = date.Format(time.RFC3339)
		orders = append(orders, order)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	if len(orders) == 0 {
		return nil, errorsstorage.ErrOrdersNotExist
	}

	return orders, nil

}
func (db *DataBaseStorage) GetUserBalance(ctx context.Context, userID int) (models.Balance, error) {
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
func (db *DataBaseStorage) GetUsersWithdrawls(ctx context.Context, userID int) ([]models.WithdrawInfo, error) {
	rows, err := db.DB.Query(ctx, `select "order", sum, processed_at from withdrawals where uid = $1 order by processed_at`, userID)
	if err != nil {
		return nil, errors.Wrap(err, "Get withdrawls history error")
	}
	defer rows.Close()
	var withdrawls []models.WithdrawInfo

	for rows.Next() {
		var withdraw models.WithdrawInfo
		var date time.Time
		if err := rows.Scan(&withdraw.Order, &withdraw.Sum, &date); err != nil {
			return nil, errors.Wrap(err, "Parsing withdrawls info error")
		}
		withdraw.ProcessedAt = date.Format(time.RFC3339)
		withdraw.Order = strings.TrimSpace(withdraw.Order)
		withdrawls = append(withdrawls, withdraw)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	if len(withdrawls) == 0 {
		return nil, errorsstorage.ErrWriteOffNotExist
	}

	return withdrawls, nil
}
func (db *DataBaseStorage) InsertWriteOffBonuces(ctx context.Context, withdraw models.Withdraw, current float32, userID int) error {

	tx, err := db.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Prepare(ctx, "update user balance", "update user_balance set current = $1, withdrawn = $2 where uid = $3"); err != nil {
		return err
	}
	if _, err := tx.Prepare(ctx, "update history", `insert into withdrawals ("order", sum, uid) values ($1, $2, $3)`); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, "update user balance", current, withdraw.Sum, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, "update history", withdraw.Order, withdraw.Sum, userID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
func (db *DataBaseStorage) UpdateByAccrual(ctx context.Context, accrual models.AccrualModel, userID string) error {
	tx, err := db.DB.Begin(ctx)
	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)

	if _, err := tx.Prepare(ctx, "update order", "update orders set status = $1, accrual = $2 where number = $3"); err != nil {
		return err
	}
	if _, err := tx.Prepare(ctx, "update balance", "update user_balance set current = $1 where uid = $2"); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, "update order", accrual.Status, accrual.Accrual, accrual.OrderNumber); err != nil {
		return err
	}
	uid, err := strconv.Atoi(userID)
	if err != nil {
		logger.Log.Error("str to int err", zap.Error(err))
	}
	logger.Log.Info("UserID", zap.String("UserId str", userID), zap.Int("UserId int", uid))
	if _, err := tx.Exec(ctx, "update balance", accrual.Accrual, uid); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (db *DataBaseStorage) CreateTables(ctx context.Context) error {
	tx, err := db.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS users
	(
			uid serial PRIMARY KEY,
			login character(255) NOT NULL,
			password character(64) NOT NULL
	)`)
	if err != nil {
		return errors.Wrap(err, "users table err")
	}

	_, err = tx.Exec(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS login_id ON users (login)`)
	if err != nil {
		return errors.Wrap(err, "users index err")
	}

	_, err = tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS user_balance
	(
		id serial PRIMARY KEY,
		uid integer NOT NULL,
		current numeric(5,2) NOT NULL,
		withdrawn numeric(5,2) NOT NULL,
		FOREIGN KEY (uid) REFERENCES users (uid) ON UPDATE CASCADE ON DELETE CASCADE
	)`)
	if err != nil {
		return errors.Wrap(err, "users_balance table err")
	}
	_, err = tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS orders
	(
		id serial PRIMARY KEY,
		"number" character(55) NOT NULL,
		status character(125),
		accrual numeric(5,2),
		date timestamp with time zone NOT NULL DEFAULT now(),
		uid integer NOT NULL DEFAULT 1,
		FOREIGN KEY (uid) REFERENCES users (uid) ON UPDATE CASCADE ON DELETE CASCADE
	)`)
	if err != nil {
		return errors.Wrap(err, "orders table err")
	}

	_, err = tx.Exec(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS order_id ON orders (number)`)
	if err != nil {
		return errors.Wrap(err, "orders table index err")
	}

	_, err = tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS withdrawals
	(
		w_id serial PRIMARY KEY,
		"order" character(255) NOT NULL,
		sum numeric(5,2) NOT NULL,
		processed_at timestamp with time zone NOT NULL DEFAULT now(),
		uid integer NOT NULL,
		FOREIGN KEY (uid) REFERENCES users (uid) ON UPDATE CASCADE ON DELETE CASCADE
	)`)
	if err != nil {
		return errors.Wrap(err, "withdrawals table err")
	}
	_, err = tx.Exec(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS order_id ON orders (number)`)
	if err != nil {
		return errors.Wrap(err, "withdrawals table index err")
	}
	return tx.Commit(ctx)
}

func (db *DataBaseStorage) ClearTables(ctx context.Context) error {
	tx, err := db.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `DELETE FROM users`)
	if err != nil {
		return errors.Wrap(err, "users table err")
	}

	_, err = tx.Exec(ctx, `DELETE FROM user_balance`)
	if err != nil {
		return errors.Wrap(err, "users_balance table err")
	}
	_, err = tx.Exec(ctx, `DELETE FROM orders`)
	if err != nil {
		return errors.Wrap(err, "orders table err")
	}

	_, err = tx.Exec(ctx, `DELETE FROM withdrawals`)
	if err != nil {
		return errors.Wrap(err, "withdrawals table err")
	}
	return tx.Commit(ctx)
}

func (db *DataBaseStorage) MigrationUp(migrationPath string, dbURL string) error {
	var fs embed.FS
	d, err := iofs.New(fs, migrationPath)
	if err != nil {
		logger.Log.Error("Migration iofs error:", zap.Error(err))
	}
	m, err := migrate.NewWithSourceInstance("iofs", d, dbURL)
	if err != nil {
		return err
	}

	err = m.Up()
	if err != nil {
		if errors.As(err, migrate.ErrNoChange) {
			logger.Log.Info("Migration Data Base No Change")
			return nil
		}
		return err
	}

	logger.Log.Info("Migration applied succsessfully")
	return nil
}

func (db *DataBaseStorage) GetNoTerminateOrders(ctx context.Context) ([]string, error) {
	row, err := db.DB.Query(ctx, `SELECT number FROM orders WHERE status != 'INVALID' or status !='PROCESSED'`)
	if err != nil {
		return nil, errors.Wrap(err, "Get orders error")
	}
	defer row.Close()
	var orderNumbers []string
	for row.Next() {
		var orderNumber string
		if err := row.Scan(&orderNumber); err != nil {
			return nil, errors.Wrap(err, "Parsing withdrawls info error")
		}
		orderNumbers = append(orderNumbers, orderNumber)
	}
	err = row.Err()
	if err != nil {
		return nil, err
	}

	if len(orderNumbers) == 0 {
		return nil, errorsstorage.ErrOrderNotExist
	}

	return orderNumbers, nil
}
