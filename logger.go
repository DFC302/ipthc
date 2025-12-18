package main

import (
	"fmt"
	"os"
	"time"
)

// ErrorLogger handles logging errors to a file
type ErrorLogger struct {
	file *os.File
}

// NewErrorLogger creates a new error logger that writes to the specified file
func NewErrorLogger(filename string) (*ErrorLogger, error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return &ErrorLogger{file: file}, nil
}

// Log writes an error entry to the log file
// Format: [timestamp] [mode] [input] error_message
func (l *ErrorLogger) Log(mode, input, message string) error {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logLine := fmt.Sprintf("%s [%s] %s %s\n", timestamp, mode, input, message)

	_, err := l.file.WriteString(logLine)
	if err != nil {
		return fmt.Errorf("failed to write to log: %w", err)
	}

	return nil
}

// Close closes the log file
func (l *ErrorLogger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}
