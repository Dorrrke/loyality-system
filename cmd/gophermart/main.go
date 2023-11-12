package main

import (
	"net/http"

	"github.com/Dorrrke/loyality-system.git/internal/logger"
	"github.com/Dorrrke/loyality-system.git/pkg/server"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func main() {

	if err := logger.Initialize(zap.InfoLevel.String()); err != nil {
		panic(err)
	}

	var s server.Server
	err := run(s)
	if err != nil {
		panic(err)
	}
}

func run(s server.Server) error {

	r := chi.NewRouter()

	r.Route("/api/users", func(r chi.Router) {
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

	return http.ListenAndServe(":8080", r)
}
