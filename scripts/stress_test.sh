#!/bin/bash

# Default count is 1000 if not provided
COUNT=${1:-1000}

# Target URL
URL="http://localhost:9090/stress?count=${COUNT}"

echo "Starting stress test with ${COUNT} logs..."
echo "Requesting: ${URL}"

# Helper to get time in milliseconds (macOS/Linux compatible)
get_millis() {
    python3 -c 'import time; print(int(time.time() * 1000))'
}

# Send request
START_TIME=$(get_millis)
RESPONSE=$(curl -s "${URL}")
END_TIME=$(get_millis)

DURATION=$(( END_TIME - START_TIME ))

echo "Response: ${RESPONSE}"
echo "Total client-side time: ${DURATION}ms"
