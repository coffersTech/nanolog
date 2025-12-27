#!/bin/bash

# Configuration
URL="http://localhost:8088/api/ingest"
CONTENT_TYPE="Content-Type: application/json"

echo "Starting Ingest Test to $URL..."

for i in {1..10}
do
   # Fix for macOS date which doesn't support %N
   TS=$(date +%s)000000000
   LEVEL=$((i % 4))
   LEVEL_STR="INFO"
   if [ $LEVEL -eq 2 ]; then LEVEL_STR="WARN"; fi
   if [ $LEVEL -eq 3 ]; then LEVEL_STR="ERROR"; fi
   
   PAYLOAD="{\"timestamp\":$TS, \"level\":\"$LEVEL_STR\", \"service\":\"test-svc-$i\", \"message\":\"Test log entry $i\"}"
   
   echo -n "Sending Log $i: "
   curl -s -o /dev/null -w "%{http_code}" -X POST "$URL" -H "$CONTENT_TYPE" -d "$PAYLOAD"
   echo ""
done

echo "Test Complete."
