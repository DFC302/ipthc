# ipthc CLI Tool Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a fast Go CLI tool to query ip.thc.org API for DNS reverse lookups, subdomain enumeration, and CNAME lookups, reading from stdin and outputting to stdout.

**Architecture:** Simple pipeline architecture with stdin reader → validator → API client → parser → stdout. Sequential request processing with configurable rate limiting. Error logging to file, graceful continuation on failures.

**Tech Stack:** Go 1.21+ (standard library only: net/http, flag, bufio, net, strings, time, os, fmt)

---

## Task 1: Project Initialization

**Files:**
- Create: `go.mod`
- Create: `README.md`
- Create: `.gitignore`

**Step 1: Initialize Go module**

Run:
```bash
go mod init github.com/USERNAME/ipthc
```

Expected: `go.mod` created with module path

**Step 2: Create .gitignore**

Create `.gitignore`:
```
# Binaries
ipthc
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test binaries
*.test

# Output
*.out

# Error logs
ipthc-errors.log

# IDE
.vscode/
.idea/
*.swp
*.swo
*~

# OS
.DS_Store
Thumbs.db
```

**Step 3: Create README.md**

Create `README.md`:
```markdown
# ipthc

Fast CLI tool for querying the ip.thc.org API.

## Installation

```bash
go install github.com/USERNAME/ipthc@latest
```

## Usage

### DNS Reverse Lookup
```bash
echo "1.1.1.1" | ipthc -dns
cat ips.txt | ipthc -dns
```

### Subdomain Enumeration
```bash
echo "example.com" | ipthc -subs
cat domains.txt | ipthc -subs
```

### CNAME Lookup
```bash
echo "example.com" | ipthc -cname
cat domains.txt | ipthc -cname
```

## Flags

- `-dns`: DNS reverse lookup (IP → domains)
- `-subs`: Subdomain enumeration
- `-cname`: CNAME lookup (domains pointing to target)
- `-v`: Verbose mode (show API metadata and errors)
- `-l <int>`: Results limit per request (default: 200)
- `-r <float>`: Rate limit delay in seconds (default: 1.0)

## Examples

```bash
# Verbose output with custom limit
cat domains.txt | ipthc -subs -v -l 100

# Custom rate limiting
cat ips.txt | ipthc -dns -r 2.0

# Pipeline with other tools
cat domains.txt | ipthc -subs | sort | uniq
```

## Error Handling

Errors are logged to `ipthc-errors.log` in the current directory. Use `-v` flag to see errors in stderr during execution.

Exit codes:
- `0`: All queries succeeded
- `1`: One or more queries failed (check error log)

## API

Uses https://ip.thc.org/ API endpoints:
- DNS: `https://ip.thc.org/{IP}?l={limit}`
- Subdomains: `https://ip.thc.org/sb/{domain}?l={limit}`
- CNAME: `https://ip.thc.org/cn/{domain}?l={limit}`
```

**Step 4: Commit initial setup**

Run:
```bash
git init
git add go.mod .gitignore README.md docs/
git commit -m "chore: initialize project structure"
```

Expected: Initial commit created

---

## Task 2: Error Logger Component

**Files:**
- Create: `logger.go`
- Create: `logger_test.go`

**Step 1: Write the failing test**

Create `logger_test.go`:
```go
package main

import (
	"os"
	"strings"
	"testing"
	"time"
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
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test -v -run TestErrorLogger
```

Expected: FAIL with "undefined: NewErrorLogger"

**Step 3: Write minimal implementation**

Create `logger.go`:
```go
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
```

**Step 4: Run test to verify it passes**

Run:
```bash
go test -v -run TestErrorLogger
```

Expected: PASS (all tests pass)

**Step 5: Commit**

Run:
```bash
git add logger.go logger_test.go
git commit -m "feat: add error logger component"
```

---

## Task 3: Input Validator Component

**Files:**
- Create: `validator.go`
- Create: `validator_test.go`

**Step 1: Write the failing test**

Create `validator_test.go`:
```go
package main

import (
	"testing"
)

func TestValidateIP(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"1.1.1.1", true},
		{"192.168.1.1", true},
		{"2606:4700:4700::1111", true},
		{"::1", true},
		{"256.1.1.1", false},
		{"not.an.ip", false},
		{"example.com", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			err := ValidateIP(tt.input)
			if tt.valid && err != nil {
				t.Errorf("ValidateIP(%q) returned error for valid IP: %v", tt.input, err)
			}
			if !tt.valid && err == nil {
				t.Errorf("ValidateIP(%q) returned nil for invalid IP", tt.input)
			}
		})
	}
}

func TestValidateDomain(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"example.com", true},
		{"sub.example.com", true},
		{"test.co.uk", true},
		{"a.b.c.d.e.f", true},
		{"example", false},           // no TLD
		{"", false},
		{".com", false},
		{"example.", false},
		{"ex ample.com", false},      // space
		{"example..com", false},      // double dot
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			err := ValidateDomain(tt.input)
			if tt.valid && err != nil {
				t.Errorf("ValidateDomain(%q) returned error for valid domain: %v", tt.input, err)
			}
			if !tt.valid && err == nil {
				t.Errorf("ValidateDomain(%q) returned nil for invalid domain", tt.input)
			}
		})
	}
}

func TestSanitizeInput(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"  example.com  ", "example.com"},
		{"\texample.com\n", "example.com"},
		{"example.com", "example.com"},
		{"  ", ""},
		{"\n", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := SanitizeInput(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeInput(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test -v -run TestValidate
```

Expected: FAIL with "undefined: ValidateIP, ValidateDomain, SanitizeInput"

**Step 3: Write minimal implementation**

Create `validator.go`:
```go
package main

import (
	"fmt"
	"net"
	"strings"
)

// SanitizeInput trims whitespace from input
func SanitizeInput(input string) string {
	return strings.TrimSpace(input)
}

// ValidateIP validates that the input is a valid IPv4 or IPv6 address
func ValidateIP(input string) error {
	ip := net.ParseIP(input)
	if ip == nil {
		return fmt.Errorf("invalid IP address: %s", input)
	}
	return nil
}

// ValidateDomain validates that the input is a valid domain name
func ValidateDomain(input string) error {
	if input == "" {
		return fmt.Errorf("domain cannot be empty")
	}

	// Must contain at least one dot (TLD required)
	if !strings.Contains(input, ".") {
		return fmt.Errorf("invalid domain: must contain TLD")
	}

	// Cannot start or end with dot
	if strings.HasPrefix(input, ".") || strings.HasSuffix(input, ".") {
		return fmt.Errorf("invalid domain: cannot start or end with dot")
	}

	// Cannot contain double dots
	if strings.Contains(input, "..") {
		return fmt.Errorf("invalid domain: cannot contain consecutive dots")
	}

	// Cannot contain spaces
	if strings.Contains(input, " ") {
		return fmt.Errorf("invalid domain: cannot contain spaces")
	}

	return nil
}
```

**Step 4: Run test to verify it passes**

Run:
```bash
go test -v -run TestValidate
go test -v -run TestSanitize
```

Expected: PASS (all tests pass)

**Step 5: Commit**

Run:
```bash
git add validator.go validator_test.go
git commit -m "feat: add input validator component"
```

---

## Task 4: Response Parser Component

**Files:**
- Create: `parser.go`
- Create: `parser_test.go`

**Step 1: Write the failing test**

Create `parser_test.go`:
```go
package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestResponseParser_Parse(t *testing.T) {
	input := `;;Subdomains For: segfault.net
;;Entries: 12/12
;;Rate Limit: You can make 249 requests
segfault.net
adm.segfault.net
lookup.segfault.net

lsd.segfault.net`

	parser := NewResponseParser(false)
	results := parser.Parse(input)

	expected := []string{
		"segfault.net",
		"adm.segfault.net",
		"lookup.segfault.net",
		"lsd.segfault.net",
	}

	if len(results) != len(expected) {
		t.Errorf("got %d results, want %d", len(results), len(expected))
	}

	for i, result := range results {
		if result != expected[i] {
			t.Errorf("result[%d] = %q, want %q", i, result, expected[i])
		}
	}
}

func TestResponseParser_ParseVerbose(t *testing.T) {
	input := `;;Subdomains For: example.com
;;Entries: 2/2
sub1.example.com
sub2.example.com`

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	parser := NewResponseParser(true)
	results := parser.Parse(input)

	// Restore stderr
	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	stderrOutput := buf.String()

	// Verify data results
	expected := []string{"sub1.example.com", "sub2.example.com"}
	if len(results) != len(expected) {
		t.Errorf("got %d results, want %d", len(results), len(expected))
	}

	// Verify comments were printed to stderr
	if !strings.Contains(stderrOutput, ";;Subdomains For:") {
		t.Errorf("verbose mode should print comments to stderr")
	}
	if !strings.Contains(stderrOutput, ";;Entries:") {
		t.Errorf("verbose mode should print all comment lines to stderr")
	}
}

func TestResponseParser_EmptyInput(t *testing.T) {
	parser := NewResponseParser(false)
	results := parser.Parse("")

	if len(results) != 0 {
		t.Errorf("expected empty results, got %d items", len(results))
	}
}

func TestResponseParser_OnlyComments(t *testing.T) {
	input := `;;Comment 1
;;Comment 2
;;Comment 3`

	parser := NewResponseParser(false)
	results := parser.Parse(input)

	if len(results) != 0 {
		t.Errorf("expected no results for comment-only input, got %d items", len(results))
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test -v -run TestResponseParser
```

Expected: FAIL with "undefined: NewResponseParser"

**Step 3: Write minimal implementation**

Create `parser.go`:
```go
package main

import (
	"fmt"
	"os"
	"strings"
)

// ResponseParser handles parsing API responses
type ResponseParser struct {
	Verbose bool
}

// NewResponseParser creates a new response parser
func NewResponseParser(verbose bool) *ResponseParser {
	return &ResponseParser{Verbose: verbose}
}

// Parse extracts data lines from API response
// Comment lines (starting with ;) are printed to stderr if verbose mode is enabled
// Returns slice of data lines (non-comment, non-empty)
func (p *ResponseParser) Parse(body string) []string {
	lines := strings.Split(body, "\n")
	results := []string{}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines
		if trimmed == "" {
			continue
		}

		// Comment line (starts with ;)
		if strings.HasPrefix(trimmed, ";") {
			if p.Verbose {
				fmt.Fprintln(os.Stderr, trimmed)
			}
			continue
		}

		// Data line - add to results
		results = append(results, trimmed)
	}

	return results
}
```

**Step 4: Run test to verify it passes**

Run:
```bash
go test -v -run TestResponseParser
```

Expected: PASS (all tests pass)

**Step 5: Commit**

Run:
```bash
git add parser.go parser_test.go
git commit -m "feat: add response parser component"
```

---

## Task 5: API Client Component

**Files:**
- Create: `client.go`
- Create: `client_test.go`

**Step 1: Write the failing test**

Create `client_test.go`:
```go
package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAPIClient_QueryDNS(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/1.1.1.1") {
			t.Errorf("expected path to contain IP, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("l") != "200" {
			t.Errorf("expected limit param l=200, got %s", r.URL.Query().Get("l"))
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(";;DNS Response\ndomain1.com\ndomain2.com"))
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, 200, 0, false)
	body, err := client.QueryDNS("1.1.1.1")

	if err != nil {
		t.Fatalf("QueryDNS failed: %v", err)
	}

	if !strings.Contains(body, "domain1.com") {
		t.Errorf("expected response to contain domain1.com")
	}
}

func TestAPIClient_QuerySubdomains(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/sb/example.com") {
			t.Errorf("expected path /sb/example.com, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(";;Subdomains\nsub1.example.com\nsub2.example.com"))
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, 200, 0, false)
	body, err := client.QuerySubdomains("example.com")

	if err != nil {
		t.Fatalf("QuerySubdomains failed: %v", err)
	}

	if !strings.Contains(body, "sub1.example.com") {
		t.Errorf("expected response to contain sub1.example.com")
	}
}

func TestAPIClient_QueryCNAME(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/cn/example.com") {
			t.Errorf("expected path /cn/example.com, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(";;CNAME Response\ncname1.com\ncname2.com"))
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, 200, 0, false)
	body, err := client.QueryCNAME("example.com")

	if err != nil {
		t.Fatalf("QueryCNAME failed: %v", err)
	}

	if !strings.Contains(body, "cname1.com") {
		t.Errorf("expected response to contain cname1.com")
	}
}

func TestAPIClient_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Server Error"))
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, 200, 0, false)
	_, err := client.QueryDNS("1.1.1.1")

	if err == nil {
		t.Errorf("expected error for 500 status, got nil")
	}

	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error should mention status code 500: %v", err)
	}
}

func TestAPIClient_RateLimit(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, 200, 0.1, false)

	start := time.Now()
	client.QueryDNS("1.1.1.1")
	client.QueryDNS("1.1.1.2")
	elapsed := time.Since(start)

	// Should take at least 100ms due to rate limit
	if elapsed < 100*time.Millisecond {
		t.Errorf("rate limiting not working: took %v, expected >= 100ms", elapsed)
	}

	if requestCount != 2 {
		t.Errorf("expected 2 requests, got %d", requestCount)
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test -v -run TestAPIClient
```

Expected: FAIL with "undefined: NewAPIClient"

**Step 3: Write minimal implementation**

Create `client.go`:
```go
package main

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

// APIClient handles API requests to ip.thc.org
type APIClient struct {
	BaseURL    string
	Limit      int
	RateLimit  float64
	HTTPClient *http.Client
	Verbose    bool
	lastRequest time.Time
}

// NewAPIClient creates a new API client
func NewAPIClient(baseURL string, limit int, rateLimit float64, verbose bool) *APIClient {
	return &APIClient{
		BaseURL:   baseURL,
		Limit:     limit,
		RateLimit: rateLimit,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		Verbose: verbose,
	}
}

// QueryDNS performs a reverse DNS lookup for an IP address
func (c *APIClient) QueryDNS(ip string) (string, error) {
	url := fmt.Sprintf("%s/%s?l=%d", c.BaseURL, ip, c.Limit)
	return c.makeRequest(url)
}

// QuerySubdomains performs subdomain enumeration for a domain
func (c *APIClient) QuerySubdomains(domain string) (string, error) {
	url := fmt.Sprintf("%s/sb/%s?l=%d", c.BaseURL, domain, c.Limit)
	return c.makeRequest(url)
}

// QueryCNAME performs CNAME lookup for a domain
func (c *APIClient) QueryCNAME(domain string) (string, error) {
	url := fmt.Sprintf("%s/cn/%s?l=%d", c.BaseURL, domain, c.Limit)
	return c.makeRequest(url)
}

// makeRequest performs the HTTP request with rate limiting
func (c *APIClient) makeRequest(url string) (string, error) {
	// Apply rate limiting
	if c.RateLimit > 0 && !c.lastRequest.IsZero() {
		elapsed := time.Since(c.lastRequest)
		delay := time.Duration(c.RateLimit * float64(time.Second))
		if elapsed < delay {
			time.Sleep(delay - elapsed)
		}
	}

	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	c.lastRequest = time.Now()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return string(body), nil
}
```

**Step 4: Run test to verify it passes**

Run:
```bash
go test -v -run TestAPIClient
```

Expected: PASS (all tests pass)

**Step 5: Commit**

Run:
```bash
git add client.go client_test.go
git commit -m "feat: add API client component"
```

---

## Task 6: Main Orchestrator

**Files:**
- Create: `main.go`

**Step 1: Write the implementation**

Create `main.go`:
```go
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
)

const (
	defaultBaseURL   = "https://ip.thc.org"
	defaultLimit     = 200
	defaultRateLimit = 1.0
	errorLogFile     = "ipthc-errors.log"
)

func main() {
	// Define flags
	dnsMode := flag.Bool("dns", false, "DNS reverse lookup mode")
	subsMode := flag.Bool("subs", false, "Subdomain enumeration mode")
	cnameMode := flag.Bool("cname", false, "CNAME lookup mode")
	verbose := flag.Bool("v", false, "Verbose mode (show API metadata and errors)")
	limit := flag.Int("l", defaultLimit, "Results limit per request")
	rateLimit := flag.Float64("r", defaultRateLimit, "Rate limit delay in seconds")

	flag.Parse()

	// Validate flags
	modeCount := 0
	var mode string
	if *dnsMode {
		modeCount++
		mode = "dns"
	}
	if *subsMode {
		modeCount++
		mode = "subs"
	}
	if *cnameMode {
		modeCount++
		mode = "cname"
	}

	if modeCount == 0 {
		fmt.Fprintln(os.Stderr, "Error: must specify one mode: -dns, -subs, or -cname")
		flag.Usage()
		os.Exit(1)
	}

	if modeCount > 1 {
		fmt.Fprintln(os.Stderr, "Error: cannot specify multiple modes")
		flag.Usage()
		os.Exit(1)
	}

	if *limit < 1 {
		fmt.Fprintln(os.Stderr, "Error: limit must be positive")
		os.Exit(1)
	}

	if *rateLimit < 0 {
		fmt.Fprintln(os.Stderr, "Error: rate limit cannot be negative")
		os.Exit(1)
	}

	// Initialize components
	logger, err := NewErrorLogger(errorLogFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize error logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Close()

	client := NewAPIClient(defaultBaseURL, *limit, *rateLimit, *verbose)
	parser := NewResponseParser(*verbose)

	// Process stdin
	scanner := bufio.NewScanner(os.Stdin)
	failureCount := 0

	for scanner.Scan() {
		input := SanitizeInput(scanner.Text())

		// Skip empty lines and comments
		if input == "" || input[0] == '#' {
			continue
		}

		// Validate and query based on mode
		var body string
		var err error

		switch mode {
		case "dns":
			if err = ValidateIP(input); err != nil {
				failureCount++
				logger.Log(mode, input, err.Error())
				if *verbose {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				}
				continue
			}
			body, err = client.QueryDNS(input)

		case "subs":
			if err = ValidateDomain(input); err != nil {
				failureCount++
				logger.Log(mode, input, err.Error())
				if *verbose {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				}
				continue
			}
			body, err = client.QuerySubdomains(input)

		case "cname":
			if err = ValidateDomain(input); err != nil {
				failureCount++
				logger.Log(mode, input, err.Error())
				if *verbose {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				}
				continue
			}
			body, err = client.QueryCNAME(input)
		}

		if err != nil {
			failureCount++
			logger.Log(mode, input, err.Error())
			if *verbose {
				fmt.Fprintf(os.Stderr, "Error querying %s: %v\n", input, err)
			}
			continue
		}

		// Parse and output results
		results := parser.Parse(body)
		for _, result := range results {
			fmt.Println(result)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
		os.Exit(1)
	}

	// Exit with failure code if any queries failed
	if failureCount > 0 {
		os.Exit(1)
	}
}
```

**Step 2: Test manually with DNS mode**

Run:
```bash
go build -o ipthc
echo "1.1.1.1" | ./ipthc -dns -v
```

Expected:
- HTTP request to ip.thc.org
- Response parsed and domains printed to stdout
- Comments visible in stderr (verbose mode)

**Step 3: Test with invalid input**

Run:
```bash
echo "invalid.ip" | ./ipthc -dns
cat ipthc-errors.log
```

Expected:
- No output to stdout
- Error logged to ipthc-errors.log
- Exit code 1

**Step 4: Test subs mode**

Run:
```bash
echo "example.com" | ./ipthc -subs -v
```

Expected:
- Subdomains printed to stdout
- Comments to stderr

**Step 5: Commit**

Run:
```bash
git add main.go
git commit -m "feat: add main orchestrator"
```

---

## Task 7: Integration Testing

**Files:**
- Create: `integration_test.go`

**Step 1: Write integration tests**

Create `integration_test.go`:
```go
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

	err := cmd.Run()
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
```

**Step 2: Run integration tests**

Run:
```bash
go test -v -run TestIntegration
```

Expected: Tests pass (or skip if using -short)

**Step 3: Run all tests**

Run:
```bash
go test -v ./...
```

Expected: All unit and integration tests pass

**Step 4: Commit**

Run:
```bash
git add integration_test.go
git commit -m "test: add integration tests"
```

---

## Task 8: Final Touches and Documentation

**Files:**
- Modify: `README.md`
- Create: `.github/workflows/test.yml` (optional)

**Step 1: Update README with installation instructions**

Edit `README.md` to replace `USERNAME` placeholder:
```markdown
## Installation

Replace `USERNAME` with your GitHub username, then run:

```bash
go install github.com/USERNAME/ipthc@latest
```

Or clone and build locally:

```bash
git clone https://github.com/USERNAME/ipthc
cd ipthc
go build -o ipthc
./ipthc -h
```
```

**Step 2: Build final binary**

Run:
```bash
go build -o ipthc
./ipthc -h
```

Expected: Usage message displayed

**Step 3: Test with real API (optional)**

Run:
```bash
echo "1.1.1.1" | ./ipthc -dns -v -l 10
```

Expected: Real results from ip.thc.org API

**Step 4: Run final test suite**

Run:
```bash
go test -v ./...
go test -race ./...
```

Expected: All tests pass, no race conditions

**Step 5: Commit**

Run:
```bash
git add README.md
git commit -m "docs: update installation instructions"
```

---

## Task 9: Git Repository Setup for Publishing

**Step 1: Initialize git repository (if not already done)**

Run:
```bash
git status
```

Expected: Should show git repository initialized

**Step 2: Create GitHub repository**

Instructions:
1. Go to https://github.com/new
2. Create repository named `ipthc`
3. Do NOT initialize with README (we already have one)
4. Copy the repository URL

**Step 3: Add remote and push**

Run (replace USERNAME):
```bash
git remote add origin https://github.com/USERNAME/ipthc.git
git branch -M main
git push -u origin main
```

Expected: Code pushed to GitHub

**Step 4: Create initial release tag**

Run:
```bash
git tag v1.0.0
git push origin v1.0.0
```

Expected: Tag v1.0.0 created and pushed

**Step 5: Verify go install works**

Run (replace USERNAME):
```bash
go install github.com/USERNAME/ipthc@latest
which ipthc
ipthc -h
```

Expected:
- Binary installed to `$GOPATH/bin/ipthc`
- Help message displayed

---

## Task 10: Create Example Usage Script

**Files:**
- Create: `examples/usage.sh`

**Step 1: Create examples directory and script**

Create `examples/usage.sh`:
```bash
#!/bin/bash

# ipthc usage examples

echo "=== DNS Reverse Lookup ==="
echo "1.1.1.1" | ipthc -dns

echo -e "\n=== Subdomain Enumeration ==="
echo "example.com" | ipthc -subs -l 20

echo -e "\n=== CNAME Lookup ==="
echo "example.com" | ipthc -cname

echo -e "\n=== Batch Processing from File ==="
cat << EOF > /tmp/test-domains.txt
example.com
google.com
github.com
EOF

cat /tmp/test-domains.txt | ipthc -subs -v -l 10

echo -e "\n=== Pipeline with other tools ==="
echo "example.com" | ipthc -subs | head -5

rm /tmp/test-domains.txt
```

**Step 2: Make executable**

Run:
```bash
chmod +x examples/usage.sh
```

**Step 3: Test examples**

Run:
```bash
./examples/usage.sh
```

Expected: Examples run successfully

**Step 4: Commit**

Run:
```bash
git add examples/
git commit -m "docs: add usage examples"
git push
```

---

## Completion Checklist

- [ ] Project initialized with go.mod
- [ ] Error logger component implemented and tested
- [ ] Input validator component implemented and tested
- [ ] Response parser component implemented and tested
- [ ] API client component implemented and tested
- [ ] Main orchestrator implemented
- [ ] Integration tests passing
- [ ] README documentation complete
- [ ] Git repository created on GitHub
- [ ] Tagged release v1.0.0
- [ ] Verified `go install github.com/USERNAME/ipthc@latest` works
- [ ] Example usage scripts created

## Post-Implementation

After completing all tasks:

1. **Update README** with your actual GitHub username
2. **Test installation** on a clean system
3. **Create GitHub release** with release notes
4. **Consider adding**:
   - GitHub Actions for CI/CD
   - More comprehensive error messages
   - Support for reading from files directly with `-f` flag
   - Output formatting options (JSON, CSV)
   - Proxy support for API requests

## Testing the Final Product

```bash
# Install from GitHub
go install github.com/USERNAME/ipthc@latest

# Test all modes
echo "1.1.1.1" | ipthc -dns
echo "example.com" | ipthc -subs
echo "example.com" | ipthc -cname

# Test error handling
echo "invalid" | ipthc -dns
echo $?  # Should be 1
cat ipthc-errors.log

# Test rate limiting
seq 1 5 | xargs -I {} echo "1.1.1.{}" | ipthc -dns -r 0.5 -v

# Test with file
echo -e "example.com\ngoogle.com\ngithub.com" > domains.txt
cat domains.txt | ipthc -subs -l 50
```
