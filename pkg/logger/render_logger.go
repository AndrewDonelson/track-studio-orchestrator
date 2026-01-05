package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// RenderLogger handles verbose logging for video rendering process
type RenderLogger struct {
	songID    int
	logPath   string
	file      *os.File
	mu        sync.Mutex
	startTime time.Time
}

// NewRenderLogger creates a new render logger for a song
// Deletes existing log file if present and creates a new one
func NewRenderLogger(storagePath string, songID int) (*RenderLogger, error) {
	// Create logs directory structure: /storage/logs/song_id/
	logDir := filepath.Join(storagePath, "logs", fmt.Sprintf("%d", songID))
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	logPath := filepath.Join(logDir, "log.txt")

	// Delete existing log if present
	if _, err := os.Stat(logPath); err == nil {
		if err := os.Remove(logPath); err != nil {
			return nil, fmt.Errorf("failed to delete existing log: %w", err)
		}
	}

	// Create new log file
	file, err := os.Create(logPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	rl := &RenderLogger{
		songID:    songID,
		logPath:   logPath,
		file:      file,
		startTime: time.Now(),
	}

	// Write header
	rl.writeHeader()

	return rl, nil
}

// writeHeader writes the log file header
func (rl *RenderLogger) writeHeader() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	header := fmt.Sprintf(`================================================================================
TRACK STUDIO - VIDEO RENDER LOG
Song ID: %d
Started: %s
================================================================================

`, rl.songID, rl.startTime.Format("2006-01-02 15:04:05 MST"))

	rl.file.WriteString(header)
	rl.file.Sync()
}

// Phase logs the start of a processing phase
func (rl *RenderLogger) Phase(name string, description string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	elapsed := time.Since(rl.startTime).Round(time.Millisecond)
	msg := fmt.Sprintf("\n[%s] ========== PHASE: %s ==========\n", elapsed, name)
	if description != "" {
		msg += fmt.Sprintf("Description: %s\n", description)
	}
	msg += "\n"

	rl.file.WriteString(msg)
	rl.file.Sync()
}

// Info logs an informational message
func (rl *RenderLogger) Info(format string, args ...interface{}) {
	rl.log("INFO", format, args...)
}

// Debug logs a debug message with verbose details
func (rl *RenderLogger) Debug(format string, args ...interface{}) {
	rl.log("DEBUG", format, args...)
}

// Property logs a key-value property
func (rl *RenderLogger) Property(key string, value interface{}) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	elapsed := time.Since(rl.startTime).Round(time.Millisecond)
	msg := fmt.Sprintf("[%s] PROPERTY: %s = %v\n", elapsed, key, value)

	rl.file.WriteString(msg)
	rl.file.Sync()
}

// Command logs a command that will be executed
func (rl *RenderLogger) Command(cmdStr string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	elapsed := time.Since(rl.startTime).Round(time.Millisecond)
	msg := fmt.Sprintf("[%s] COMMAND: %s\n", elapsed, cmdStr)

	rl.file.WriteString(msg)
	rl.file.Sync()
}

// Output logs command output
func (rl *RenderLogger) Output(output string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if output == "" {
		return
	}

	elapsed := time.Since(rl.startTime).Round(time.Millisecond)
	msg := fmt.Sprintf("[%s] OUTPUT:\n%s\n", elapsed, output)

	rl.file.WriteString(msg)
	rl.file.Sync()
}

// Error logs an error message
func (rl *RenderLogger) Error(format string, args ...interface{}) {
	rl.log("ERROR", format, args...)
}

// Success logs a success message
func (rl *RenderLogger) Success(format string, args ...interface{}) {
	rl.log("SUCCESS", format, args...)
}

// log is the internal logging function
func (rl *RenderLogger) log(level string, format string, args ...interface{}) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	elapsed := time.Since(rl.startTime).Round(time.Millisecond)
	message := fmt.Sprintf(format, args...)
	msg := fmt.Sprintf("[%s] %s: %s\n", elapsed, level, message)

	rl.file.WriteString(msg)
	rl.file.Sync()
}

// Close closes the log file and writes footer
func (rl *RenderLogger) Close(success bool, finalMessage string) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	elapsed := time.Since(rl.startTime).Round(time.Millisecond)
	endTime := time.Now()

	status := "COMPLETED SUCCESSFULLY"
	if !success {
		status = "FAILED"
	}

	footer := fmt.Sprintf(`
================================================================================
RENDER %s
Duration: %s
Completed: %s
%s
================================================================================
`, status, elapsed, endTime.Format("2006-01-02 15:04:05 MST"), finalMessage)

	rl.file.WriteString(footer)
	rl.file.Sync()

	return rl.file.Close()
}

// GetLogPath returns the path to the log file
func (rl *RenderLogger) GetLogPath() string {
	return rl.logPath
}
