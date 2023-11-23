package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Dorrrke/loyality-system.git/internal/config"
	"github.com/Dorrrke/loyality-system.git/internal/logger"
	"github.com/Dorrrke/loyality-system.git/pkg/models"
	"github.com/Dorrrke/loyality-system.git/pkg/storage"
	"github.com/Dorrrke/loyality-system.git/pkg/storage/storageErrors"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

const SecretKey = "SecretFurinaNotFokalors333"

type Server struct {
	storage storage.Storage
	Config  config.Config
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

	salt := uuid.New().String()
	pass, err := s.hashPassword(authModel.Password)
	if err != nil {
		logger.Log.Error("Hashing pass error", zap.Error(err))
		http.Error(res, "Внутренняя ошибка серевера", http.StatusInternalServerError)
		return
	}
	authModel.Password = pass

	uid, err := s.saveUser(authModel, salt)
	if err != nil { // После сохранения, нужно достовать uid пользователя
		if errors.Is(err, storageErrors.ErrLoginCOnflict) {
			http.Error(res, "Логин занят", http.StatusConflict)
			return
		}
		logger.Log.Error("Save in db error", zap.Error(err))
		http.Error(res, "Внутренняя ошибка серевера", http.StatusInternalServerError)
		return
	}
	token, err := createJWTToken(uid)
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
		if errors.Is(err, storageErrors.ErrUserNotExists) || err.Error() == "Password does not correct" {
			logger.Log.Error("User not exist", zap.Error(err))
			http.Error(res, "Неверная пара логин/пароль", http.StatusUnauthorized)
			return
		}
		logger.Log.Error("Check info from db error", zap.Error(err))
		http.Error(res, "Внутренняя ошибка серевера", http.StatusInternalServerError)
		return
	}
	token, err := createJWTToken(fmt.Sprint(uid))
	if err != nil {
		logger.Log.Info("cannot create token", zap.Error(err))
	}
	res.Header().Add("Authorization", token)
	res.WriteHeader(http.StatusOK)
}

func (s *Server) UploadOrderHandler(res http.ResponseWriter, req *http.Request) {
	//проверка аунтификации пользователя
	token := req.Header.Get("Authorization")
	userID := getUID(token)
	if userID == "" {
		http.Error(res, "User unauth", http.StatusUnauthorized)
		return
	}
	orderNum, err := io.ReadAll(req.Body)
	if err != nil {
		logger.Log.Error("Read from request error", zap.Error(err))
		http.Error(res, "Неверный формат запроса", http.StatusBadRequest)
		return
	}

	if !orderNumberValid(string(orderNum)) {
		logger.Log.Error("Order number isnt valid")
		http.Error(res, "Неверный формат номера заказа", http.StatusUnprocessableEntity)
		return
	}

	uid, err := s.checkOrder(string(orderNum), userID)
	if err != nil {
		if errors.Is(err, storageErrors.ErrOrderNotExist) {
			logger.Log.Info("UserID", zap.String("UID from toketn", userID), zap.String("UserId from db", uid))
			err = s.uploadOrder(string(orderNum), userID)
			if err != nil {
				logger.Log.Error("Insert order err - ", zap.Error(err))
				http.Error(res, "Внутренняя ошибка серевера", http.StatusInternalServerError)
				return
			}
			go s.getFromAccrualSys(string(orderNum), userID) // TODO асинхронность
			res.WriteHeader(http.StatusAccepted)
			return
		}
		logger.Log.Error("Check order err - ", zap.Error(err))
		http.Error(res, "Внутренняя ошибка серевера", http.StatusInternalServerError)
		return
	}
	if uid == userID {
		http.Error(res, "Номер заказа уже был загружен этим пользователем", http.StatusOK)
		return
	}
	http.Error(res, "Номер заказа уже был загружен другим пользователем", http.StatusConflict)

}

func (s *Server) UnloadHandler(res http.ResponseWriter, req *http.Request) {
	token := req.Header.Get("Authorization")
	userID := getUID(token)
	if userID == "" {
		http.Error(res, "User unauth", http.StatusUnauthorized)
		return
	}
	orders, err := s.getAllOrders(userID)
	if err != nil {
		if errors.Is(err, storageErrors.ErrOrdersNotExist) {
			logger.Log.Error("User havnt orders")
			http.Error(res, "Нет данных для ответа", http.StatusNoContent)
			return
		}
		logger.Log.Error("Error when get order data from db", zap.Error(err))
		http.Error(res, "Внутренняя ошибка серевера", http.StatusInternalServerError)
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
	token := req.Header.Get("Authorization")
	userID := getUID(token)
	if userID == "" {
		http.Error(res, "User unauth", http.StatusUnauthorized)
		return
	}
	balance, err := s.getUserBalance(userID)
	if err != nil {
		http.Error(res, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}
	res.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(res)
	if err := enc.Encode(balance); err != nil {
		logger.Log.Error("Encode orders error", zap.Error(err))
		http.Error(res, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}
	res.WriteHeader(http.StatusOK)

}

func (s *Server) WriteOffBonusHandler(res http.ResponseWriter, req *http.Request) {
	token := req.Header.Get("Authorization")
	userID := getUID(token)
	if userID == "" {
		http.Error(res, "User unauth", http.StatusUnauthorized)
		return
	}

	dec := json.NewDecoder(req.Body)
	var withdraw models.Withdraw

	if err := dec.Decode(&withdraw); err != nil {
		logger.Log.Error("Cannot parse req body", zap.Error(err))
		http.Error(res, "Неверный формат запроса", http.StatusBadRequest)
		return
	}

	if !orderNumberValid(withdraw.Order) {
		http.Error(res, "Неверный номер заказа", http.StatusUnprocessableEntity)
		return
	}
	if err := s.writeOffBonuces(withdraw, userID); err != nil {
		logger.Log.Error("write off error", zap.Error(err))
		http.Error(res, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}
	res.WriteHeader(http.StatusOK)
}

func (s *Server) WriteOffBalanceHistoryHandler(res http.ResponseWriter, req *http.Request) {
	token := req.Header.Get("Authorization")
	userID := getUID(token)
	if userID == "" {
		http.Error(res, "User unauth", http.StatusUnauthorized)
		return
	}
	history, err := s.getWriteOffHistory(userID)
	if err != nil {
		logger.Log.Error("get history error", zap.Error(err))
		http.Error(res, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}
	res.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(res)
	if err := enc.Encode(history); err != nil {
		logger.Log.Error("Encode orders error", zap.Error(err))
		http.Error(res, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}
}

func (s *Server) getFromAccrualSys(orderNumber string, userID string) error {
	client := &http.Client{}
	response, err := client.Get("http://" + s.Config.AccrualConfig.String() + "/api/orders/" + orderNumber)
	if err != nil {
		logger.Log.Error("Accrual sys responce error", zap.Error(err))
		return err
	}
	defer response.Body.Close()
	statusCode := response.StatusCode
	if statusCode == 200 {
		logger.Log.Info("Accrual", zap.Int("StatusCode", statusCode))
		var accrualModel models.AccrualModel
		dec := json.NewDecoder(response.Body)
		if err := dec.Decode(&accrualModel); err != nil {
			logger.Log.Error("Cannot parse req body", zap.Error(err))
			return err
		}
		logger.Log.Info("Acrrual sys responce:",
			zap.String("Order", accrualModel.OrderNumber),
			zap.Float32("Accrual", accrualModel.Accrual),
			zap.String("Status", accrualModel.Status))
		logger.Log.Info("1 User ID", zap.String("UID", userID))
		if err := s.updateOrderAndBalance(accrualModel, userID); err != nil {
			logger.Log.Error("Accrual db update Error", zap.Error(err))
			return err
		}
		return nil
	}
	if statusCode == 204 {
		logger.Log.Info("Accrual", zap.Int("StatusCode", statusCode))
		return errors.New("Заказ не зарегестрирован")
	}
	if statusCode == 429 {
		logger.Log.Info("Accrual", zap.Int("StatusCode", statusCode))
		return errors.New("Превышено количество запросов")
	}
	log.Print(statusCode)
	return nil
}

func (s *Server) saveUser(user models.AuthModel, salt string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	uid, err := s.storage.InsertUser(ctx, user.Login, user.Password)
	if err != nil {
		return "", err
	}

	return uid, nil
}

func (s *Server) getUser(user models.AuthModel) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	uid, pass, err := s.storage.GetUserByLogin(ctx, user.Login, user.Password)
	if err != nil {
		return -1, err
	}
	if !s.matchPasswords(user.Password, pass) {
		return -1, errors.New("Password does not correct")
	}

	return uid, nil
}

func (s *Server) getUserBalance(userID string) (models.Balance, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	uid, err := strconv.Atoi(userID)
	if err != nil {
		logger.Log.Error("str to int err", zap.Error(err))
		return models.Balance{Current: 0,
			Withdraw: 0}, err
	}
	balance, err := s.storage.GetUserBalance(ctx, uid)
	if err != nil {
		return models.Balance{Current: 0,
			Withdraw: 0}, err
	}
	return balance, nil
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

func (s *Server) getWriteOffHistory(userID string) ([]models.WithdrawInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	uid, err := strconv.Atoi(userID)
	if err != nil {
		logger.Log.Error("str to int err", zap.Error(err))
		return nil, err
	}

	history, err := s.storage.GetUsersWithdrawls(ctx, uid)
	if err != nil {
		return nil, err
	}
	return history, nil
}

func (s *Server) updateOrderAndBalance(accrual models.AccrualModel, userID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	logger.Log.Info("2 User ID", zap.String("UID", userID))

	err := s.storage.UpdateByAccrual(ctx, accrual, userID)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) writeOffBonuces(withdraw models.Withdraw, userID string) error {
	balance, err := s.getUserBalance(userID)
	if err != nil {
		return err
	}

	if balance.Current-withdraw.Sum < 0 {
		return errors.New("insufficient fund")
	}

	uid, err := strconv.Atoi(userID)
	if err != nil {
		logger.Log.Error("str to int err", zap.Error(err))
		return err
	}
	current := balance.Current - withdraw.Sum

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = s.storage.InsertWriteOffBonuces(ctx, withdraw, current, uid)
	if err != nil {
		return err
	}
	return nil
}

func orderNumberValid(number string) bool {
	digits := strings.Split(strings.ReplaceAll(number, " ", ""), "")
	lengthOfString := len(digits)

	if lengthOfString < 2 {
		return false
	}

	sum := 0
	flag := false

	for i := lengthOfString - 1; i > -1; i-- {
		digit, _ := strconv.Atoi(digits[i])

		if flag {
			digit *= 2

			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
		flag = !flag
	}

	return math.Mod(float64(sum), 10) == 0
}

func createJWTToken(uuid string) (string, error) {
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

func getUID(tokenString string) string {
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

func (s *Server) hashPassword(pass string) (string, error) {

	hashedPassword, err := bcrypt.
		GenerateFromPassword([]byte(pass), bcrypt.MinCost)
	if err != nil {
		return ``, err
	}

	return string(hashedPassword), nil
}

func (s *Server) matchPasswords(currentPass string, passFromDB string) bool {
	{
		err := bcrypt.CompareHashAndPassword(
			[]byte(passFromDB), []byte(currentPass))
		return err == nil
	}
}

func (s *Server) checkOrder(order string, uid string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID, err := s.storage.CheckOrder(ctx, order)
	if err != nil {
		return "", err
	}
	return userID, nil
}

func (s *Server) uploadOrder(order string, uid string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := s.storage.InsertOrder(ctx, uid, order)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) CreateTable() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.storage.CreateTables(ctx); err != nil {
		return err
	}
	return nil
}

func (s *Server) ConnStorage(stor storage.Storage) {
	s.storage = stor
}
