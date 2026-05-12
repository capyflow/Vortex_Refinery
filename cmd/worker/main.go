package main

import (
	"context"
	"encoding/base64"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"Vortex_Refinery/config"
	"Vortex_Refinery/internal/bus"
	plugin_pkg "Vortex_Refinery/internal/plugin"
)

type Worker struct {
	cfg        *config.Config
	pluginHost *plugin_pkg.PluginHost
	loader     *plugin_pkg.Loader
	eventBus   *bus.TaskBus
	workerID   string
}

func NewWorker(cfg *config.Config) (*Worker, error) {
	workerID := "worker-" + uuid.New().String()[:8]

	// Build verifier from config
	var verifier *plugin_pkg.Verifier
	if len(cfg.Worker.Plugin.TrustedSigners) > 0 {
		var keys []plugin_pkg.TrustedKey
		for _, s := range cfg.Worker.Plugin.TrustedSigners {
			pubKey, err := base64.StdEncoding.DecodeString(s.PublicKey)
			if err != nil {
				log.Printf("invalid public key for signer %s: %v", s.Name, err)
				continue
			}
			keys = append(keys, plugin_pkg.TrustedKey{
				Name:      s.Name,
				Algorithm: s.Algorithm,
				PublicKey: pubKey,
			})
		}
		verifier = plugin_pkg.NewVerifier(keys, cfg.Worker.Plugin.RequireSignature)
	}

	// Create plugin loader
	host := &plugin_pkg.PluginHost{
		WorkerID: workerID,
		Logger:   &SimpleLogger{workerID: workerID},
	}
	loader := plugin_pkg.NewLoader(cfg.Worker.Plugin.Dir, host, verifier)

	return &Worker{
		cfg:        cfg,
		pluginHost: host,
		loader:     loader,
		workerID:   workerID,
	}, nil
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

func (w *Worker) Start(ctx context.Context) error {
	log.Printf("Worker %s starting...", w.workerID)

	// Initialize task bus
	eventBus, err := bus.NewTaskBus(
		w.cfg.Redis.Addr,
		w.cfg.Redis.Password,
		w.cfg.Redis.DB,
		w.cfg.Redis.TaskStreamKey,
		"vortex-workers",
	)
	if err != nil {
		return err
	}
	w.eventBus = eventBus

	// Discover and load plugins
	if err := w.loader.DiscoverAndLoad(ctx); err != nil {
		log.Printf("plugin discovery failed: %v", err)
	}
	log.Printf("loaded plugins: %v", w.loader.List())

	// Start file watcher for hot-reload
	if err := w.loader.Watch(ctx); err != nil {
		log.Printf("failed to start plugin watcher: %v", err)
	}

	// TODO: Register to master
	// TODO: Start task polling loop
	// TODO: Start heartbeat

	log.Printf("Worker %s started successfully", w.workerID)
	return nil
}

func main() {
	configPath := flag.String("config", "config/worker.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	w, err := NewWorker(cfg)
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Printf("shutdown signal received, stopping worker...")
		cancel()
	}()

	if err := w.Start(ctx); err != nil {
		log.Fatalf("worker failed: %v", err)
	}

	// Keep running until context cancelled
	<-ctx.Done()
	log.Printf("Worker %s stopped", w.workerID)
}
