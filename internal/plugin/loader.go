package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Loader discovers, loads, and hot-reloads plugins
type Loader struct {
	mu       sync.RWMutex
	plugins  map[string]Plugin
	metadata map[string]*PluginMetadata
	host     *PluginHost
	dir      string
	watcher  *fsnotify.Watcher
	verifier *Verifier
}

// NewLoader creates a new plugin loader
func NewLoader(dir string, host *PluginHost, verifier *Verifier) *Loader {
	return &Loader{
		plugins:  make(map[string]Plugin),
		metadata: make(map[string]*PluginMetadata),
		host:     host,
		dir:      dir,
		verifier: verifier,
	}
}

// DiscoverAndLoad discovers plugins in the plugin directory and loads them
func (l *Loader) DiscoverAndLoad(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

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

		meta, err := l.loadMetadata(pluginPath)
		if err != nil {
			l.host.Logger.Error("failed to load plugin metadata: %v", err)
			continue
		}

		p, err := l.loadPluginFile(meta, filepath.Join(l.dir, entry.Name(), "main"))
		if err != nil {
			l.host.Logger.Error("failed to load plugin %s: %v", meta.Name, err)
			continue
		}

		l.plugins[meta.Name] = p
		l.metadata[meta.Name] = meta
		l.host.Logger.Info("loaded plugin: %s v%s", meta.Name, meta.Version)
	}

	return nil
}

// Watch starts file system monitoring for plugin hot-reload
func (l *Loader) Watch(ctx context.Context) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	l.watcher = watcher

	// Watch plugin subdirectories
	entries, err := os.ReadDir(l.dir)
	if err != nil {
		return fmt.Errorf("failed to read plugin directory: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			pluginDir := filepath.Join(l.dir, entry.Name())
			if err := watcher.Add(pluginDir); err != nil {
				l.host.Logger.Error("failed to watch plugin dir %s: %v", pluginDir, err)
			}
		}
	}
	if err := watcher.Add(l.dir); err != nil {
		return fmt.Errorf("failed to watch root dir: %w", err)
	}

	go l.watchLoop(ctx)
	l.host.Logger.Info("plugin file watcher started")
	return nil
}

// watchLoop handles file system events
func (l *Loader) watchLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			l.watcher.Close()
			return
		case event := <-l.watcher.Events:
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove) != 0 {
				// Determine plugin name from path
				rel, err := filepath.Rel(l.dir, event.Name)
				if err != nil {
					continue
				}
				parts := filepath.SplitList(rel)
				if len(parts) < 2 {
					continue
				}
				pluginName := parts[0]
				l.host.Logger.Info("plugin file changed: %s (%s)", pluginName, event.Op)
				// Trigger hot reload
				if err := l.Reload(pluginName); err != nil {
					l.host.Logger.Error("failed to reload plugin %s: %v", pluginName, err)
				}
			}
		case err := <-l.watcher.Errors:
			l.host.Logger.Error("watcher error: %v", err)
		}
	}
}

// Unload unloads a plugin by name
func (l *Loader) Unload(name string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	p, exists := l.plugins[name]
	if !exists {
		return fmt.Errorf("plugin not found: %s", name)
	}

	// Call destroy with timeout
	destroyCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := p.Destroy(destroyCtx); err != nil {
		l.host.Logger.Error("plugin %s destroy failed: %v", name, err)
	}

	delete(l.plugins, name)
	delete(l.metadata, name)
	l.host.Logger.Info("plugin unloaded: %s", name)
	return nil
}

// Reload hot-reloads a single plugin
func (l *Loader) Reload(name string) error {
	l.mu.Lock()

	// Unload old version
	if old, exists := l.plugins[name]; exists {
		destroyCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		old.Destroy(destroyCtx)
		delete(l.plugins, name)
	}

	// Find plugin directory
	pluginDir := filepath.Join(l.dir, name)
	pluginPath := filepath.Join(pluginDir, "plugin.json")
	meta, err := l.loadMetadata(pluginPath)
	if err != nil {
		l.mu.Unlock()
		return fmt.Errorf("failed to load metadata for %s: %w", name, err)
	}

	// Load new version
	p, err := l.loadPluginFile(meta, filepath.Join(pluginDir, "main"))
	if err != nil {
		l.mu.Unlock()
		return fmt.Errorf("failed to load plugin %s: %w", name, err)
	}

	l.plugins[meta.Name] = p
	l.metadata[meta.Name] = meta
	l.mu.Unlock()

	l.host.Logger.Info("plugin reloaded: %s v%s", meta.Name, meta.Version)
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

// loadPluginFile loads a plugin from a .so file or external binary
func (l *Loader) loadPluginFile(meta *PluginMetadata, path string) (Plugin, error) {
	// Verify signature if required
	if l.verifier != nil {
		binaryData, err := os.ReadFile(path)
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to read plugin binary: %w", err)
		}
		if err := l.verifier.Verify(meta, binaryData); err != nil {
			return nil, fmt.Errorf("signature verification failed: %w", err)
		}
	}

	// Try Go plugin (.so)
	p, err := plugin.Open(path)
	if err == nil {
		sym, err := p.Lookup("Plugin")
		if err == nil {
			if plug, ok := sym.(Plugin); ok {
				return plug, nil
			}
		}
	}

	// Fallback: external command plugin
	return &ExternalPluginRunner{name: meta.Name, entry: path}, nil
}

// Get returns a plugin by name
func (l *Loader) Get(name string) (Plugin, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	p, ok := l.plugins[name]
	return p, ok
}

// GetMetadata returns plugin metadata by name
func (l *Loader) GetMetadata(name string) (*PluginMetadata, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	m, ok := l.metadata[name]
	return m, ok
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
	Name         string                 `json:"name"`
	Version      string                 `json:"version"`
	Description  string                 `json:"description"`
	Entry        string                 `json:"entry"`
	Dependencies []string               `json:"dependencies"`
	ConfigSchema map[string]interface{} `json:"config_schema"`
	// New fields
	Author        string   `json:"author,omitempty"`
	License       string   `json:"license,omitempty"`
	Signature     string   `json:"signature,omitempty"`
	SignAlgorithm string   `json:"sign_algorithm,omitempty"`
	MinAPIVersion string   `json:"min_api_version,omitempty"`
	MaxAPIVersion string   `json:"max_api_version,omitempty"`
}

// ExternalPluginRunner runs external tool plugins (fallback when .so unavailable)
type ExternalPluginRunner struct {
	name  string
	entry string
}

func (r *ExternalPluginRunner) Name() string {
	return r.name
}

func (r *ExternalPluginRunner) Execute(ctx context.Context, input []byte, config json.RawMessage) ([]byte, error) {
	cmd := exec.CommandContext(ctx, r.entry)
	cmd.Stdin = bytes.NewReader(input)
	return cmd.Output()
}

func (r *ExternalPluginRunner) Health() error {
	return nil
}

func (r *ExternalPluginRunner) Destroy(ctx context.Context) error {
	return nil
}
