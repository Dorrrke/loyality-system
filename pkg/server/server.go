package server

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Dorrrke/loyality-system.git/internal/logger"
	"github.com/Dorrrke/loyality-system.git/pkg/models"
	"github.com/Dorrrke/loyality-system.git/pkg/storage"
	"github.com/golang-jwt/jwt/v4"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const SecretKey = "SecretFurinaNotFokalors333"

type Server struct {
	storage storage.Storage
	// config  string
}

type Claims struct {
	jwt.RegisteredClaims
	UserID string
}

func (s *Server) RegisterHandler(res http.ResponseWriter, req *http.Request) {
	dec := json.NewDecoder(req.Body)
	var authModel models.AuthModel

	if err := dec.Decode(&authModel); err != nil {
		logger.Log.Error("Cannot parse req body", zap.Error(err))
		http.Error(res, "Неверный формат запроса", http.StatusBadRequest)
		return
	}

	uid, err := s.SaveUser(authModel)
	if err != nil { // После сохранения, нужно достовать uid пользователя
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
	token, err := CreateJWTToken(uid)
	if err != nil {
		logger.Log.Info("cannot create token", zap.Error(err))
	}
	res.Header().Add("Authorization", token)

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

	uid, err := s.getUser(authModel)
	if err != nil {
		logger.Log.Error("Check info from db error", zap.Error(err))
		http.Error(res, "Внутренняя ошибка серевера", http.StatusInternalServerError)
		return
	}
	if uid == -1 {
		logger.Log.Info("User not exist")
		http.Error(res, "Неверная пара логин/пароль", http.StatusUnauthorized)
		return
	}
	token, err := CreateJWTToken(fmt.Sprint(uid))
	if err != nil {
		logger.Log.Info("cannot create token", zap.Error(err))
	}
	res.Header().Add("Authorization", token)
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

func (s *Server) SaveUser(user models.AuthModel) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	uid, err := s.storage.InsertUser(ctx, user.Login, user.Password)
	if err != nil {
		return "", err
	}

	return uid, nil
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

func (s *Server) getUser(user models.AuthModel) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	uid, err := s.storage.GetUserForLoginAndPass(ctx, user.Login, user.Password)
	if err != nil {
		return -1, errors.Wrap(err, "Get user for login/pass error")
	}
	if uid != 0 {
		return uid, nil
	}
	return -1, nil
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

func CreateJWTToken(uuid string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 3)),
		},
		UserID: uuid,
	})

	tokenString, err := token.SignedString([]byte(SecretKey))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func GetUID(tokenString string) string {
	claim := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claim, func(t *jwt.Token) (interface{}, error) {
		return []byte(SecretKey), nil
	})
	if err != nil {
		return ""
	}

	if !token.Valid {
		return ""
	}
	logger.Log.Info("Decode token:", zap.String("UserID:", claim.UserID))
	return claim.UserID
}
