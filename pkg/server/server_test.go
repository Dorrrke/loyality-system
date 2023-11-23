package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Dorrrke/loyality-system.git/pkg/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-resty/resty/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
)

func TestRegisterHandler(t *testing.T) {

	var server Server
	conn, err := pgxpool.New(context.Background(), "postgres://postgres:6406655@localhost:5432/postgres")
	if err != nil {
		panic(err)
	}
	server.ConnStorage(&storage.DataBaseStorage{DB: conn})
	defer conn.Close()

	r := chi.NewRouter()

	r.Route("/api/user", func(r chi.Router) {
		r.Post("/register", server.RegisterHandler)
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
			name: "Test register hadler #1",
			want: want{
				code: http.StatusOK,
			},
			request: "/api/user/register",
			body: `{
				"login": "admintest1",
				"password": "adminPassTest1"
			}`,
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
			name: "Test register hadler #2",
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
	conn, err := pgxpool.New(context.Background(), "postgres://postgres:6406655@localhost:5432/postgres")
	if err != nil {
		panic(err)
	}
	server.ConnStorage(&storage.DataBaseStorage{DB: conn})
	defer conn.Close()

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
				"login": "admintest1",
				"password": "adminPassTest1"
			}`,
			method: http.MethodPost,
		},
		{
			name: "Test login hadler #2",
			want: want{
				code: http.StatusBadRequest,
			},
			request: "/api/user/login",
			body:    `"login": "login", "password": "password"`,
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
	conn, err := pgxpool.New(context.Background(), "postgres://postgres:6406655@localhost:5432/postgres")
	if err != nil {
		panic(err)
	}
	server.ConnStorage(&storage.DataBaseStorage{DB: conn})
	defer conn.Close()

	r := chi.NewRouter()

	r.Route("/api/user", func(r chi.Router) {
		r.Post("/orders", server.UploadOrderHandler)
	})
	srv := httptest.NewServer(r)

	type want struct {
		code int
	}

	tests := []struct {
		name    string
		body    string
		request string
		header  string
		method  string
		want    want
	}{
		{
			name: "Test Upload hadler #1",
			want: want{
				code: http.StatusAccepted,
			},
			header:  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MDA3NTU5OTcsIlVzZXJJRCI6IjE0In0.yc-YN7H9bfBUDTE0NoIq6oBrLTXW7veUJRWNurkLbNs",
			request: "/api/user/orders",
			body:    `3437326824`,
			method:  http.MethodPost,
		},
		{
			name: "Test Upload hadler #2",
			want: want{
				code: http.StatusUnprocessableEntity,
			},
			header:  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MDA3NTU5OTcsIlVzZXJJRCI6IjE0In0.yc-YN7H9bfBUDTE0NoIq6oBrLTXW7veUJRWNurkLbNs",
			request: "/api/user/orders",
			body:    `1234`,
			method:  http.MethodPost,
		},
		{
			name: "Test Upload hadler #3",
			want: want{
				code: http.StatusUnauthorized,
			},
			request: "/api/user/orders",
			body:    `3437326824`,
			method:  http.MethodPost,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := resty.New().R()
			req.Method = http.MethodPost
			req.URL = srv.URL + tt.request
			req.Header.Add("Authorization", tt.header)
			req.Body = tt.body
			resp, err := req.Send()
			assert.NoError(t, err, "error making HTTP request")
			assert.Equal(t, tt.want.code, resp.StatusCode())

		})
	}
}
