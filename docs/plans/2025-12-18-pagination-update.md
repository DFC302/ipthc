# Pagination Feature Update

**Date:** 2025-12-18
**Update to:** 2025-12-18-ipthc-cli-tool.md

This document contains updated implementations for Task 4 and Task 5 to add **Smart Auto-Pagination** support.

---

## Updated Task 4: Response Parser Component (WITH PAGINATION)

**Files:**
- Create: `parser.go`
- Create: `parser_test.go`

### Changes from Original

The parser now:
1. Extracts pagination metadata from `;;Entries: X/Y` lines
2. Returns both data results AND pagination info (current/total counts)
3. Allows the API client to determine if more results are available

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
;;Entries: 4/12
;;Rate Limit: You can make 249 requests
segfault.net
adm.segfault.net
lookup.segfault.net

lsd.segfault.net`

	parser := NewResponseParser(false)
	result := parser.Parse(input)

	expected := []string{
		"segfault.net",
		"adm.segfault.net",
		"lookup.segfault.net",
		"lsd.segfault.net",
	}

	if len(result.Data) != len(expected) {
		t.Errorf("got %d results, want %d", len(result.Data), len(expected))
	}

	for i, data := range result.Data {
		if data != expected[i] {
			t.Errorf("result[%d] = %q, want %q", i, data, expected[i])
		}
	}

	// Check pagination info
	if result.CurrentCount != 4 {
		t.Errorf("CurrentCount = %d, want 4", result.CurrentCount)
	}

	if result.TotalCount != 12 {
		t.Errorf("TotalCount = %d, want 12", result.TotalCount)
	}

	if !result.HasMore() {
		t.Error("HasMore() should return true when 4 < 12")
	}
}

func TestResponseParser_ParseComplete(t *testing.T) {
	input := `;;Entries: 12/12
sub1.example.com
sub2.example.com`

	parser := NewResponseParser(false)
	result := parser.Parse(input)

	if result.CurrentCount != 12 {
		t.Errorf("CurrentCount = %d, want 12", result.CurrentCount)
	}

	if result.TotalCount != 12 {
		t.Errorf("TotalCount = %d, want 12", result.TotalCount)
	}

	if result.HasMore() {
		t.Error("HasMore() should return false when counts are equal")
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
	result := parser.Parse(input)

	// Restore stderr
	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	stderrOutput := buf.String()

	// Verify data results
	expected := []string{"sub1.example.com", "sub2.example.com"}
	if len(result.Data) != len(expected) {
		t.Errorf("got %d results, want %d", len(result.Data), len(expected))
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
	result := parser.Parse("")

	if len(result.Data) != 0 {
		t.Errorf("expected empty results, got %d items", len(result.Data))
	}

	if result.HasMore() {
		t.Error("empty input should not have more results")
	}
}

func TestResponseParser_OnlyComments(t *testing.T) {
	input := `;;Comment 1
;;Comment 2
;;Comment 3`

	parser := NewResponseParser(false)
	result := parser.Parse(input)

	if len(result.Data) != 0 {
		t.Errorf("expected no results for comment-only input, got %d items", len(result.Data))
	}
}

func TestResponseParser_NoEntriesLine(t *testing.T) {
	input := `sub1.example.com
sub2.example.com`

	parser := NewResponseParser(false)
	result := parser.Parse(input)

	// Should still parse data
	if len(result.Data) != 2 {
		t.Errorf("got %d results, want 2", len(result.Data))
	}

	// No pagination info means counts are 0
	if result.CurrentCount != 0 {
		t.Errorf("CurrentCount should be 0 when no ;;Entries line")
	}

	if result.TotalCount != 0 {
		t.Errorf("TotalCount should be 0 when no ;;Entries line")
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test -v -run TestResponseParser
```

Expected: FAIL with "undefined: NewResponseParser" or type mismatch errors

**Step 3: Write minimal implementation**

Create `parser.go`:
```go
package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// ParseResult contains both data and pagination metadata
type ParseResult struct {
	Data         []string // The actual result data lines
	CurrentCount int      // Number of results in this response
	TotalCount   int      // Total number of results available
}

// HasMore returns true if there are more results available
func (r *ParseResult) HasMore() bool {
	// If pagination info wasn't found, assume we have everything
	if r.CurrentCount == 0 && r.TotalCount == 0 {
		return false
	}
	return r.CurrentCount < r.TotalCount
}

// ResponseParser handles parsing API responses
type ResponseParser struct {
	Verbose bool
}

// NewResponseParser creates a new response parser
func NewResponseParser(verbose bool) *ResponseParser {
	return &ResponseParser{Verbose: verbose}
}

// Regular expression to match ;;Entries: X/Y format
var entriesRegex = regexp.MustCompile(`;;Entries:\s*(\d+)/(\d+)`)

// Parse extracts data lines and pagination metadata from API response
// Comment lines (starting with ;) are printed to stderr if verbose mode is enabled
// Returns ParseResult with data and pagination info
func (p *ResponseParser) Parse(body string) *ParseResult {
	lines := strings.Split(body, "\n")
	result := &ParseResult{
		Data:         []string{},
		CurrentCount: 0,
		TotalCount:   0,
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines
		if trimmed == "" {
			continue
		}

		// Comment line (starts with ;)
		if strings.HasPrefix(trimmed, ";") {
			// Try to extract pagination info from ;;Entries: line
			if matches := entriesRegex.FindStringSubmatch(trimmed); matches != nil {
				if len(matches) == 3 {
					current, _ := strconv.Atoi(matches[1])
					total, _ := strconv.Atoi(matches[2])
					result.CurrentCount = current
					result.TotalCount = total
				}
			}

			if p.Verbose {
				fmt.Fprintln(os.Stderr, trimmed)
			}
			continue
		}

		// Data line - add to results
		result.Data = append(result.Data, trimmed)
	}

	return result
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
git commit -m "feat: add response parser with pagination support"
```

---

## Updated Task 5: API Client Component (WITH PAGINATION)

**Files:**
- Create: `client.go`
- Create: `client_test.go`

### Changes from Original

The API client now:
1. Makes initial request
2. Parses response to check for pagination
3. Automatically re-requests with full limit if more results exist
4. Respects rate limiting between pagination requests
5. Returns combined results

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
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(";;Entries: 2/2\n;;DNS Response\ndomain1.com\ndomain2.com"))
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
		w.Write([]byte(";;Entries: 2/2\n;;Subdomains\nsub1.example.com\nsub2.example.com"))
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
		w.Write([]byte(";;Entries: 2/2\n;;CNAME Response\ncname1.com\ncname2.com"))
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

func TestAPIClient_Pagination(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		limit := r.URL.Query().Get("l")

		if requestCount == 1 {
			// First request: return partial results
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(";;Entries: 2/10\nsub1.example.com\nsub2.example.com"))
		} else if requestCount == 2 {
			// Second request: should have limit=10
			if limit != "10" {
				t.Errorf("second request should have l=10, got l=%s", limit)
			}
			// Return all results
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(";;Entries: 10/10\nsub1.example.com\nsub2.example.com\nsub3.example.com\nsub4.example.com\nsub5.example.com\nsub6.example.com\nsub7.example.com\nsub8.example.com\nsub9.example.com\nsub10.example.com"))
		}
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, 200, 0, false)
	body, err := client.QuerySubdomains("example.com")

	if err != nil {
		t.Fatalf("QuerySubdomains with pagination failed: %v", err)
	}

	// Should have made 2 requests
	if requestCount != 2 {
		t.Errorf("expected 2 requests (initial + pagination), got %d", requestCount)
	}

	// Should have all 10 results
	if !strings.Contains(body, "sub10.example.com") {
		t.Errorf("expected response to contain all results including sub10.example.com")
	}
}

func TestAPIClient_NoPagination(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
		// All results in first response
		w.Write([]byte(";;Entries: 5/5\nsub1.example.com\nsub2.example.com\nsub3.example.com\nsub4.example.com\nsub5.example.com"))
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, 200, 0, false)
	body, err := client.QuerySubdomains("example.com")

	if err != nil {
		t.Fatalf("QuerySubdomains failed: %v", err)
	}

	// Should only make 1 request since we got everything
	if requestCount != 1 {
		t.Errorf("expected 1 request (no pagination needed), got %d", requestCount)
	}

	if !strings.Contains(body, "sub5.example.com") {
		t.Errorf("expected response to contain all 5 results")
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
		w.Write([]byte(";;Entries: 1/1\nok"))
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

func TestAPIClient_PaginationRateLimit(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			w.Write([]byte(";;Entries: 2/10\nsub1\nsub2"))
		} else {
			w.Write([]byte(";;Entries: 10/10\nsub1\nsub2\nsub3\nsub4\nsub5\nsub6\nsub7\nsub8\nsub9\nsub10"))
		}
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, 200, 0.1, false)

	start := time.Now()
	client.QuerySubdomains("example.com")
	elapsed := time.Since(start)

	// Should wait between pagination requests
	if elapsed < 100*time.Millisecond {
		t.Errorf("pagination should respect rate limit: took %v, expected >= 100ms", elapsed)
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

Expected: FAIL with "undefined: NewAPIClient" or failures due to missing pagination

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
	BaseURL     string
	Limit       int
	RateLimit   float64
	HTTPClient  *http.Client
	Verbose     bool
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
	endpoint := fmt.Sprintf("/%s", ip)
	return c.queryWithPagination(endpoint)
}

// QuerySubdomains performs subdomain enumeration for a domain
func (c *APIClient) QuerySubdomains(domain string) (string, error) {
	endpoint := fmt.Sprintf("/sb/%s", domain)
	return c.queryWithPagination(endpoint)
}

// QueryCNAME performs CNAME lookup for a domain
func (c *APIClient) QueryCNAME(domain string) (string, error) {
	endpoint := fmt.Sprintf("/cn/%s", domain)
	return c.queryWithPagination(endpoint)
}

// queryWithPagination handles automatic pagination
func (c *APIClient) queryWithPagination(endpoint string) (string, error) {
	// Make initial request
	url := fmt.Sprintf("%s%s", c.BaseURL, endpoint)
	if c.Limit > 0 {
		url = fmt.Sprintf("%s?l=%d", url, c.Limit)
	}

	body, err := c.makeRequest(url)
	if err != nil {
		return "", err
	}

	// Parse response to check for pagination
	parser := NewResponseParser(false) // Don't print comments during internal parsing
	result := parser.Parse(body)

	// If we have all results, return
	if !result.HasMore() {
		return body, nil
	}

	// If user specified a limit, respect it and don't auto-paginate beyond it
	if c.Limit > 0 && result.TotalCount > c.Limit {
		// User wants limited results, respect their choice
		return body, nil
	}

	// More results available - fetch with full limit
	if c.Verbose {
		fmt.Fprintf(os.Stderr, "Auto-pagination: fetching all %d results...\n", result.TotalCount)
	}

	// Make second request with full count
	baseEndpoint := endpoint
	fullURL := fmt.Sprintf("%s%s?l=%d", c.BaseURL, baseEndpoint, result.TotalCount)

	fullBody, err := c.makeRequest(fullURL)
	if err != nil {
		return body, err // Return partial results if pagination fails
	}

	return fullBody, nil
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

Expected: PASS (all tests pass, including pagination tests)

**Step 5: Commit**

Run:
```bash
git add client.go client_test.go
git commit -m "feat: add API client with smart auto-pagination"
```

---

## Updated Main Orchestrator Note

Task 6 (Main Orchestrator) requires minimal changes:
- The main.go now uses the updated parser which returns `ParseResult` instead of `[]string`
- Access results via `result.Data` instead of just `result`

The pagination logic is entirely handled by the API client, so main.go just needs to iterate over `parser.Parse(body).Data`.

---

## Summary of Changes

**Response Parser (`parser.go`):**
- Returns `ParseResult` struct with Data, CurrentCount, TotalCount
- Extracts pagination metadata from `;;Entries: X/Y`
- Provides `HasMore()` method to check for additional results

**API Client (`client.go`):**
- New `queryWithPagination()` internal method
- Automatically detects partial results
- Makes second request with full limit if needed
- Respects user's `-l` flag preference
- Applies rate limiting between pagination requests

**Benefits:**
- Users get complete results automatically
- No need to guess the right `-l` value
- Respects rate limiting
- Still allows manual limit control with `-l` flag
