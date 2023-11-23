package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"

	"github.com/Dorrrke/loyality-system.git/internal/logger"
	"github.com/Dorrrke/loyality-system.git/pkg/server"
	"github.com/Dorrrke/loyality-system.git/pkg/storage"
	"github.com/caarlos0/env"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func main() {

	log.Println("Init logger")
	if err := logger.Initialize(zap.InfoLevel.String()); err != nil {
		log.Println("init logger error" + err.Error())
		log.Println("Panic logger")
		panic(err)
	}
	var s server.Server
	var DBaddr string

	flag.Var(&s.Config.HostConfig, "a", "address and port to run server")
	flag.StringVar(&DBaddr, "d", "", "databse addr")
	flag.Var(&s.Config.AccrualConfig, "r", "address and port accrual")

	flag.Parse()
	servErr := env.Parse(&s.Config.EnvValues.ServerCfg)
	if servErr == nil {
		s.Config.HostConfig.Set(s.Config.EnvValues.ServerCfg.Addr)
	}

	dbDsnErr := env.Parse(&s.Config.EnvValues.DataBaseDsn)
	if dbDsnErr == nil {
		log.Println("DB env str" + s.Config.EnvValues.DataBaseDsn.DBDSN)
		conn := initDB(s.Config.EnvValues.DataBaseDsn.DBDSN)
		s.ConnStorage(&storage.DataBaseStorage{DB: conn})
		defer conn.Close()
	}
	if dbDsnErr != nil {
		if DBaddr != "" {
			log.Println("DB flag str" + DBaddr)
			conn := initDB(DBaddr)
			s.ConnStorage(&storage.DataBaseStorage{DB: conn})
			defer conn.Close()
		}
	}
	accrualErr := env.Parse(&s.Config.EnvValues.AccrualCfg)
	if accrualErr == nil {
		s.Config.HostConfig.Set(s.Config.EnvValues.AccrualCfg.AccrualAddr)
	}
	if s.Config.EnvValues.DataBaseDsn.DBDSN == "" && DBaddr == "" {
		log.Println("Error init db")
		log.Println("DB env str" + s.Config.EnvValues.DataBaseDsn.DBDSN)
		log.Println("DB flag str" + DBaddr)
		panic(errors.New("Not init db"))
	}
	go func() {
		if err := s.CreateTable(); err != nil {
			logger.Log.Error("Error create tables", zap.Error(err))
		}
	}()
	err := run(s)
	if err != nil {
		logger.Log.Error("Run server error", zap.Error(err))
		log.Println("Panic run")
		panic(err)
	}
}

func run(s server.Server) error {

	r := chi.NewRouter()

	r.Route("/api/user", func(r chi.Router) {
		r.Post("/register", logger.WithLog(s.RegisterHandler))
		r.Post("/login", logger.WithLog(s.LoginHandler))
		r.Post("/orders", logger.WithLog(s.UploadOrderHandler))
		r.Get("/orders", logger.WithLog(s.UnloadHandler))
		r.Route("/balance", func(r chi.Router) {
			r.Get("/", logger.WithLog(s.GetBalanceHandler))
			r.Post("/withdraw", logger.WithLog(s.WriteOffBonusHandler))
		})
		r.Get("/withdrawals", logger.WithLog(s.WriteOffBalanceHistoryHandler))
	})
	if s.Config.HostConfig.String() != "" && s.Config.HostConfig.String() != ":0" {
		logger.Log.Info("Run Server on", zap.String("Server addr from flag", s.Config.HostConfig.String()))
		return http.ListenAndServe(s.Config.HostConfig.String(), r)
	}
	if s.Config.EnvValues.ServerCfg.Addr != "" {
		logger.Log.Info("Run Server on", zap.String("Server addr from env", s.Config.EnvValues.ServerCfg.Addr))
		return http.ListenAndServe(s.Config.EnvValues.ServerCfg.Addr, r)
	}
	logger.Log.Info("Run server on", zap.String("Server addr default", "localhost:8080"))
	return http.ListenAndServe(":8080", r)
}

func initDB(DBAddr string) *pgxpool.Pool {
	pool, err := pgxpool.New(context.Background(), DBAddr)
	if err != nil {
		logger.Log.Error("Error wile init db driver: " + err.Error())
		log.Println("Panic db")
		panic(err)
	}
	return pool

}
