#!/bin/bash

# Configuration
URL="http://localhost:8088/api/ingest"
CONTENT_TYPE="Content-Type: application/json"
TOKEN="${1:-"sk-8f5e8e4ea501489657faf9eca73ec303"}" # Take token from first argument

if [ -z "$TOKEN" ]; then
    echo "Warning: No token provided. If the server requires authentication, this will fail with 401."
    echo "Usage: $0 <API_TOKEN>"
fi

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
   curl -s -o /dev/null -w "%{http_code}" -X POST "$URL" \
        -H "$CONTENT_TYPE" \
        -H "Authorization: Bearer $TOKEN" \
        -d "$PAYLOAD"
   echo ""
done

echo "Test Complete."
