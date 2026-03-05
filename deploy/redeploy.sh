#!/bin/sh
set -e

cd "$(dirname "$0")/.."

echo "==> Rebuilding Docker images..."
docker compose build --no-cache

echo "==> Restarting containers..."
docker compose up -d --force-recreate

echo "==> Waiting for app to be healthy..."
# Extract port from APP_HOST_PORT (format: [host:]port), default 3000
HEALTH_PORT="${APP_HOST_PORT:-0.0.0.0:3000}"
HEALTH_PORT="${HEALTH_PORT##*:}"
retries=0
max_retries=15
while [ $retries -lt $max_retries ]; do
  if curl -sf "http://localhost:${HEALTH_PORT}/api/health" > /dev/null 2>&1; then
    echo "==> App is up!"
    docker compose ps
    exit 0
  fi
  retries=$((retries + 1))
  echo "    Waiting... (attempt $retries/$max_retries)"
  sleep 2
done

echo "==> WARNING: App did not become healthy in time"
docker compose logs --tail=20 app
exit 1
