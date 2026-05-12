package api

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

// Server is the REST API server
type Server struct {
	addr    string
	router  *mux.Router
	handler *Handler
}

// NewServer creates a new API server
func NewServer(addr string, h *Handler) *Server {
	r := mux.NewRouter()
	h.RegisterRoutes(r)
	return &Server{
		addr:    addr,
		router:  r,
		handler: h,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	log.Printf("[API] REST server listening on %s", s.addr)
	return http.ListenAndServe(s.addr, s.router)
}
