package worker

import (
	"context"
	"log"
	"time"

	"Vortex_Refinery/config"
	plugin_pkg "Vortex_Refinery/internal/plugin"
	store_pkg "Vortex_Refinery/internal/store"
	"Vortex_Refinery/pkg/types"
)

// Registrar handles plugin registration with master
type Registrar struct {
	cfg        *config.Config
	workerID   string
	pluginHost *plugin_pkg.PluginHost
	workerStore *store_pkg.WorkerStore
	executor   *Executor
	httpClient *HTTPClient
	stopCh     chan struct{}
}

// HTTPClient is a simple HTTP client interface
type HTTPClient interface {
	Post(url string, body interface{}) error
}

// NewRegistrar creates a new plugin registrar
func NewRegistrar(
	cfg *config.Config,
	workerID string,
	pluginHost *plugin_pkg.PluginHost,
	workerStore *store_pkg.WorkerStore,
	executor *Executor,
) *Registrar {
	return &Registrar{
		cfg:         cfg,
		workerID:    workerID,
		pluginHost:  pluginHost,
		workerStore: workerStore,
		executor:    executor,
		stopCh:      make(chan struct{}),
	}
}

// RegisterPlugins registers all plugins with master
func (r *Registrar) RegisterPlugins(ctx context.Context, plugins []string) error {
	worker := &types.WorkerRegistry{
		WorkerID: r.workerID,
		Plugins:  plugins,
		Status:   "online",
	}

	if err := r.workerStore.Register(ctx, worker); err != nil {
		return err
	}

	log.Printf("[%s] Registered %d plugins with master", r.workerID, len(plugins))
	return nil
}

// StartHeartbeat starts the heartbeat loop
func (r *Registrar) StartHeartbeat(ctx context.Context) {
	log.Printf("[%s] Heartbeat sender starting", r.workerID)

	ticker := time.NewTicker(r.cfg.Master.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("[%s] Heartbeat sender stopping (context cancelled)", r.workerID)
			return
		case <-r.stopCh:
			log.Printf("[%s] Heartbeat sender stopping", r.workerID)
			return
		case <-ticker.C:
			if err := r.sendHeartbeat(ctx); err != nil {
				log.Printf("[%s] Failed to send heartbeat: %v", r.workerID, err)
			}
		}
	}
}

// Stop stops the heartbeat sender
func (r *Registrar) Stop() {
	close(r.stopCh)
}

// sendHeartbeat sends a heartbeat to master
func (r *Registrar) sendHeartbeat(ctx context.Context) error {
	return r.workerStore.UpdateHeartbeat(ctx, r.workerID)
}

// ReportTaskResult reports a task result to master
func (r *Registrar) ReportTaskResult(ctx context.Context, result *ExecutionResult) error {
	// In a real implementation, this would call the master's HTTP API
	// For now, this is handled via the task bus acknowledgment
	log.Printf("[%s] Task result: %s success=%v", r.workerID, result.TaskID, result.Success)
	return nil
}
