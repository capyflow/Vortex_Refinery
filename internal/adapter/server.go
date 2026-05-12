package adapter

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"Vortex_Refinery/internal/bus"
	"Vortex_Refinery/pkg/types"
)

// Server is the webhook adapter HTTP server
type Server struct {
	cfg      *Config
	eventBus *bus.EventBus
	router   *mux.Router
	server   *http.Server
}

// Config holds server configuration
type Config struct {
	Addr      string
	EventBus  *bus.EventBus
}

// NewServer creates a new webhook adapter server
func NewServer(cfg *Config) *Server {
	router := mux.NewRouter()
	s := &Server{
		cfg:      cfg,
		eventBus: cfg.EventBus,
		router:   router,
	}

	s.setupRoutes()
	return s
}

// Config holds adapter server configuration
type ServerConfig struct {
	Addr string
}

// setupRoutes configures HTTP routes
func (s *Server) setupRoutes() {
	s.router.HandleFunc("/webhook", s.handleWebhook).Methods("POST")
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")
}

// handleWebhook handles incoming webhook requests
func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	var event types.Event

	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		// Try to parse as generic payload
		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		event.Payload = payload
	}

	// Generate event ID and timestamp if not provided
	if event.EventID == "" {
		event.EventID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Push to Redis Stream
	ctx := context.Background()
	if err := s.eventBus.PushEvent(ctx, &event); err != nil {
		log.Printf("Failed to push event: %v", err)
		http.Error(w, "Failed to process event", http.StatusInternalServerError)
		return
	}

	// Return 202 Accepted
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"event_id": event.EventID,
		"status":   "accepted",
	})
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
	})
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.server = &http.Server{
		Addr:    s.cfg.Addr,
		Handler: s.router,
	}

	log.Printf("Webhook adapter starting on %s", s.cfg.Addr)
	return s.server.ListenAndServe()
}

// Stop stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
