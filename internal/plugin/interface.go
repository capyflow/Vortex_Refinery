package plugin

import (
	"context"
	"encoding/json"
)

// Plugin is the interface that all plugins must implement
type Plugin interface {
	// Name returns the unique identifier of the plugin
	Name() string

	// Execute processes the input and returns the output
	Execute(ctx context.Context, input []byte, config json.RawMessage) ([]byte, error)

	// Health checks if the plugin is healthy
	Health() error
}

// PluginHost is the context passed to plugins during registration
type PluginHost struct {
	// WorkerID is the ID of the worker hosting this plugin
	WorkerID string
	// Logger can be used for plugin logging
	Logger Logger
}

// Logger interface for plugin logging
type Logger interface {
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
}

// RegisterFunc is the function signature for plugin registration
type RegisterFunc func(host *PluginHost) error
