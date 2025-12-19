package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
)

const (
	defaultBaseURL   = "https://ip.thc.org"
	defaultLimit     = 0 // 0 means no limit, auto-pagination will fetch all results
	defaultRateLimit = 1.0
	errorLogFile     = "ipthc-errors.log"
)

func main() {
	// Define flags
	dnsMode := flag.Bool("dns", false, "DNS reverse lookup mode")
	subsMode := flag.Bool("subs", false, "Subdomain enumeration mode")
	cnameMode := flag.Bool("cname", false, "CNAME lookup mode")
	verbose := flag.Bool("v", false, "Verbose mode (show API metadata and errors)")
	limit := flag.Int("l", defaultLimit, "Results limit per request (0 for auto-pagination to fetch all)")
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

	if *limit < 0 {
		fmt.Fprintln(os.Stderr, "Error: limit cannot be negative")
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
		result := parser.Parse(body)
		for _, data := range result.Data {
			fmt.Println(data)
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
