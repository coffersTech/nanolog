#!/bin/bash

# NanoLog Live Data Generator
# Targets the ingest API at port 8088

URL="http://localhost:9090/api/ingest"
SERVICES=("auth-service" "order-service" "payment-api")
LEVELS=("INFO" "WARN" "ERROR")
MESSAGES=(
    "User authentication success"
    "Order #10293 processing started"
    "Payment gateway timed out"
    "Database connection pool saturated"
    "New user registered: user_992"
    "Invalid token received"
    "Rate limit exceeded for client_id: abc"
)

echo "Starting data generation to $URL..."
echo "Press [CTRL+C] to stop."

while true; do
  # Get Current Nanosecond Timestamp
  if [[ "$OSTYPE" == "darwin"* ]]; then
      TS=$(python3 -c 'import time; print(int(time.time() * 1000000000))')
  else
      TS=$(date +%s%N)
  fi

  # Random values
  LEVEL=${LEVELS[$RANDOM % ${#LEVELS[@]}]}
  SERVICE=${SERVICES[$RANDOM % ${#SERVICES[@]}]}
  MSG=${MESSAGES[$RANDOM % ${#MESSAGES[@]}]}
  
  # Generate a simulated JSON payload for the message
  INNER_PAYLOAD="{\\\"user_id\\\": $RANDOM, \\\"action\\\": \\\"login\\\", \\\"meta\\\": {\\\"ip\\\": \\\"192.168.1.$((RANDOM%255))\\\", \\\"browser\\\": \\\"Chrome\\\"}}"

  # Construct the full log entry with the JSON payload in the message
  # Note: Including 'host' field as well for completeness
  DATA="[{\"timestamp\": $TS, \"level\": \"$LEVEL\", \"service\": \"$SERVICE\", \"host\": \"gen-script.local\", \"message\": \"API Request Body: $INNER_PAYLOAD\"}]"

  curl -s -X POST -H "Content-Type: application/json" -d "$DATA" "$URL" > /dev/null

  # Status output
  echo "[$(date +%H:%M:%S)] Sent Log: $LEVEL | $SERVICE | $MSG"

  sleep 0.5
done
