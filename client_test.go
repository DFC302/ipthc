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
