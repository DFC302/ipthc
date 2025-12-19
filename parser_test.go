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
