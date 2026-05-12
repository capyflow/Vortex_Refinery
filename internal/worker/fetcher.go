package worker

import (
	"context"
	"log"
	"time"

	"Vortex_Refinery/config"
	"Vortex_Refinery/internal/bus"
)

// Fetcher fetches tasks from Redis
type Fetcher struct {
	cfg       *config.Config
	taskBus   *bus.TaskBus
	workerID  string
	executor  *Executor
	stopCh    chan struct{}
}

// NewFetcher creates a new task fetcher
func NewFetcher(
	cfg *config.Config,
	taskBus *bus.TaskBus,
	workerID string,
	executor *Executor,
) *Fetcher {
	return &Fetcher{
		cfg:      cfg,
		taskBus:  taskBus,
		workerID: workerID,
		executor: executor,
		stopCh:   make(chan struct{}),
	}
}

// Start starts the task fetching loop
func (f *Fetcher) Start(ctx context.Context) {
	log.Printf("[%s] Task fetcher starting", f.workerID)

	ticker := time.NewTicker(f.cfg.Master.TaskPullInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("[%s] Task fetcher stopping (context cancelled)", f.workerID)
			return
		case <-f.stopCh:
			log.Printf("[%s] Task fetcher stopping", f.workerID)
			return
		case <-ticker.C:
			if err := f.fetchTasks(ctx); err != nil {
				log.Printf("[%s] Error fetching tasks: %v", f.workerID, err)
			}
		}
	}
}

// Stop stops the task fetcher
func (f *Fetcher) Stop() {
	close(f.stopCh)
}

// fetchTasks fetches pending tasks from Redis
func (f *Fetcher) fetchTasks(ctx context.Context) error {
	tasks, err := f.taskBus.PullTasks(ctx, f.workerID, 10)
	if err != nil {
		return err
	}

	for _, task := range tasks {
		log.Printf("[%s] Received task %s for node %s", f.workerID, task.TaskID, task.NodeID)

		// Execute the task
		result, err := f.executor.Execute(ctx, task)
		if err != nil {
			log.Printf("[%s] Task %s execution failed: %v", f.workerID, task.TaskID, err)
			continue
		}

		// Report result back to master
		if err := f.reportResult(ctx, result); err != nil {
			log.Printf("[%s] Failed to report task result: %v", f.workerID, err)
		}
	}

	return nil
}

// reportResult reports task result back to master
func (f *Fetcher) reportResult(ctx context.Context, result *ExecutionResult) error {
	// Report via HTTP to master
	// This would typically use the master's HTTP API
	return nil
}

// ExecutionResult holds the result of a task execution
type ExecutionResult struct {
	TaskID  string
	Success bool
	Output  map[string]string
	Error   string
}
