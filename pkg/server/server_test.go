package server

import (
	"context"
	"flag"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Dorrrke/loyality-system.git/pkg/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-resty/resty/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var db = flag.String("db", "", "DataBase url")
var accrualAddr = flag.String("r", "", "address and port accrual")

func TestRegisterHandler(t *testing.T) {

	var server Server
	server.Config.AccrualConfig.Set(*accrualAddr)
	conn, err := pgxpool.New(context.Background(), *db)
	if err != nil {
		panic(err)
	}
	server.ConnStorage(&storage.DataBaseStorage{DB: conn})
	defer conn.Close()

	if err := server.CreateTable(); err != nil {
		panic(err)
	}

	r := chi.NewRouter()

	r.Route("/api/user", func(r chi.Router) {
		r.Post("/register", server.RegisterHandler)
	})
	srv := httptest.NewServer(r)

	log.Println("Server addr ", srv.URL)

	if err := server.ClearTables(); err != nil {
		log.Println(err.Error())
	}

	type want struct {
		code int
	}

	tests := []struct {
		name    string
		body    string
		request string
		method  string
		want    want
	}{
		{
			name: "Test register hadler #1",
			want: want{
				code: http.StatusOK,
			},
			request: "/api/user/register",
			body: `{
				"login": "admin",
				"password": "adminPass"
			}`,
			method: http.MethodPost,
		},
		{
			name: "Test register hadler #1.1",
			want: want{
				code: http.StatusOK,
			},
			request: "/api/user/register",
			body: `{
				"login": "22admin",
				"password": "23adminPass" }`,
			method: http.MethodPost,
		},
		{
			name: "Test register hadler #2",
			want: want{
				code: http.StatusBadRequest,
			},
			request: "/api/user/register",
			body:    `"login": "login", "password": "password"`,
			method:  http.MethodPost,
		},
		{
			name: "Test register hadler #3",
			want: want{
				code: http.StatusConflict,
			},
			request: "/api/user/register",
			body: `{
				"login": "admin",
				"password": "adminPass"
			}`,
			method: http.MethodPost,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := resty.New().R()
			req.Method = http.MethodPost
			req.URL = srv.URL + tt.request
			req.Body = tt.body
			resp, err := req.Send()
			assert.NoError(t, err, "error making HTTP request")
			assert.Equal(t, tt.want.code, resp.StatusCode())

		})
	}
}

func TestLoginHandler(t *testing.T) {

	var server Server
	server.Config.AccrualConfig.Set(*accrualAddr)
	conn, err := pgxpool.New(context.Background(), *db)
	if err != nil {
		panic(err)
	}
	server.ConnStorage(&storage.DataBaseStorage{DB: conn})
	defer conn.Close()

	if err := server.CreateTable(); err != nil {
		panic(err)
	}

	r := chi.NewRouter()

	r.Route("/api/user", func(r chi.Router) {
		r.Post("/login", server.LoginHandler)
	})
	srv := httptest.NewServer(r)

	type want struct {
		code int
	}

	tests := []struct {
		name    string
		body    string
		request string
		method  string
		want    want
	}{
		{
			name: "Test login hadler #1",
			want: want{
				code: http.StatusOK,
			},
			request: "/api/user/login",
			body: `{
				"login": "admin",
				"password": "adminPass"
			}`,
			method: http.MethodPost,
		},
		{
			name: "Test login hadler #2",
			want: want{
				code: http.StatusBadRequest,
			},
			request: "/api/user/login",
			body:    `"login": "login"`,
			method:  http.MethodPost,
		},
		{
			name: "Test login hadler #3",
			want: want{
				code: http.StatusUnauthorized,
			},
			request: "/api/user/login",
			body: `{
				"login": "admin",
				"password": "Pass"
			}`,
			method: http.MethodPost,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := resty.New().R()
			req.Method = http.MethodPost
			req.URL = srv.URL + tt.request
			req.Body = tt.body
			resp, err := req.Send()
			assert.NoError(t, err, "error making HTTP request")
			assert.Equal(t, tt.want.code, resp.StatusCode())

		})
	}
}

func TestUploadOrderHandler(t *testing.T) {

	var server Server
	server.Config.AccrualConfig.Set(*accrualAddr)
	conn, err := pgxpool.New(context.Background(), *db)
	if err != nil {
		panic(err)
	}
	server.ConnStorage(&storage.DataBaseStorage{DB: conn})
	defer conn.Close()

	err = server.CreateTable()
	require.NoError(t, err)

	r := chi.NewRouter()

	r.Route("/api/user", func(r chi.Router) {
		r.Post("/orders", server.UploadOrderHandler)
		r.Post("/login", server.LoginHandler)
	})
	srv := httptest.NewServer(r)

	type want struct {
		code int
	}

	tests := []struct {
		name      string
		body      string
		loginBody string
		request   string
		header    bool
		method    string
		want      want
	}{
		{
			name: "Test Upload hadler #1",
			want: want{
				code: http.StatusAccepted,
			},
			header: true,
			loginBody: `{
				"login": "admin",
				"password": "adminPass" }`,
			request: "/api/user/orders",
			body:    `12345678903`,
			method:  http.MethodPost,
		},
		{
			name: "Test Upload hadler #2",
			want: want{
				code: http.StatusUnprocessableEntity,
			},
			header: true,
			loginBody: `{
				"login": "admin",
				"password": "adminPass" }`,
			request: "/api/user/orders",
			body:    `1234`,
			method:  http.MethodPost,
		},
		{
			name: "Test Upload hadler #3",
			want: want{
				code: http.StatusUnauthorized,
			},
			header: false,
			loginBody: `{
				"login": "admin",
				"password": "adminPass" }`,
			request: "/api/user/orders",
			body:    `3437326824`,
			method:  http.MethodPost,
		},
		{
			name: "Test Upload hadler #4",
			want: want{
				code: http.StatusConflict,
			},
			header: true,
			loginBody: `{
				"login": "22admin",
				"password": "23adminPass" }`,
			request: "/api/user/orders",
			body:    `12345678903`,
			method:  http.MethodPost,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqLogin := resty.New().R()
			reqLogin.Method = http.MethodPost
			reqLogin.URL = srv.URL + `/api/user/login`
			reqLogin.Body = tt.loginBody
			respLogin, err := reqLogin.Send()
			if err != nil {
				panic(err)
			}

			auth := respLogin.Header().Get("Authorization")

			req := resty.New().R()
			req.Method = tt.method
			req.URL = srv.URL + tt.request
			if tt.header {
				req.Header.Add("Authorization", auth)
			}
			req.Body = tt.body
			resp, err := req.Send()
			assert.NoError(t, err, "error making HTTP request")
			assert.Equal(t, tt.want.code, resp.StatusCode())

		})
	}
}

func TestGetBalanceHandler(t *testing.T) {

	var server Server
	server.Config.AccrualConfig.Set(*accrualAddr)
	conn, err := pgxpool.New(context.Background(), *db)
	if err != nil {
		panic(err)
	}

	server.ConnStorage(&storage.DataBaseStorage{DB: conn})
	defer conn.Close()

	if err := server.CreateTable(); err != nil {
		panic(err)
	}

	r := chi.NewRouter()

	r.Route("/api/user", func(r chi.Router) {
		r.Get("/balance", server.GetBalanceHandler)
		r.Post("/login", server.LoginHandler)
	})
	srv := httptest.NewServer(r)

	type want struct {
		code        int
		contentType string
	}

	tests := []struct {
		name      string
		loginBody string
		request   string
		header    bool
		method    string
		want      want
	}{
		{
			name: "Test GetBalanceHandler #1",
			want: want{
				code:        http.StatusOK,
				contentType: "application/json",
			},
			header: true,
			loginBody: `{
				"login": "admin",
				"password": "adminPass" }`,
			request: "/api/user/balance",
			method:  http.MethodGet,
		},
		{
			name: "Test GetBalanceHandler #3",
			want: want{
				code:        http.StatusUnauthorized,
				contentType: "text/plain; charset=utf-8",
			},
			header: false,
			loginBody: `{
				"login": "admin",
				"password": "adminPass" }`,
			request: "/api/user/balance",
			method:  http.MethodGet,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqLogin := resty.New().R()
			reqLogin.Method = http.MethodPost
			reqLogin.URL = srv.URL + `/api/user/login`
			reqLogin.Body = tt.loginBody
			respLogin, err := reqLogin.Send()
			if err != nil {
				panic(err)
			}

			auth := respLogin.Header().Get("Authorization")

			req := resty.New().R()
			req.Method = tt.method
			req.URL = srv.URL + tt.request
			if tt.header {
				req.Header.Add("Authorization", auth)
			}
			resp, err := req.Send()
			assert.NoError(t, err, "error making HTTP request")
			assert.Equal(t, tt.want.code, resp.StatusCode())
			assert.Equal(t, tt.want.contentType, resp.Header().Get("Content-Type"))

		})
	}
}

func TestWriteOffBonusHandler(t *testing.T) {

	var server Server
	server.Config.AccrualConfig.Set(*accrualAddr)
	conn, err := pgxpool.New(context.Background(), *db)
	if err != nil {
		panic(err)
	}

	server.ConnStorage(&storage.DataBaseStorage{DB: conn})
	defer conn.Close()

	if err := server.CreateTable(); err != nil {
		panic(err)
	}

	r := chi.NewRouter()

	r.Route("/api/user", func(r chi.Router) {
		r.Post("/login", server.LoginHandler)
		r.Route("/balance", func(r chi.Router) {
			r.Get("/", server.GetBalanceHandler)
			r.Post("/withdraw", server.WriteOffBonusHandler)
		})
	})
	srv := httptest.NewServer(r)

	type want struct {
		code int
	}

	tests := []struct {
		name      string
		loginBody string
		body      string
		request   string
		header    bool
		method    string
		want      want
	}{
		{
			name: "Test WriteOff Bonus Handler #1",
			want: want{
				code: http.StatusOK,
			},
			header: true,
			loginBody: `{
				"login": "admin",
				"password": "adminPass" }`,
			body: `{
				"order": "2377225624",
				"sum": 100
			} `,
			request: "/api/user/balance/withdraw",
			method:  http.MethodPost,
		},
		{
			name: "Test Test WriteOff Bonus Handler #2",
			want: want{
				code: http.StatusPaymentRequired,
			},
			header: true,
			loginBody: `{
				"login": "admin",
				"password": "adminPass" }`,
			body: `{
				"order": "2377225616",
				"sum": 999
			} `,
			request: "/api/user/balance/withdraw",
			method:  http.MethodPost,
		},
		{
			name: "Test Test WriteOff Bonus Handler #3",
			want: want{
				code: http.StatusUnauthorized,
			},
			header: false,
			loginBody: `{
				"login": "admin",
				"password": "adminPass" }`,
			request: "/api/user/balance/withdraw",
			method:  http.MethodPost,
		},
		{
			name: "Test Test WriteOff Bonus Handler #4",
			want: want{
				code: http.StatusUnprocessableEntity,
			},
			header: true,
			loginBody: `{
				"login": "admin",
				"password": "adminPass" }`,
			body: `{
				"order": "32521541546",
				"sum": 10
			} `,
			request: "/api/user/balance/withdraw",
			method:  http.MethodPost,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqLogin := resty.New().R()
			reqLogin.Method = http.MethodPost
			reqLogin.URL = srv.URL + `/api/user/login`
			reqLogin.Body = tt.loginBody
			respLogin, err := reqLogin.Send()
			if err != nil {
				panic(err)
			}

			auth := respLogin.Header().Get("Authorization")

			req := resty.New().R()
			req.Method = tt.method
			req.Body = tt.body
			req.URL = srv.URL + tt.request
			if tt.header {
				req.Header.Add("Authorization", auth)
			}
			resp, err := req.Send()
			assert.NoError(t, err, "error making HTTP request")
			assert.Equal(t, tt.want.code, resp.StatusCode())

		})
	}
}

func TestWriteOffBalanceHistoryHandler(t *testing.T) {

	var server Server
	server.Config.AccrualConfig.Set(*accrualAddr)
	conn, err := pgxpool.New(context.Background(), *db)
	if err != nil {
		panic(err)
	}

	server.ConnStorage(&storage.DataBaseStorage{DB: conn})
	defer conn.Close()

	if err := server.CreateTable(); err != nil {
		panic(err)
	}

	r := chi.NewRouter()

	r.Route("/api/user", func(r chi.Router) {
		r.Post("/login", server.LoginHandler)
		r.Route("/balance", func(r chi.Router) {
			r.Get("/", server.GetBalanceHandler)
			r.Post("/withdraw", server.WriteOffBonusHandler)
		})
		r.Get("/withdrawals", server.WriteOffBalanceHistoryHandler)
	})
	srv := httptest.NewServer(r)

	type want struct {
		code        int
		contentType string
	}

	tests := []struct {
		name      string
		loginBody string
		request   string
		header    bool
		method    string
		want      want
	}{
		{
			name: "Test GetHistoryHandler #1",
			want: want{
				code:        http.StatusOK,
				contentType: "application/json",
			},
			header: true,
			loginBody: `{
				"login": "admin",
				"password": "adminPass" }`,
			request: "/api/user/withdrawals",
			method:  http.MethodGet,
		},
		{
			name: "Test GetHistoryHandler #2",
			want: want{
				code:        http.StatusNoContent,
				contentType: "text/plain; charset=utf-8",
			},
			header: true,
			loginBody: `{
				"login": "22admin",
				"password": "23adminPass" }`,
			request: "/api/user/withdrawals",
			method:  http.MethodGet,
		},
		{
			name: "Test GetHistoryHandler #3",
			want: want{
				code:        http.StatusUnauthorized,
				contentType: "text/plain; charset=utf-8",
			},
			header: false,
			loginBody: `{
				"login": "admin",
				"password": "adminPass" }`,
			request: "/api/user/withdrawals",
			method:  http.MethodGet,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqLogin := resty.New().R()
			reqLogin.Method = http.MethodPost
			reqLogin.URL = srv.URL + `/api/user/login`
			reqLogin.Body = tt.loginBody
			respLogin, err := reqLogin.Send()
			if err != nil {
				panic(err)
			}

			auth := respLogin.Header().Get("Authorization")

			req := resty.New().R()
			req.Method = tt.method
			req.URL = srv.URL + tt.request
			if tt.header {
				req.Header.Add("Authorization", auth)
			}
			resp, err := req.Send()
			assert.NoError(t, err, "error making HTTP request")
			assert.Equal(t, tt.want.code, resp.StatusCode())
			assert.Equal(t, tt.want.contentType, resp.Header().Get("Content-Type"))

		})
	}
}

func TestUnloadHandler(t *testing.T) {

	var server Server
	server.Config.AccrualConfig.Set(*accrualAddr)
	conn, err := pgxpool.New(context.Background(), *db)
	if err != nil {
		panic(err)
	}

	server.ConnStorage(&storage.DataBaseStorage{DB: conn})
	defer conn.Close()

	if err := server.CreateTable(); err != nil {
		panic(err)
	}

	r := chi.NewRouter()

	r.Route("/api/user", func(r chi.Router) {
		r.Post("/login", server.LoginHandler)
		r.Get("/orders", server.UnloadHandler)
	})
	srv := httptest.NewServer(r)

	type want struct {
		code        int
		contentType string
	}

	tests := []struct {
		name      string
		loginBody string
		request   string
		header    bool
		method    string
		want      want
	}{
		{
			name: "Test Unload Handler #1",
			want: want{
				code:        http.StatusOK,
				contentType: "application/json",
			},
			header: true,
			loginBody: `{
				"login": "admin",
				"password": "adminPass" }`,
			request: "/api/user/orders",
			method:  http.MethodGet,
		},
		{
			name: "Test Unload Handler #2",
			want: want{
				code:        http.StatusNoContent,
				contentType: "text/plain; charset=utf-8",
			},
			header: true,
			loginBody: `{
				"login": "22admin",
				"password": "23adminPass" }`,
			request: "/api/user/orders",
			method:  http.MethodGet,
		},
		{
			name: "Test Unload Handler #3",
			want: want{
				code:        http.StatusUnauthorized,
				contentType: "text/plain; charset=utf-8",
			},
			header: false,
			loginBody: `{
				"login": "admin",
				"password": "adminPass" }`,
			request: "/api/user/orders",
			method:  http.MethodGet,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqLogin := resty.New().R()
			reqLogin.Method = http.MethodPost
			reqLogin.URL = srv.URL + `/api/user/login`
			reqLogin.Body = tt.loginBody
			respLogin, err := reqLogin.Send()
			if err != nil {
				panic(err)
			}

			auth := respLogin.Header().Get("Authorization")

			req := resty.New().R()
			req.Method = tt.method
			req.URL = srv.URL + tt.request
			if tt.header {
				req.Header.Add("Authorization", auth)
			}
			resp, err := req.Send()
			assert.NoError(t, err, "error making HTTP request")
			assert.Equal(t, tt.want.code, resp.StatusCode())
			assert.Equal(t, tt.want.contentType, resp.Header().Get("Content-Type"))

		})
	}
}

// var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// func randSeq(n int) string {
// 	random := rand.New(rand.NewSource(time.Now().UnixNano()))
// 	b := make([]rune, n)
// 	for i := range b {
// 		b[i] = letters[random.Intn(len(letters))]
// 	}
// 	return string(b)
// }
