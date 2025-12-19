package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
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
