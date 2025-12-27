#!/bin/bash

# NanoLog Live Data Generator
# Targets the ingest API at port 8088

URL="http://localhost:8088/api/ingest"
SERVICES=("auth-service" "order-service" "payment-api")
LEVELS=("1" "2" "3")
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
  # macOS date command might not support %N, using a fallback
  if [[ "$OSTYPE" == "darwin"* ]]; then
      TS=$(python3 -c 'import time; print(int(time.time() * 1000000000))')
  else
      TS=$(date +%s%N)
  fi

  # Random values
  LEVEL=${LEVELS[$RANDOM % ${#LEVELS[@]}]}
  SERVICE=${SERVICES[$RANDOM % ${#SERVICES[@]}]}
  MSG=${MESSAGES[$RANDOM % ${#MESSAGES[@]}]}

  # Payload as JSON array
  # Note: server expects [JSON] array or single object depending on parser logic.
  # Current ingest server handleIngest uses s.parser.Get().ParseBytes(body)
  # then v.GetInt64("timestamp"), etc.
  # If the server is not optimized for arrays yet, we send a single object.
  # As of previous implementation, it parses the body directly as an object.
  
  PAYLOAD="{\"timestamp\": $TS, \"level\": \"$LEVEL\", \"service\": \"$SERVICE\", \"message\": \"$MSG\"}"

  curl -s -X POST -H "Content-Type: application/json" -d "$PAYLOAD" "$URL" > /dev/null

  # Status output
  echo "[$(date +%H:%M:%S)] Sent Log: $LEVEL | $SERVICE | $MSG"

  sleep 0.5
done
