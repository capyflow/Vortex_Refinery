package worker

import (
	"context"
	"encoding/json"
	"log"

	"Vortex_Refinery/config"
	"Vortex_Refinery/internal/plugin"
	plugin_pkg "Vortex_Refinery/internal/plugin"
	"Vortex_Refinery/pkg/types"
)

// Executor executes tasks by calling plugins
type Executor struct {
	cfg        *config.Config
	pluginHost *plugin_pkg.PluginHost
	plugins    map[string]plugin_pkg.Plugin
	loader     *plugin_pkg.Loader
	validator  *plugin_pkg.Validator
}

// NewExecutor creates a new task executor
func NewExecutor(cfg *config.Config, pluginHost *plugin_pkg.PluginHost, loader *plugin_pkg.Loader) *Executor {
	return &Executor{
		cfg:        cfg,
		pluginHost: pluginHost,
		plugins:    make(map[string]plugin_pkg.Plugin),
		loader:     loader,
		validator:  plugin_pkg.NewValidator(),
	}
}

// RegisterPlugin registers a plugin with the executor
func (e *Executor) RegisterPlugin(p plugin_pkg.Plugin) error {
	name := p.Name()
	e.plugins[name] = p
	log.Printf("[%s] Plugin registered: %s", e.pluginHost.WorkerID, name)
	return nil
}

// Execute executes a task
func (e *Executor) Execute(ctx context.Context, task *types.Task) (*ExecutionResult, error) {
	pluginName := task.Plugin

	plugin, exists := e.plugins[pluginName]
	if !exists {
		return &ExecutionResult{
			TaskID:  task.TaskID,
			Success: false,
			Error:   "plugin not found: " + pluginName,
		}, nil
	}

	// Validate config against schema before execution
	if meta, ok := e.loader.GetMetadata(pluginName); ok {
		configBytes, err := json.Marshal(task.Config)
		if err != nil {
			return &ExecutionResult{
				TaskID:  task.TaskID,
				Success: false,
				Error:   "failed to serialize config: " + err.Error(),
			}, nil
		}
		if err := e.validator.Validate(meta, configBytes); err != nil {
			log.Printf("[%s] plugin %s config validation failed: %v", e.pluginHost.WorkerID, pluginName, err)
			return &ExecutionResult{
				TaskID:  task.TaskID,
				Success: false,
				Error:   "config validation failed: " + err.Error(),
			}, nil
		}
	}

	// Serialize context as input
	inputBytes, err := json.Marshal(task.Context)
	if err != nil {
		return &ExecutionResult{
			TaskID:  task.TaskID,
			Success: false,
			Error:   "failed to serialize input: " + err.Error(),
		}, nil
	}

	// Serialize config
	configBytes, err := json.Marshal(task.Config)
	if err != nil {
		return &ExecutionResult{
			TaskID:  task.TaskID,
			Success: false,
			Error:   "failed to serialize config: " + err.Error(),
		}, nil
	}

	// Execute plugin
	output, err := plugin.Execute(ctx, inputBytes, configBytes)
	if err != nil {
		return &ExecutionResult{
			TaskID:  task.TaskID,
			Success: false,
			Error:   "plugin execution failed: " + err.Error(),
		}, nil
	}

	// Parse output
	var outputMap map[string]string
	if err := json.Unmarshal(output, &outputMap); err != nil {
		// Try to use raw output as a single value
		outputMap = map[string]string{
			"result": string(output),
		}
	}

	return &ExecutionResult{
		TaskID:  task.TaskID,
		Success: true,
		Output:  outputMap,
	}, nil
}

// GetPlugin returns a plugin by name
func (e *Executor) GetPlugin(name string) (plugin_pkg.Plugin, bool) {
	p, ok := e.plugins[name]
	return p, ok
}

// ListPlugins lists all registered plugins
func (e *Executor) ListPlugins() []string {
	names := make([]string, 0, len(e.plugins))
	for name := range e.plugins {
		names = append(names, name)
	}
	return names
}
