package main

import (
	"net/http"

	"github.com/Dorrrke/loyality-system.git/pkg/server"
	"github.com/go-chi/chi/v5"
)

func main() {

	var s server.Server
	err := run(s)
	if err != nil {
		panic(err)
	}
}

func run(s server.Server) error {

	r := chi.NewRouter()

	r.Route("/api/users", func(r chi.Router) {
		r.Post("/register", s.RegisterHandler)
		r.Post("/login", s.LoginHandler)
		r.Post("/orders", s.UploadOrderHandler)
		r.Get("/orders", s.UnloadHandler)
		r.Route("/balance", func(r chi.Router) {
			r.Get("/", s.GetBalanceHandler)
			r.Post("/withdraw", s.WriteOffBonusHandler)
		})
		r.Get("/withdrawals", s.WriteOffBalanceHistoryHandler)
	})

	return http.ListenAndServe(":8080", r)
}
