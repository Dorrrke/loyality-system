package server

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/Dorrrke/loyality-system.git/internal/logger"
	"github.com/Dorrrke/loyality-system.git/pkg/models"
	"github.com/Dorrrke/loyality-system.git/pkg/storage"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

type Server struct {
	storage storage.Storage
	// config  string
}

func (s *Server) RegisterHandler(res http.ResponseWriter, req *http.Request) {
	dec := json.NewDecoder(req.Body)
	var authModel models.AuthModel

	if err := dec.Decode(&authModel); err != nil {
		logger.Log.Error("Cannot parse req body", zap.Error(err))
		http.Error(res, "Неверный формат запроса", http.StatusBadRequest)
		return
	}

	if err := s.SaveUser(authModel); err != nil { // После сохранения, нужно достовать uid пользователя
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgerrcode.IsIntegrityConstraintViolation(pgErr.Code) {
				logger.Log.Info("register error, login is alredy exists")
				http.Error(res, "Логин занят", http.StatusConflict)
				return
			}
		}
		logger.Log.Error("Save in db error", zap.Error(err))
		http.Error(res, "Внутренняя ошибка серевера", http.StatusInternalServerError)
		return
	}

	// Так же надо сразу логинить пользователя, тобишь отправлть куку или в Header поле заносить userID
	res.WriteHeader(http.StatusOK)

}

func (s *Server) LoginHandler(res http.ResponseWriter, req *http.Request) {

	dec := json.NewDecoder(req.Body)
	var authModel models.AuthModel

	if err := dec.Decode(&authModel); err != nil {
		logger.Log.Error("Cannot parse req body", zap.Error(err))
		http.Error(res, "Неверный формат запроса", http.StatusBadRequest)
		return
	}

	check, err := s.CheckUser(authModel)
	if err != nil {
		logger.Log.Error("Check info from db error", zap.Error(err))
		http.Error(res, "Внутренняя ошибка серевера", http.StatusInternalServerError)
		return
	}
	if !check {
		logger.Log.Info("Login&Password incorrect")
		http.Error(res, "Неверная пара логин/пароль", http.StatusUnauthorized)
		return
	}
	// надо логинить пользователя, тобишь отправлть куку или в Header поле заносить userID
	res.WriteHeader(http.StatusOK)
}

func (s *Server) UploadOrderHandler(res http.ResponseWriter, req *http.Request) {
	//проверка аунтификации пользователя
	orderNum, err := io.ReadAll(req.Body)
	if err != nil {
		logger.Log.Error("Read from request error", zap.Error(err))
		http.Error(res, "Неверный формат запроса", http.StatusBadRequest)
	}

	if !Valid(orderNum) {
		logger.Log.Error("Order number isnt valid")
		http.Error(res, "Неверный формат номера заказа", http.StatusUnprocessableEntity)
		return
	}

}

func (s *Server) UnloadHandler(res http.ResponseWriter, req *http.Request) {
	userID := "444" // Будет браться из куки или хеддера если юзер залогинин
	orders, err := s.getAllOrders(userID)
	if err != nil {
		logger.Log.Error("Error when get order data from db")
		http.Error(res, "Внутренняя ошибка серевера", http.StatusInternalServerError)
		return
	}
	if len(orders) == 0 {
		logger.Log.Error("User havnt orders")
		http.Error(res, "Нет данных для ответа", http.StatusNoContent)
		return
	}
	res.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(res)
	if err := enc.Encode(orders); err != nil {
		logger.Log.Error("Encode orders error", zap.Error(err))
		http.Error(res, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}
	res.WriteHeader(http.StatusOK)
}

func (s *Server) GetBalanceHandler(res http.ResponseWriter, req *http.Request) {

}

func (s *Server) WriteOffBonusHandler(res http.ResponseWriter, req *http.Request) {

}

func (s *Server) WriteOffBalanceHistoryHandler(res http.ResponseWriter, req *http.Request) {

}

func (s *Server) SaveUser(user models.AuthModel) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.storage.InsertUser(ctx, user.Login, user.Password); err != nil {
		return err
	}

	return nil
}

func (s *Server) CheckUser(user models.AuthModel) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	check, err := s.storage.CheckUser(ctx, user.Login, user.Password)
	if err != nil {
		return false, err
	}
	return check, nil
}

func (s *Server) getAllOrders(userID string) ([]models.Order, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	orders, err := s.storage.GetAllOrders(ctx, userID)
	if err != nil {
		return nil, err
	}

	return orders, nil
}

// Алгоритм Луна
func Valid(data []byte) bool {
	number := int(binary.BigEndian.Uint64(data))
	return (number%10+checksum(number/10))%10 == 0
}

func checksum(number int) int {
	var luhn int

	for i := 0; number > 0; i++ {
		cur := number % 10

		if i%2 == 0 { // even
			cur = cur * 2
			if cur > 9 {
				cur = cur%10 + cur/10
			}
		}

		luhn += cur
		number = number / 10
	}
	return luhn % 10
}
