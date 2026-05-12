package plugin

import (
	"context"
	"log"
	"time"
)

// PluginHost is the context passed to plugins during registration
type PluginHost struct {
	WorkerID string
	Logger   Logger
}

// Logger interface for plugin logging
type Logger interface {
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
}

// DefaultLogger is a simple logger implementation
type DefaultLogger struct {
	workerID string
}

// NewDefaultLogger creates a new default logger
func NewDefaultLogger(workerID string) *DefaultLogger {
	return &DefaultLogger{workerID: workerID}
}

func (l *DefaultLogger) Info(msg string, args ...interface{}) {
	log.Printf("[%s] INFO: %s", l.workerID, msg)
}

func (l *DefaultLogger) Error(msg string, args ...interface{}) {
	log.Printf("[%s] ERROR: %s", l.workerID, msg)
}

func (l *DefaultLogger) Warn(msg string, args ...interface{}) {
	log.Printf("[%s] WARN: %s", l.workerID, msg)
}

func (l *DefaultLogger) Debug(msg string, args ...interface{}) {
	log.Printf("[%s] DEBUG: %s", l.workerID, msg)
}

// SimpleLogger implements Logger interface for use in cmd/worker
type SimpleLogger struct {
	workerID string
}

func (l *SimpleLogger) Info(msg string, args ...interface{}) {
	log.Printf("[%s] INFO: %s", l.workerID, msg)
}

func (l *SimpleLogger) Error(msg string, args ...interface{}) {
	log.Printf("[%s] ERROR: %s", l.workerID, msg)
}

func (l *SimpleLogger) Warn(msg string, args ...interface{}) {
	log.Printf("[%s] WARN: %s", l.workerID, msg)
}

func (l *SimpleLogger) Debug(msg string, args ...interface{}) {
	log.Printf("[%s] DEBUG: %s", l.workerID, msg)
}

// HostContext holds context for plugin execution
type HostContext struct {
	WorkerID   string
	CancelFunc context.CancelFunc
	Timeout    time.Duration
}

// NewHostContext creates a new host context
func NewHostContext(workerID string) *HostContext {
	return &HostContext{
		WorkerID: workerID,
		Timeout:  5 * time.Minute,
	}
}
