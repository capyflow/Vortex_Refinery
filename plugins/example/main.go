package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

// Plugin implements the Vortex_Refinery plugin interface
type Plugin struct{}

// Name returns the plugin name
func (p *Plugin) Name() string {
	return "example-processor"
}

// Execute processes the input and returns the output
func (p *Plugin) Execute(ctx context.Context, input []byte, config json.RawMessage) ([]byte, error) {
	var cfg struct {
		Operation string `json:"operation"`
	}
	if err := json.Unmarshal(config, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	var inputData map[string]string
	if err := json.Unmarshal(input, &inputData); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	result := map[string]string{
		"status":    "completed",
		"operation": cfg.Operation,
		"processed": "true",
	}

	// Example: call external tool
	if cfg.Operation == "external" {
		cmd := exec.CommandContext(ctx, "echo", "hello from external tool")
		output, err := cmd.Output()
		if err != nil {
			result["external_output"] = err.Error()
		} else {
			result["external_output"] = string(output)
		}
	}

	return json.Marshal(result)
}

// Health checks if the plugin is healthy
func (p *Plugin) Health() error {
	return nil
}

// Destroy cleans up plugin resources
func (p *Plugin) Destroy(ctx context.Context) error {
	// Cleanup resources, close connections, etc.
	return nil
}

// Register is the entry point for plugin registration
func Register(host *PluginHost) error {
	return host.Register(&Plugin{})
}

// PluginHost is the context passed during registration
type PluginHost struct {
	WorkerID string
	Logger   Logger
}

// Logger interface for plugin logging
type Logger interface {
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
}

// Register registers a plugin with the host
func (h *PluginHost) Register(p *Plugin) error {
	// This would be implemented by the plugin host
	return nil
}
