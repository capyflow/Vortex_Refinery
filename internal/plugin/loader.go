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

	// P1-1 fixes: prevent reload loops and concurrent reloads
	reloadMu   sync.Mutex
	reloadCh   chan string
	stopReload chan struct{}

	// P1-2 fix: periodic full scan as fallback for missed fsnotify events
	stopFullScan chan struct{}
	fullScanMu   sync.RWMutex
	fullScanDone map[string]time.Time // plugin -> last scan time
}

// NewLoader creates a new plugin loader
func NewLoader(dir string, host *PluginHost, verifier *Verifier) *Loader {
	l := &Loader{
		plugins:      make(map[string]Plugin),
		metadata:     make(map[string]*PluginMetadata),
		host:         host,
		dir:          dir,
		verifier:     verifier,
		reloadCh:     make(chan string, 10),
		stopReload:   make(chan struct{}),
		stopFullScan: make(chan struct{}),
		fullScanDone: make(map[string]time.Time),
	}
	return l
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

	// P1-2: Start periodic full scan as fallback (every 5 minutes)
	go l.periodicFullScan(ctx)

	// P2-1: Protected watch loop with panic recovery + P1-1 debounced reload
	go l.watchLoopWithRecovery(ctx)

	l.host.Logger.Info("plugin file watcher started")
	return nil
}

// periodicFullScan runs a full plugin scan every 5 minutes as fallback
// This catches any fsnotify events that may have been missed
func (l *Loader) periodicFullScan(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-l.stopFullScan:
			return
		case <-ticker.C:
			l.doFullScan()
		}
	}
}

// doFullScan performs a full scan of plugin directory and detects changes
func (l *Loader) doFullScan() {
	l.mu.RLock()
	currentPlugins := make(map[string]bool)
	for name := range l.plugins {
		currentPlugins[name] = true
	}
	l.mu.RUnlock()

	entries, err := os.ReadDir(l.dir)
	if err != nil {
		l.host.Logger.Error("full scan failed: %v", err)
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pluginName := entry.Name()
		pluginDir := filepath.Join(l.dir, pluginName)
		pluginPath := filepath.Join(pluginDir, "plugin.json")

		if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
			// Plugin removed — trigger unload
			l.mu.RLock()
			_, exists := l.plugins[pluginName]
			l.mu.RUnlock()
			if exists {
				l.host.Logger.Info("full scan detected plugin removed: %s", pluginName)
				if err := l.Unload(pluginName); err != nil {
					l.host.Logger.Error("failed to unload removed plugin %s: %v", pluginName, err)
				}
			}
			continue
		}

		// Check modification time
		info, err := os.Stat(pluginPath)
		if err != nil {
			continue
		}
		mtime := info.ModTime()

		l.fullScanMu.RLock()
		lastDone, wasScanned := l.fullScanDone[pluginName]
		l.fullScanMu.RUnlock()

		if !wasScanned || mtime.After(lastDone) {
			// File is new or modified — queue reload
			select {
			case l.reloadCh <- pluginName:
				l.host.Logger.Info("full scan queued reload for: %s", pluginName)
			default:
				l.host.Logger.Warn("reload channel full, skipping: %s", pluginName)
			}
		}
	}
}

// watchLoopWithRecovery handles fsnotify events with debouncing and panic recovery
// P2-1: panic recovery prevents crash
// P1-1: debouncing prevents reload loops
func (l *Loader) watchLoopWithRecovery(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			l.host.Logger.Error("watchLoop panic recovered: %v", r)
			// Restart watcher in a new goroutine
			go l.watchLoopWithRecovery(ctx)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			close(l.stopReload)
			close(l.stopFullScan)
			return
		case event := <-l.watcher.Events:
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove) != 0 {
				rel, err := filepath.Rel(l.dir, event.Name)
				if err != nil {
					continue
				}
				parts := filepath.SplitList(rel)
				if len(parts) < 2 {
					continue
				}
				pluginName := parts[0]

				// P1-1: Debounce — queue reload request instead of direct call
				select {
				case l.reloadCh <- pluginName:
				default:
					l.host.Logger.Warn("reload channel full, dropping event for: %s", pluginName)
				}
			}
		case pluginName := <-l.reloadCh:
			l.host.Logger.Info("reload triggered: %s", pluginName)
			if err := l.safeReload(pluginName); err != nil {
				l.host.Logger.Error("failed to reload plugin %s: %v", pluginName, err)
			}
		case err := <-l.watcher.Errors:
			l.host.Logger.Error("watcher error: %v", err)
		}
	}
}

// safeReload is a thread-safe reload that prevents concurrent reloads of the same plugin
func (l *Loader) safeReload(name string) error {
	l.reloadMu.Lock()
	defer l.reloadMu.Unlock()
	return l.Reload(name)
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
	delete(l.fullScanDone, name)
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

	// Update full scan timestamp
	l.fullScanMu.Lock()
	l.fullScanDone[meta.Name] = time.Now()
	l.fullScanMu.Unlock()

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
// P1-3: reads binary only once
func (l *Loader) loadPluginFile(meta *PluginMetadata, path string) (Plugin, error) {
	var binaryData []byte
	if l.verifier != nil {
		var err error
		binaryData, err = os.ReadFile(path)
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to read plugin binary: %w", err)
		}
		// P2-2: properly handle verification error instead of ignoring
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
	Name          string                 `json:"name"`
	Version       string                 `json:"version"`
	Description   string                 `json:"description"`
	Entry         string                 `json:"entry"`
	Dependencies  []string               `json:"dependencies"`
	ConfigSchema  map[string]interface{} `json:"config_schema"`
	Author        string                 `json:"author,omitempty"`
	License       string                 `json:"license,omitempty"`
	Signature     string                 `json:"signature,omitempty"`
	SignAlgorithm string                 `json:"sign_algorithm,omitempty"`
	MinAPIVersion string                 `json:"min_api_version,omitempty"`
	MaxAPIVersion string                 `json:"max_api_version,omitempty"`
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
