# ipthc

Fast CLI tool for querying the ip.thc.org API with **automatic pagination** support.

## Features

- **Smart Auto-Pagination**: Automatically fetches ALL results (no manual limit guessing!)
- **Clean Output**: ANSI color codes stripped for easy piping
- **Rate Limiting**: Respects API limits with configurable delays
- **Error Logging**: Failed queries logged to file for review
- **Streaming**: Process results as they arrive via stdin/stdout

## Installation

```bash
go install github.com/dfc302/ipthc@latest
```

Or build from source:
```bash
git clone https://github.com/dfc302/ipthc
cd ipthc
go build -o ipthc
```

## Usage

### DNS Reverse Lookup
```bash
echo "1.1.1.1" | ipthc -dns
cat ips.txt | ipthc -dns
```

### Subdomain Enumeration
```bash
echo "example.com" | ipthc -subs
cat domains.txt | ipthc -subs
```

### CNAME Lookup
```bash
echo "example.com" | ipthc -cname
cat domains.txt | ipthc -cname
```

## Flags

### Mode Flags (required, mutually exclusive)
- `-dns`: DNS reverse lookup (IP → domains)
- `-subs`: Subdomain enumeration
- `-cname`: CNAME lookup (domains pointing to target)

### Optional Flags
- `-v`: Verbose mode (show API metadata, pagination progress, and errors)
- `-l <int>`: Results limit (default: 0 = auto-fetch all results)
- `-r <float>`: Rate limit delay in seconds between requests (default: 1.0)

## Examples

### Auto-Pagination (Default)
```bash
# Automatically fetches ALL subdomains (e.g., 1041 results across 11 pages)
echo "abbvie.com" | ipthc -subs > subdomains.txt

# With verbose mode to see pagination progress
echo "abbvie.com" | ipthc -subs -v
```

### Manual Limit
```bash
# Limit to first 100 results only
cat domains.txt | ipthc -subs -l 100
```

### Custom Rate Limiting
```bash
# Slower requests (2 second delay)
cat ips.txt | ipthc -dns -r 2.0
```

### Pipeline with Other Tools
```bash
# Get unique subdomains, sorted
cat domains.txt | ipthc -subs | sort | uniq

# Count total subdomains
echo "example.com" | ipthc -subs | wc -l

# Filter specific patterns
echo "example.com" | ipthc -subs | grep "admin"
```

## Error Handling

Errors are logged to `ipthc-errors.log` in the current directory. Use `-v` flag to see errors in stderr during execution.

Exit codes:
- `0`: All queries succeeded
- `1`: One or more queries failed (check error log)

## API

Uses https://ip.thc.org/ API endpoints:
- DNS: `https://ip.thc.org/{IP}?l={limit}`
- Subdomains: `https://ip.thc.org/sb/{domain}?l={limit}`
- CNAME: `https://ip.thc.org/cn/{domain}?l={limit}`
