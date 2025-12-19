#!/bin/bash

# ipthc usage examples

echo "=== DNS Reverse Lookup ==="
echo "1.1.1.1" | ipthc -dns

echo -e "\n=== Subdomain Enumeration ==="
echo "example.com" | ipthc -subs -l 20

echo -e "\n=== CNAME Lookup ==="
echo "example.com" | ipthc -cname

echo -e "\n=== Batch Processing from File ==="
cat << DOMAINS > /tmp/test-domains.txt
example.com
google.com
github.com
DOMAINS

cat /tmp/test-domains.txt | ipthc -subs -l 10

echo -e "\n=== Verbose Mode ==="
echo "example.com" | ipthc -subs -v -l 5

echo -e "\n=== Pipeline with other tools ==="
echo "example.com" | ipthc -subs | head -5

echo -e "\n=== Auto-pagination (fetch all results) ==="
echo "abbvie.com" | ipthc -subs | wc -l

rm /tmp/test-domains.txt
