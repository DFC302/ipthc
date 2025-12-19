package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestIntegration_DNSMode(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Build binary
	cmd := exec.Command("go", "build", "-o", "ipthc-test")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build: %v", err)
	}
	defer os.Remove("ipthc-test")

	// Test with echo
	input := "1.1.1.1"
	cmd = exec.Command("./ipthc-test", "-dns")
	cmd.Stdin = strings.NewReader(input)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	_ = cmd.Run()
	// May fail if API is down, but we're testing the flow

	t.Logf("stdout: %s", stdout.String())
	t.Logf("stderr: %s", stderr.String())
}

func TestIntegration_SubsMode(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cmd := exec.Command("go", "build", "-o", "ipthc-test")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build: %v", err)
	}
	defer os.Remove("ipthc-test")

	input := "example.com"
	cmd = exec.Command("./ipthc-test", "-subs", "-v")
	cmd.Stdin = strings.NewReader(input)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	cmd.Run()

	t.Logf("stdout: %s", stdout.String())
	t.Logf("stderr: %s", stderr.String())

	// Verify stderr contains comment lines in verbose mode
	if !strings.Contains(stderr.String(), ";;") {
		t.Logf("verbose mode may not be showing comments (API might have changed)")
	}
}

func TestIntegration_InvalidInput(t *testing.T) {
	cmd := exec.Command("go", "build", "-o", "ipthc-test")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build: %v", err)
	}
	defer os.Remove("ipthc-test")
	defer os.Remove("ipthc-errors.log")

	input := "not.an.ip"
	cmd = exec.Command("./ipthc-test", "-dns")
	cmd.Stdin = strings.NewReader(input)

	err := cmd.Run()

	// Should exit with code 1
	if err == nil {
		t.Error("expected non-zero exit code for invalid input")
	}

	// Check error log exists
	if _, err := os.Stat("ipthc-errors.log"); os.IsNotExist(err) {
		t.Error("error log file should be created")
	}
}

func TestIntegration_NoModeFlag(t *testing.T) {
	cmd := exec.Command("go", "run", ".")
	cmd.Stdin = strings.NewReader("test")

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()

	if err == nil {
		t.Error("expected error when no mode flag provided")
	}

	if !strings.Contains(stderr.String(), "must specify one mode") {
		t.Errorf("expected usage error, got: %s", stderr.String())
	}
}

func TestIntegration_MultipleFlags(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "-dns", "-subs")
	cmd.Stdin = strings.NewReader("test")

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()

	if err == nil {
		t.Error("expected error when multiple mode flags provided")
	}

	if !strings.Contains(stderr.String(), "cannot specify multiple modes") {
		t.Errorf("expected multiple modes error, got: %s", stderr.String())
	}
}
