package master

import (
	"context"
	"log"
	"net/http"
	"time"

	"Vortex_Refinery/config"
	"Vortex_Refinery/internal/api"
	"Vortex_Refinery/internal/bus"
	"Vortex_Refinery/internal/store"
)

type Master struct {
	cfg       *config.Config
	store     *store.MongoStore
	eventBus  *bus.EventBus
	taskBus   *bus.TaskBus
	apiServer *api.Server
}

func New(s *store.MongoStore, eb *bus.EventBus, tb *bus.TaskBus, cfg *config.Config) *Master {
	return &Master{
		cfg:      cfg,
		store:    s,
		eventBus: eb,
		taskBus:  tb,
	}
}

func (m *Master) Start() error {
	log.Println("Master starting...")

	// Start REST API server
	if m.cfg.Server.HTTPAddr != "" {
		wfDir := m.cfg.Workflows.Dir
		if wfDir == "" {
			wfDir = "./workflows"
		}
		h := api.NewHandler(m.store, wfDir)
		m.apiServer = api.NewServer(m.cfg.Server.HTTPAddr, h)

		go func() {
			log.Printf("REST API server listening on %s", m.cfg.Server.HTTPAddr)
			if err := m.apiServer.Start(); err != nil && err != http.ErrServerClosed {
				log.Printf("API server error: %v", err)
			}
		}()
	}

	// TODO: Initialize gRPC server
	// TODO: Start event consumer loop
	// TODO: Start task result receiver
	// TODO: Start heartbeat checker

	log.Println("Master started successfully")
	return nil
}

func (m *Master) Stop(ctx context.Context) error {
	if m.apiServer != nil {
		return nil // http.Server handles shutdown via ctx
	}
	return nil
}

func (m *Master) Health() map[string]interface{} {
	return map[string]interface{}{
		"status":     "ok",
		"uptime":     time.Now().Unix(),
	}
}
