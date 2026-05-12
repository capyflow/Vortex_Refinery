package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
	"sync"
)

// Loader discovers and loads plugins
type Loader struct {
	mu      sync.RWMutex
	plugins map[string]Plugin
	host    *PluginHost
	dir     string
}

// NewLoader creates a new plugin loader
func NewLoader(dir string, host *PluginHost) *Loader {
	return &Loader{
		plugins: make(map[string]Plugin),
		host:    host,
		dir:     dir,
	}
}

// DiscoverAndLoad discovers plugins in the plugin directory and loads them
func (l *Loader) DiscoverAndLoad(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Scan plugin directory
	entries, err := os.ReadDir(l.dir)
	if err != nil {
		return fmt.Errorf("failed to read plugin directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pluginPath := filepath.Join(l.dir, entry.Name(), "plugin.json")
		if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
			continue
		}

		// Load plugin metadata
		meta, err := l.loadMetadata(pluginPath)
		if err != nil {
			l.host.Logger.Error("failed to load plugin metadata: %v", err)
			continue
		}

		// Load plugin binary
		p, err := l.loadPlugin(filepath.Join(l.dir, entry.Name(), "main"))
		if err != nil {
			l.host.Logger.Error("failed to load plugin %s: %v", meta.Name, err)
			continue
		}

		l.plugins[meta.Name] = p
		l.host.Logger.Info("loaded plugin: %s v%s", meta.Name, meta.Version)
	}

	return nil
}

// loadMetadata loads plugin metadata from plugin.json
func (l *Loader) loadMetadata(path string) (*PluginMetadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var meta PluginMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}

	return &meta, nil
}

// loadPlugin loads a plugin from a .so file
func (l *Loader) loadPlugin(path string) (Plugin, error) {
	// Try to load as Go plugin (.so file)
	p, err := plugin.Open(path)
	if err != nil {
		// Fallback: treat as external command
		return nil, err
	}

	sym, err := p.Lookup("Plugin")
	if err != nil {
		return nil, fmt.Errorf("symbol Plugin not found: %w", err)
	}

	plug, ok := sym.(Plugin)
	if !ok {
		return nil, fmt.Errorf("invalid plugin type")
	}

	return plug, nil
}

// Get returns a plugin by name
func (l *Loader) Get(name string) (Plugin, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	p, ok := l.plugins[name]
	return p, ok
}

// List returns all loaded plugin names
func (l *Loader) List() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	names := make([]string, 0, len(l.plugins))
	for name := range l.plugins {
		names = append(names, name)
	}
	return names
}

// PluginMetadata represents plugin metadata from plugin.json
type PluginMetadata struct {
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	Entry       string                 `json:"entry"`
	Dependencies []string              `json:"dependencies"`
	ConfigSchema map[string]interface{} `json:"config_schema"`
}

// ExternalPluginRunner runs external tool plugins
type ExternalPluginRunner struct{}

// RunExternal runs an external command with the given input
func (r *ExternalPluginRunner) RunExternal(ctx context.Context, cmd string, args []string, input []byte) ([]byte, error) {
	c := exec.CommandContext(ctx, cmd, args...)
	c.Stdin = bytes.NewReader(input)
	return c.Output()
}
