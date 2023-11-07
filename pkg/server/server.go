package server

import "net/http"

type Server struct {
	storage string
	config  string
}

func (s *Server) RegisterHandler(res http.ResponseWriter, req *http.Request) {

}

func (s *Server) LoginHandler(res http.ResponseWriter, req *http.Request) {

}

func (s *Server) UploadOrderHandler(res http.ResponseWriter, req *http.Request) {

}

func (s *Server) UnloadHandler(res http.ResponseWriter, req *http.Request) {

}

func (s *Server) GetBalanceHandler(res http.ResponseWriter, req *http.Request) {

}

func (s *Server) WriteOffBonusHandler(res http.ResponseWriter, req *http.Request) {

}

func (s *Server) WriteOffBalanceHistoryHandler(res http.ResponseWriter, req *http.Request) {

}
