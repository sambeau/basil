package server

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher monitors files for changes and triggers reload actions
type Watcher struct {
	watcher     *fsnotify.Watcher
	server      *Server
	configPath  string
	handlerDirs []string
	staticDirs  []string
	stdout      io.Writer
	stderr      io.Writer

	// Track last change time to debounce rapid changes
	mu         sync.Mutex
	lastChange time.Time
	changeSeq  uint64 // Incremented on each file change for live reload
}

// NewWatcher creates a file watcher for hot reload in dev mode
func NewWatcher(s *Server, configPath string, stdout, stderr io.Writer) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		watcher:    fsWatcher,
		server:     s,
		configPath: configPath,
		stdout:     stdout,
		stderr:     stderr,
	}

	// Collect directories to watch
	w.handlerDirs = w.collectHandlerDirs()
	w.staticDirs = w.collectStaticDirs()

	return w, nil
}

// collectHandlerDirs returns unique directories containing handler scripts
func (w *Watcher) collectHandlerDirs() []string {
	dirs := make(map[string]bool)
	
	// In site mode, watch the handler root (parent of site directory)
	if w.server.config.Site != "" {
		handlerRoot := filepath.Dir(w.server.config.Site)
		dirs[handlerRoot] = true
	}
	
	// In route mode, watch directories containing handlers
	for _, route := range w.server.config.Routes {
		dir := filepath.Dir(route.Handler)
		dirs[dir] = true
	}

	result := make([]string, 0, len(dirs))
	for dir := range dirs {
		result = append(result, dir)
	}
	return result
}

// collectStaticDirs returns directories configured for static serving
func (w *Watcher) collectStaticDirs() []string {
	dirs := make([]string, 0)
	for _, static := range w.server.config.Static {
		if static.Root != "" {
			dirs = append(dirs, static.Root)
		}
	}
	return dirs
}

// Start begins watching for file changes
func (w *Watcher) Start(ctx context.Context) error {
	// Watch config file
	if w.configPath != "" {
		configDir := filepath.Dir(w.configPath)
		if err := w.watcher.Add(configDir); err != nil {
			w.logError("failed to watch config dir %s: %v", configDir, err)
		} else {
			w.logInfo("watching config: %s", w.configPath)
		}
	}

	// Watch handler directories (recursively)
	for _, dir := range w.handlerDirs {
		if err := w.watchDirRecursive(dir); err != nil {
			w.logError("failed to watch handler dir %s: %v", dir, err)
		} else {
			w.logInfo("watching handlers: %s", dir)
		}
	}

	// Watch static directories
	for _, dir := range w.staticDirs {
		if err := w.watchDirRecursive(dir); err != nil {
			w.logError("failed to watch static dir %s: %v", dir, err)
		} else {
			w.logInfo("watching static: %s", dir)
		}
	}

	// Start event loop
	go w.eventLoop(ctx)

	return nil
}

// watchDirRecursive adds a directory and its subdirectories to the watch list
func (w *Watcher) watchDirRecursive(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if info.IsDir() {
			// Skip hidden directories
			if strings.HasPrefix(info.Name(), ".") && path != root {
				return filepath.SkipDir
			}
			return w.watcher.Add(path)
		}
		return nil
	})
}

// eventLoop processes file system events
func (w *Watcher) eventLoop(ctx context.Context) {
	// Debounce duration - wait for rapid changes to settle
	const debounce = 100 * time.Millisecond

	for {
		select {
		case <-ctx.Done():
			return

		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			// Only handle write and create events
			if !event.Has(fsnotify.Write) && !event.Has(fsnotify.Create) {
				continue
			}

			// Debounce rapid changes
			w.mu.Lock()
			if time.Since(w.lastChange) < debounce {
				w.mu.Unlock()
				continue
			}
			w.lastChange = time.Now()
			w.changeSeq++
			w.mu.Unlock()

			w.handleFileChange(event.Name)

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			w.logError("watcher error: %v", err)
		}
	}
}

// handleFileChange processes a file change event
func (w *Watcher) handleFileChange(path string) {
	// Check if it's the config file
	if w.configPath != "" && filepath.Base(path) == filepath.Base(w.configPath) {
		w.logInfo("config changed: %s (browser will reload, restart server for config changes to take effect)", path)
		return
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".pars", ".parsley":
		w.logInfo("handler changed: %s", path)
		// Scripts are reloaded on next request (no caching in dev mode)

	case ".css", ".js":
		w.logInfo("asset changed: %s", path)
		// Rebuild asset bundle if this is under handlers directory
		if w.isHandlerAsset(path) {
			if err := w.server.assetBundle.Rebuild(); err != nil {
				w.logError("failed to rebuild asset bundle: %v", err)
			}
		}

	case ".html", ".htm":
		w.logInfo("static file changed: %s", path)

	default:
		// Ignore other files
		return
	}
}

// isHandlerAsset checks if a file path is under one of the handler directories.
func (w *Watcher) isHandlerAsset(path string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	for _, dir := range w.handlerDirs {
		absDir, err := filepath.Abs(dir)
		if err != nil {
			continue
		}
		// Check if path is under this directory
		relPath, err := filepath.Rel(absDir, absPath)
		if err == nil && !strings.HasPrefix(relPath, "..") {
			return true
		}
	}
	return false
}

// GetChangeSeq returns the current change sequence number for live reload
func (w *Watcher) GetChangeSeq() uint64 {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.changeSeq
}

// TriggerReload increments the change sequence to trigger browser reload
func (w *Watcher) TriggerReload() {
	w.mu.Lock()
	w.changeSeq++
	w.mu.Unlock()
}

// Close stops the watcher
func (w *Watcher) Close() error {
	return w.watcher.Close()
}

func (w *Watcher) logInfo(format string, args ...interface{}) {
	fmt.Fprintf(w.stdout, "[WATCH] "+format+"\n", args...)
}

func (w *Watcher) logError(format string, args ...interface{}) {
	fmt.Fprintf(w.stderr, "[WATCH ERROR] "+format+"\n", args...)
}
