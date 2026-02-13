package daemon

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// globalLogger is the singleton logger instance used across the daemon package
var globalLogger *DualLogger
var loggerInit sync.Once

// InitLogger initializes the global logger with dual output (file + stdout)
// This must be called before any other logging operation to ensure logs go to both destinations
func InitLogger(logPath string) error {
	var err error
	loggerInit.Do(func() {
		globalLogger, err = NewDualLogger(logPath)
	})
	return err
}

// CloseLogger closes the global logger's log file if it was opened
func CloseLogger() {
	if globalLogger != nil {
		globalLogger.Close()
	}
}

// Logger provides unified timestamped logging to both stdout and log file
// It uses the global logger if initialized, otherwise falls back to stdout only
func Logger(format string, args ...interface{}) {
	if globalLogger != nil {
		// Log using DualLogger which writes to both stdout and file
		globalLogger.Log(format, args...)
	} else {
		// Fallback: write to stdout only if logger not initialized
		timestamp := time.Now().Format("2006-01-02T15:04:05")
		fmt.Printf("[%s] %s\n", timestamp, fmt.Sprintf(format, args...))
	}
}

// GetLogger returns the global logger instance
func GetLogger() *DualLogger {
	return globalLogger
}

// GetLogWriter returns an io.Writer that writes to both stdout and log file
// This is useful for redirecting subprocess output
func GetLogWriter() io.Writer {
	if globalLogger != nil {
		return globalLogger.GetStdout()
	}
	return os.Stdout
}

// GetStderrWriter returns an io.Writer that writes to both stderr and log file
func GetStderrWriter() io.Writer {
	if globalLogger != nil {
		return globalLogger.GetStderr()
	}
	return os.Stderr
}
