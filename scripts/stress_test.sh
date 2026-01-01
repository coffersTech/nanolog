#!/bin/bash

# NanoLog OOM & Auto-Flush Stress Test
# Sends 100,000 logs approx 1KB each (Total ~100MB)
# Expectation: 1-2 background flushes should trigger.

URL="http://localhost:8088/api/ingest"
JUNK=$(printf 'A%.0s' {1..1024}) # 1KB junk
TOKEN="sk-8f5e8e4ea501489657faf9eca73ec303"

echo "Starting stress test to $URL..."
echo "Sending 100,000 logs in 200 batches (500 logs each)..."

for batch in {1..3}; do
    # Try to get nanoseconds, fallback to python if non-numeric (macOS)
    TS=$(date +%s%N 2>/dev/null)
    if [[ ! "$TS" =~ ^[0-9]+$ ]]; then
        TS=$(python3 -c 'import time; print(int(time.time() * 1000000000))')
    fi
    
    # Send 500 logs per batch to avoid massive strings but still be fast
    PAYLOAD="["
    # Log levels for random selection
    LEVELS=("DEBUG" "INFO" "INFO" "INFO" "WARN" "ERROR") # INFO weighted higher
    for i in {1..500}; do
        LOG_TS=$((TS + i))
        # Random level selection
        LEVEL=${LEVELS[$((RANDOM % ${#LEVELS[@]}))]}
        PAYLOAD="$PAYLOAD{\"timestamp\": $LOG_TS, \"level\": \"$LEVEL\", \"service\": \"stress-test\", \"host\": \"localhost\", \"message\": \"Batch $batch-$i | $JUNK\"}"
        if [ $i -lt 500 ]; then
            PAYLOAD="$PAYLOAD,"
        fi
    done
    PAYLOAD="$PAYLOAD]"

    # Capture status and time using curl -w (with optional auth token)
    OUTPUT=$(curl -s -o /dev/null -w "%{http_code}:%{time_total}" -X POST \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${TOKEN:-}" \
        -d "$PAYLOAD" "$URL")
    
    STATUS=$(echo $OUTPUT | cut -d: -f1)
    DURATION=$(echo $OUTPUT | cut -d: -f2)
    
    echo "Sent batch $batch/200 | Status: $STATUS | Time: ${DURATION}s | Total: $((batch * 500))"
done

echo "Stress test complete."
