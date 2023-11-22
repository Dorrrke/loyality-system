package main

import (
	"context"
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

	if err := logger.Initialize(zap.InfoLevel.String()); err != nil {
		log.Println("init logger error" + err.Error())
		panic(err)
	}
	var s server.Server
	var DBaddr string

	flag.Var(&s.Config.HostConfig, "a", "address and port to run server")
	flag.Var(&s.Config.AccrualConfig, "r", "address and port accrual")
	flag.StringVar(&DBaddr, "d", "", "databse addr")

	flag.Parse()
	servErr := env.Parse(&s.Config.EnvValues.ServerCfg)
	if servErr == nil {
		s.Config.HostConfig.Set(s.Config.EnvValues.ServerCfg.Addr)
	}

	dbDsnErr := env.Parse(&s.Config.EnvValues.DataBaseDsn)
	if dbDsnErr == nil {
		conn := initDB(s.Config.EnvValues.DataBaseDsn.DBDSN)
		s.ConnStorage(&storage.DataBaseStorage{DB: conn})
		defer conn.Close()
	}
	accrualErr := env.Parse(&s.Config.EnvValues.AccrualCfg)
	if accrualErr == nil {
		s.Config.HostConfig.Set(s.Config.EnvValues.AccrualCfg.AccrualAddr)
	}
	err := run(s)
	if err != nil {
		logger.Log.Error("Run server error", zap.Error(err))
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
	if s.Config.HostConfig.String() != "" {
		logger.Log.Info("Run Server on", zap.String("Server addr", s.Config.HostConfig.String()))
		return http.ListenAndServe(s.Config.HostConfig.String(), r)
	}
	if s.Config.EnvValues.ServerCfg.Addr != "" {
		logger.Log.Info("Run Server on", zap.String("Server addr", s.Config.EnvValues.ServerCfg.Addr))
		return http.ListenAndServe(s.Config.EnvValues.ServerCfg.Addr, r)
	}
	logger.Log.Info("Run server on", zap.String("Server addr", "localhost:8080"))
	return http.ListenAndServe(":8080", r)
}

func initDB(DBAddr string) *pgxpool.Pool {
	pool, err := pgxpool.New(context.Background(), DBAddr)
	if err != nil {
		logger.Log.Error("Error wile init db driver: " + err.Error())
		panic(err)
	}
	return pool

}
