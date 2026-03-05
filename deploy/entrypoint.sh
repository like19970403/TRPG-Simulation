#!/bin/sh
set -e

# Wait for DNS resolution to work (Docker embedded DNS can be slow)
echo "Waiting for postgres to be reachable..."
max_retries=15
retry=0
while [ $retry -lt $max_retries ]; do
  if goose -dir ./migrations postgres "$DATABASE_URL" status > /dev/null 2>&1; then
    break
  fi
  retry=$((retry + 1))
  echo "Waiting for database... (attempt $retry/$max_retries)"
  sleep 2
done

echo "Running database migrations..."
goose -dir ./migrations postgres "$DATABASE_URL" up

echo "Starting server..."
exec ./server
