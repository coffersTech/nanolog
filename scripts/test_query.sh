#!/bin/bash

# Test Query Script for NanoLog
# Note: Server runs on port 8088

echo "=== NanoLog Query Test ==="
echo ""

URL="http://localhost:8088/api/search?limit=50"
echo "Querying: $URL"
echo ""

echo "=== Response ==="
curl -s "$URL" | python3 -m json.tool 2>/dev/null || curl -s "$URL"
echo ""

echo "=== Test Complete ==="
