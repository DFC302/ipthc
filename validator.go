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
