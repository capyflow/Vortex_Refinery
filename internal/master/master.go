package master

import (
	"context"
	"time"

	"github.com/capyflow/allspark-go/logx"

	"Vortex_Refinery/conf"
	"Vortex_Refinery/internal/bus"
	"Vortex_Refinery/internal/store"
	"Vortex_Refinery/server"
)

type Master struct {
	cfg       *conf.Config
	store     *store.MongoStore
	eventBus  *bus.EventBus
	taskBus   *bus.TaskBus
	refinery  *server.RefineryServer
}

func New(s *store.MongoStore, eb *bus.EventBus, tb *bus.TaskBus, cfg *conf.Config) *Master {
	return &Master{
		cfg:      cfg,
		store:    s,
		eventBus: eb,
		taskBus:  tb,
	}
}

func (m *Master) Start() error {
	logx.Info("Master|Start|Info|starting master")

	// Start Refinery HTTP API Server
	refinery := server.NewRefineryServer(context.Background(), m.cfg, m.store)
	m.refinery = refinery
	go refinery.Start()

	// TODO: Initialize gRPC server
	// TODO: Start event consumer loop
	// TODO: Start task result receiver
	// TODO: Start heartbeat checker

	logx.Info("Master|Start|Info|master started successfully")
	return nil
}

func (m *Master) Shutdown(ctx context.Context) error {
	logx.Infof("Master|Shutdown|Info|shutting down")
	if m.refinery != nil {
		return m.refinery.Shutdown(ctx)
	}
	return nil
}

func (m *Master) Health() map[string]interface{} {
	return map[string]interface{}{
		"status": "ok",
		"uptime": time.Now().Unix(),
	}
}
