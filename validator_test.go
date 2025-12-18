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
