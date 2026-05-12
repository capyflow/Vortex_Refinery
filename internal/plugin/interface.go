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

	// Destroy cleans up plugin resources before unload
	// Called synchronously during hot-reload or shutdown
	// Must complete within 5 seconds
	Destroy(ctx context.Context) error
}

// RegisterFunc is the function signature for plugin registration
type RegisterFunc func(host *PluginHost) error

// NoOpPlugin is a no-operation plugin that can be embedded
type NoOpPlugin struct{}

func (p *NoOpPlugin) Name() string                                    { return "no-op" }
func (p *NoOpPlugin) Execute(ctx context.Context, input []byte, config json.RawMessage) ([]byte, error) {
	return input, nil
}
func (p *NoOpPlugin) Health() error                               { return nil }
func (p *NoOpPlugin) Destroy(ctx context.Context) error           { return nil }
