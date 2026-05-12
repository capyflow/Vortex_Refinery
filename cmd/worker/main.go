package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"Vortex_Refinery/config"
	"Vortex_Refinery/internal/bus"
	plugin_pkg "Vortex_Refinery/internal/plugin"
)

type Worker struct {
	cfg       *config.Config
	pluginHost *plugin_pkg.PluginHost
	plugins   map[string]plugin_pkg.Plugin
	eventBus  *bus.TaskBus
}

func NewWorker(cfg *config.Config) *Worker {
	workerID := "worker-" + uuid.New().String()[:8]

	return &Worker{
		cfg: cfg,
		pluginHost: &plugin_pkg.PluginHost{
			WorkerID: workerID,
			Logger:   &SimpleLogger{workerID: workerID},
		},
		plugins: make(map[string]plugin_pkg.Plugin),
	}
}

type SimpleLogger struct {
	workerID string
}

func (l *SimpleLogger) Info(msg string, args ...interface{}) {
	log.Printf("[%s] INFO: %s", l.workerID, msg)
}

func (l *SimpleLogger) Error(msg string, args ...interface{}) {
	log.Printf("[%s] ERROR: %s", l.workerID, msg)
}

func (l *SimpleLogger) Debug(msg string, args ...interface{}) {
	log.Printf("[%s] DEBUG: %s", l.workerID, msg)
}

func (w *Worker) Start() error {
	log.Printf("Worker %s starting...", w.pluginHost.WorkerID)

	// Initialize task bus
	eventBus, err := bus.NewTaskBus(
		w.cfg.Redis.Addr,
		w.cfg.Redis.Password,
		w.cfg.Redis.DB,
		w.cfg.Redis.TaskStreamKey,
		"vortex:workers",
	)
	if err != nil {
		return err
	}
	w.eventBus = eventBus

	// TODO: Discover and load plugins
	// TODO: Register plugins to master
	// TODO: Start task polling loop
	// TODO: Start heartbeat

	log.Printf("Worker %s started successfully", w.pluginHost.WorkerID)
	return nil
}

func main() {
	configPath := flag.String("config", "config/worker.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	w := NewWorker(cfg)
	if err := w.Start(); err != nil {
		log.Fatalf("worker failed: %v", err)
	}
}
