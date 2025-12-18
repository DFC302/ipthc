# ipthc

Fast CLI tool for querying the ip.thc.org API.

## Installation

```bash
go install github.com/USERNAME/ipthc@latest
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

- `-dns`: DNS reverse lookup (IP → domains)
- `-subs`: Subdomain enumeration
- `-cname`: CNAME lookup (domains pointing to target)
- `-v`: Verbose mode (show API metadata and errors)
- `-l <int>`: Results limit per request (default: 200)
- `-r <float>`: Rate limit delay in seconds (default: 1.0)

## Examples

```bash
# Verbose output with custom limit
cat domains.txt | ipthc -subs -v -l 100

# Custom rate limiting
cat ips.txt | ipthc -dns -r 2.0

# Pipeline with other tools
cat domains.txt | ipthc -subs | sort | uniq
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
