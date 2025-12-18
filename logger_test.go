package main

import (
	"os"
	"strings"
	"testing"
)

func TestErrorLogger_Log(t *testing.T) {
	// Clean up any existing test log
	testLog := "test-errors.log"
	defer os.Remove(testLog)

	logger, err := NewErrorLogger(testLog)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Close()

	// Log an error
	err = logger.Log("dns", "1.1.1.1", "connection timeout")
	if err != nil {
		t.Fatalf("failed to log error: %v", err)
	}

	// Close to flush
	logger.Close()

	// Read log file
	content, err := os.ReadFile(testLog)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	logStr := string(content)

	// Verify format: [timestamp] [mode] [input] error_message
	if !strings.Contains(logStr, "[dns]") {
		t.Errorf("log missing mode: %s", logStr)
	}
	if !strings.Contains(logStr, "1.1.1.1") {
		t.Errorf("log missing input: %s", logStr)
	}
	if !strings.Contains(logStr, "connection timeout") {
		t.Errorf("log missing error message: %s", logStr)
	}
}

func TestErrorLogger_MultipleLogs(t *testing.T) {
	testLog := "test-errors-multi.log"
	defer os.Remove(testLog)

	logger, err := NewErrorLogger(testLog)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Close()

	// Log multiple errors
	logger.Log("dns", "1.1.1.1", "error 1")
	logger.Log("subs", "example.com", "error 2")
	logger.Log("cname", "test.com", "error 3")
	logger.Close()

	// Read and verify
	content, err := os.ReadFile(testLog)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	logStr := string(content)
	lines := strings.Split(strings.TrimSpace(logStr), "\n")

	if len(lines) != 3 {
		t.Errorf("expected 3 log lines, got %d", len(lines))
	}
}
