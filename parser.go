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
