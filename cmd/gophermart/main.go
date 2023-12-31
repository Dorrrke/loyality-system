package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/Dorrrke/loyality-system.git/internal/logger"
	"github.com/Dorrrke/loyality-system.git/pkg/server"
	"github.com/Dorrrke/loyality-system.git/pkg/storage"
	"github.com/caarlos0/env/v6"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func main() {
	log.Println("Init logger")
	if err := logger.Initialize(zap.InfoLevel.String()); err != nil {
		log.Println("init logger error" + err.Error())
		log.Println("Panic logger")
		os.Exit(1)
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
	} else {
		logger.Log.Error("env db err", zap.Error(dbDsnErr))
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
		s.Config.AccrualConfig.Set(s.Config.EnvValues.AccrualCfg.AccrualAddr)
	}
	if s.Config.EnvValues.DataBaseDsn.DBDSN == "" && DBaddr == "" {
		log.Println("Error init db")
		log.Println("DB env str" + s.Config.EnvValues.DataBaseDsn.DBDSN)
		log.Println("DB flag str" + DBaddr)
		os.Exit(1)
	}
	if err := s.CreateTable(); err != nil {
		logger.Log.Error("Error create tables", zap.Error(err))
	}
	// TODO: Решить проблему с миграцией; Вылетает ошибка no scheme
	// logger.Log.Info("DB migration")
	// if s.Config.EnvValues.DataBaseDsn.DBDSN != "" {
	// 	if err := s.MigrateDB(s.Config.EnvValues.DataBaseDsn.DBDSN); err != nil {
	// 		logger.Log.Error("Migration failed", zap.Error(err))
	// 	}
	// } else {
	// 	if err := s.MigrateDB(DBaddr); err != nil {
	// 		logger.Log.Error("Migration failed", zap.Error(err))
	// 	}
	// }
	s.New()
	err := run(s)
	if err != nil {
		logger.Log.Error("Run server error", zap.Error(err))
		log.Println("Panic run")
		os.Exit(1)
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
	logger.Log.Info("Run server params:",
		zap.String("flag -a:", s.Config.HostConfig.String()),
		zap.String("RUN_ADDRES ENV:", s.Config.EnvValues.ServerCfg.Addr),
		zap.String("ACCRUAL_SYSTEM_ADDRESS env:", s.Config.EnvValues.AccrualCfg.AccrualAddr))
	if s.Config.EnvValues.ServerCfg.Addr != "" {
		logger.Log.Info("Run Server on", zap.String("Server addr from env", s.Config.EnvValues.ServerCfg.Addr))
		return http.ListenAndServe(s.Config.EnvValues.ServerCfg.Addr, r)
	}
	if s.Config.HostConfig.Host != "" {
		logger.Log.Info("Run Server on", zap.String("Server addr from flag", s.Config.HostConfig.String()))
		return http.ListenAndServe(s.Config.HostConfig.String(), r)
	}
	logger.Log.Info("Run server on", zap.String("Server addr default", "localhost:8080"))
	return http.ListenAndServe(":8080", r)
}

func initDB(DBAddr string) *pgxpool.Pool {
	pool, err := pgxpool.New(context.Background(), DBAddr)
	if err != nil {
		logger.Log.Error("Error wile init db driver: " + err.Error())
		log.Println("Panic db")
		os.Exit(1)
	}
	return pool

}
