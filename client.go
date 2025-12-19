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

// PageCallback is called for each page of results
type PageCallback func(results []string, currentPage int, totalResults int) error

// QueryDNS performs a reverse DNS lookup for an IP address
func (c *APIClient) QueryDNS(ip string, callback PageCallback) error {
	endpoint := fmt.Sprintf("/%s", ip)
	return c.queryWithCallback(endpoint, callback)
}

// QuerySubdomains performs subdomain enumeration for a domain
func (c *APIClient) QuerySubdomains(domain string, callback PageCallback) error {
	endpoint := fmt.Sprintf("/sb/%s", domain)
	return c.queryWithCallback(endpoint, callback)
}

// QueryCNAME performs CNAME lookup for a domain
func (c *APIClient) QueryCNAME(domain string, callback PageCallback) error {
	endpoint := fmt.Sprintf("/cn/%s", domain)
	return c.queryWithCallback(endpoint, callback)
}

// queryWithCallback handles automatic pagination with streaming via callback
func (c *APIClient) queryWithCallback(endpoint string, callback PageCallback) error {
	// Make initial request
	url := fmt.Sprintf("%s%s", c.BaseURL, endpoint)
	if c.Limit > 0 {
		url = fmt.Sprintf("%s?l=%d", url, c.Limit)
	}

	body, err := c.makeRequest(url)
	if err != nil {
		return err
	}

	// Parse first page
	parser := NewResponseParser(c.Verbose)
	result := parser.Parse(body)

	// Call callback with first page
	if err := callback(result.Data, 1, result.TotalCount); err != nil {
		return err
	}

	// If user specified a limit, respect it and don't auto-paginate
	if c.Limit > 0 {
		return nil
	}

	// If there's no next page, we're done
	if !result.HasMore() {
		return nil
	}

	// Auto-pagination: follow next page links
	if c.Verbose {
		fmt.Fprintf(os.Stderr, "Auto-pagination: fetching all %d results...\n", result.TotalCount)
	}

	nextURL := result.NextPageURL
	pageCount := 1

	for nextURL != "" {
		pageCount++
		if c.Verbose {
			fmt.Fprintf(os.Stderr, "Fetching page %d...\n", pageCount)
		}

		pageBody, err := c.makeRequest(nextURL)
		if err != nil {
			// Return error if pagination fails
			if c.Verbose {
				fmt.Fprintf(os.Stderr, "Pagination failed: %v\n", err)
			}
			return err
		}

		pageResult := parser.Parse(pageBody)

		// Call callback with this page's data
		if err := callback(pageResult.Data, pageCount, result.TotalCount); err != nil {
			return err
		}

		nextURL = pageResult.NextPageURL

		// Safety check: limit to 100 pages max to prevent infinite loops
		if pageCount >= 100 {
			if c.Verbose {
				fmt.Fprintf(os.Stderr, "Reached maximum page limit (100)\n")
			}
			break
		}
	}

	return nil
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
