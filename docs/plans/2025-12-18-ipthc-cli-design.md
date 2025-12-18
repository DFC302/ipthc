# ipthc CLI Tool Design

**Date:** 2025-12-18
**Status:** Approved

## Overview

A fast Go CLI tool for querying the ip.thc.org API to perform DNS reverse lookups, subdomain enumeration, and CNAME lookups. Reads input from stdin and outputs results to stdout.

## API Endpoints

- **DNS (Reverse Lookup)**: `https://ip.thc.org/{IP}?l={limit}`
- **Subdomains**: `https://ip.thc.org/sb/{domain}?l={limit}`
- **CNAME**: `https://ip.thc.org/cn/{domain}?l={limit}`

## CLI Interface

### Command Structure
```bash
ipthc -dns [-v] [-l 200] [-r 1.0] < input.txt
ipthc -subs [-v] [-l 200] [-r 1.0] < input.txt
ipthc -cname [-v] [-l 200] [-r 1.0] < input.txt
```

### Flags

- **Mode flags** (mutually exclusive, one required):
  - `-dns`: Reverse DNS lookup (IP → domains)
  - `-subs`: Subdomain enumeration
  - `-cname`: CNAME lookup (domains pointing to target)

- **Optional flags**:
  - `-v`: Verbose mode (show API comments and errors to stderr)
  - `-l <int>`: Results limit per API request (default: 200)
  - `-r <float>`: Rate limit delay in seconds between requests (default: 1.0)

### Usage Examples
```bash
# Single domain
echo "example.com" | ipthc -subs

# Multiple domains from file
cat domains.txt | ipthc -subs

# With custom limit and rate
cat ips.txt | ipthc -dns -l 100 -r 2.0

# Verbose mode
cat domains.txt | ipthc -cname -v
```

## Architecture

```
stdin reader → input validator → API client → response parser → stdout
                     ↓                              ↓
              error logger ← ← ← ← ← ← ← ← ← ← ← ← ←
                     ↓
            ipthc-errors.log
```

### Components

1. **Main Orchestrator** (`main.go`)
   - Parse CLI flags
   - Read lines from stdin
   - Coordinate validation, API calls, parsing
   - Track failure count for exit code

2. **Input Validator** (`validator.go`)
   - DNS mode: Validate IPv4/IPv6 using `net.ParseIP()`
   - Subs/CNAME modes: Basic domain validation
   - Skip empty lines and `#` comments
   - Trim whitespace

3. **API Client** (`client.go`)
   - Make HTTP GET requests with 30s timeout
   - Apply rate limiting (sleep between requests)
   - Handle HTTP errors gracefully
   - Return raw response body

4. **Response Parser** (`parser.go`)
   - Split response by lines
   - Identify comment lines (start with `;`)
   - In verbose mode: print comments to stderr
   - Output data lines to stdout

5. **Error Logger** (`logger.go`)
   - Log to `ipthc-errors.log` in current directory
   - Format: `[timestamp] [mode] [input] error_message`
   - Continue processing after errors

## Input Validation

### DNS Mode
- Accept only valid IPv4/IPv6 addresses
- Use `net.ParseIP()` for validation
- Example valid: `1.1.1.1`, `2606:4700:4700::1111`

### Subs/CNAME Modes
- Accept domain names
- Basic validation: contains `.`, valid characters
- Example valid: `example.com`, `sub.example.co.uk`

### Common Processing
- Skip empty lines
- Skip lines starting with `#` (comments)
- Trim whitespace from all inputs
- Invalid inputs: log to error file, continue

## Response Parsing

### API Response Format
```
;;Subdomains For: segfault.net
;;Entries: 12/12
;;Rate Limit: You can make 249 requests
segfault.net
adm.segfault.net
lookup.segfault.net
```

### Parsing Logic
- Lines starting with `;` = comments (metadata)
- Other non-empty lines = data (results)
- **Normal mode**: Output only data lines to stdout
- **Verbose mode**:
  - Output comment lines to stderr
  - Output data lines to stdout
  - Output errors to stderr

## Error Handling

### Error Categories

1. **Usage Errors** (exit immediately, code 1)
   - No mode flag or multiple mode flags
   - Invalid flag values (negative limit/rate)
   - Print usage message to stderr

2. **Input Validation Errors** (log and continue)
   - Invalid IP/domain format
   - Log to error file
   - Increment failure counter

3. **API Errors** (log and continue)
   - HTTP timeouts, network failures
   - 4xx/5xx responses
   - Log to error file with context
   - Increment failure counter

### Exit Codes
- `0`: All inputs processed successfully
- `1`: One or more failures occurred (check error log)

### Error Log Format
```
2025-12-18 16:30:45 [dns] 1.1.1.1 connection timeout
2025-12-18 16:30:46 [subs] invalid..domain invalid domain format
2025-12-18 16:30:47 [cname] example.com HTTP 500: server error
```

## Rate Limiting

- **Default**: 1 second delay between requests
- **Configurable**: `-r` flag accepts float seconds
- **Implementation**: Simple sequential processing
  1. Make request
  2. Parse response
  3. Sleep for rate limit duration
  4. Process next input

Conservative approach ensures no API rate limit violations.

## Project Structure

```
ipthc/
├── main.go           # CLI parsing, stdin reading, orchestration
├── client.go         # API client implementation
├── parser.go         # Response parsing
├── validator.go      # Input validation
├── logger.go         # Error logging
├── go.mod
├── README.md
└── docs/
    └── plans/
        └── 2025-12-18-ipthc-cli-design.md
```

## Dependencies

- **Standard library only**:
  - `net/http` - HTTP client
  - `flag` - CLI flag parsing
  - `bufio` - stdin reading
  - `net` - IP validation
  - `strings` - string manipulation
  - `time` - rate limiting
  - `os` - file I/O, exit codes

No external dependencies required.

## Implementation Data Structures

```go
type APIClient struct {
    BaseURL     string
    Limit       int
    RateLimit   float64
    HTTPClient  *http.Client
    Verbose     bool
}

type ResponseParser struct {
    Verbose bool
}

type ErrorLogger struct {
    file *os.File
}
```

## Success Criteria

- Read domains/IPs from stdin (one per line)
- Support piping from `echo` and `cat`
- Apply correct API endpoint based on mode
- Parse responses and output clean results
- Handle errors gracefully, continue processing
- Log failures to file
- Exit with appropriate code
- Fast execution with Go
- No external dependencies
