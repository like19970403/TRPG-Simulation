#!/usr/bin/env bash
#
# TRPG Simulation — One-click development environment startup
# Usage: bash scripts/dev-start.sh  (or: make quickstart)
#
set -euo pipefail

# --yes flag: auto-confirm all prompts (port kill, etc.)
AUTO_YES=false
for arg in "$@"; do
  case "$arg" in
    --yes|-y) AUTO_YES=true ;;
  esac
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT_DIR"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

GO_PID=""
FRONTEND_PID=""

cleanup() {
  echo ""
  echo -e "${CYAN}Shutting down...${NC}"
  if [ -n "$FRONTEND_PID" ] && kill -0 "$FRONTEND_PID" 2>/dev/null; then
    kill "$FRONTEND_PID" 2>/dev/null || true
    wait "$FRONTEND_PID" 2>/dev/null || true
  fi
  if [ -n "$GO_PID" ] && kill -0 "$GO_PID" 2>/dev/null; then
    kill "$GO_PID" 2>/dev/null || true
    wait "$GO_PID" 2>/dev/null || true
  fi
  echo -e "${GREEN}All services stopped.${NC}"
  exit 0
}

trap cleanup SIGINT SIGTERM

ok()   { echo -e "  ${GREEN}\xE2\x9C\x93${NC} $1"; }
fail() { echo -e "  ${RED}\xE2\x9C\x97 $1${NC}"; exit 1; }
warn() { echo -e "  ${YELLOW}! $1${NC}"; }

# Kill process occupying a given port
free_port() {
  local port=$1
  local pids
  pids=$(lsof -ti :"$port" 2>/dev/null || true)
  if [ -n "$pids" ]; then
    local info
    info=$(lsof -i :"$port" -P -n 2>/dev/null | tail -n +2 | head -3)
    warn "Port $port is in use:"
    echo "$info" | while IFS= read -r line; do echo "    $line"; done
    if [ "$AUTO_YES" = true ]; then
      answer="y"
    else
      echo -ne "  ${YELLOW}Kill these processes to free port $port? [Y/n]${NC} "
      read -r answer
    fi
    if [[ -z "$answer" || "$answer" =~ ^[Yy] ]]; then
      echo "$pids" | xargs kill -9 2>/dev/null || true
      sleep 1
      ok "Port $port freed"
    else
      fail "Port $port is still in use — cannot continue"
    fi
  fi
}

# ─────────────────────────────────────────────
# 0. Source .env & free required ports
# ─────────────────────────────────────────────
if [ -f .env ]; then
  set -a
  # shellcheck disable=SC1091
  source .env
  set +a
fi
PG_HOST_PORT="${PG_HOST_PORT:-5432}"

REQUIRED_PORTS=("$PG_HOST_PORT" 8080 3000)

echo ""
echo -e "${BOLD}Checking required ports...${NC}"

PORTS_BUSY=false
for port in "${REQUIRED_PORTS[@]}"; do
  if lsof -ti :"$port" >/dev/null 2>&1; then
    PORTS_BUSY=true
    free_port "$port"
  else
    ok "Port $port is available"
  fi
done

# ─────────────────────────────────────────────
# 1. Prerequisites
# ─────────────────────────────────────────────
echo ""
echo -e "${BOLD}Checking prerequisites...${NC}"

command -v docker  >/dev/null 2>&1 || fail "docker not found — install Docker first"
ok "docker"

command -v go      >/dev/null 2>&1 || fail "go not found — install Go 1.22+ first"
ok "go ($(go version | awk '{print $3}'))"

command -v node    >/dev/null 2>&1 || fail "node not found — install Node.js 18+ first"
ok "node ($(node -v))"

command -v npm     >/dev/null 2>&1 || fail "npm not found"
ok "npm ($(npm -v))"

if command -v goose >/dev/null 2>&1; then
  ok "goose"
else
  warn "goose not found — installing via go install..."
  go install github.com/pressly/goose/v3/cmd/goose@latest
  export PATH="$PATH:$(go env GOPATH)/bin"
  command -v goose >/dev/null 2>&1 || fail "goose installation failed"
  ok "goose (installed)"
fi

# Detect compose command (V2 plugin vs standalone)
if docker compose version >/dev/null 2>&1; then
  COMPOSE="docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
  COMPOSE="docker-compose"
else
  fail "Neither 'docker compose' nor 'docker-compose' found"
fi

# ─────────────────────────────────────────────
# 2. PostgreSQL
# ─────────────────────────────────────────────
echo ""
echo -e "${BOLD}Starting PostgreSQL...${NC}"

$COMPOSE up postgres -d --force-recreate 2>&1 || fail "Failed to start PostgreSQL"

# Wait for PostgreSQL to accept connections on host port (up to 30s)
RETRIES=30
until pg_isready -h localhost -p "$PG_HOST_PORT" -U trpg -d trpg_simulation >/dev/null 2>&1 || \
      $COMPOSE exec postgres pg_isready -U trpg -d trpg_simulation >/dev/null 2>&1; do
  RETRIES=$((RETRIES - 1))
  if [ "$RETRIES" -le 0 ]; then
    fail "PostgreSQL did not become ready in time"
  fi
  sleep 1
done
ok "PostgreSQL is ready (localhost:$PG_HOST_PORT)"

# ─────────────────────────────────────────────
# 3. Environment
# ─────────────────────────────────────────────
echo ""
echo -e "${BOLD}Setting up environment...${NC}"

if [ ! -f .env ]; then
  cp .env.example .env
  # Override for local dev
  sed -i 's/^JWT_SECRET=.*/JWT_SECRET=dev-secret-change-in-production-min-32chars/' .env
  # If no reverse proxy (Caddy/Nginx) with HTTPS, use COOKIE_SECURE=false
  if [ "${HTTPS_ENABLED:-false}" = "true" ]; then
    ok ".env created from .env.example (COOKIE_SECURE=true, HTTPS mode)"
  else
    sed -i 's/^COOKIE_SECURE=.*/COOKIE_SECURE=false/' .env
    sed -i '/^#.*Set to false/d' .env
    ok ".env created from .env.example (COOKIE_SECURE=false)"
  fi
else
  ok ".env already exists (skipped)"
fi

# Source DATABASE_URL for goose (use PG_HOST_PORT from .env)
export DATABASE_URL="${DATABASE_URL:-postgres://trpg:trpg_secret@localhost:${PG_HOST_PORT}/trpg_simulation?sslmode=disable}"

# ─────────────────────────────────────────────
# 4. Database Migrations
# ─────────────────────────────────────────────
echo ""
echo -e "${BOLD}Running database migrations...${NC}"

# Retry migration up to 5 times (PG port mapping may take a moment)
RETRIES=5
until goose -dir migrations postgres "$DATABASE_URL" up 2>/dev/null; do
  RETRIES=$((RETRIES - 1))
  if [ "$RETRIES" -le 0 ]; then
    # Show error on final attempt
    goose -dir migrations postgres "$DATABASE_URL" up || fail "Migration failed"
  fi
  sleep 2
done
ok "Migrations applied"

# ─────────────────────────────────────────────
# 5. Go Backend
# ─────────────────────────────────────────────
echo ""
echo -e "${BOLD}Starting Go backend (port 8080)...${NC}"

go run ./cmd/server/ &
GO_PID=$!

# Wait for backend to be ready (up to 30s)
RETRIES=30
until curl -sf http://localhost:8080/api/health >/dev/null 2>&1; do
  RETRIES=$((RETRIES - 1))
  if [ "$RETRIES" -le 0 ]; then
    fail "Backend did not start in time"
  fi
  # Check if process is still alive
  if ! kill -0 "$GO_PID" 2>/dev/null; then
    fail "Backend process exited unexpectedly"
  fi
  sleep 1
done
ok "Backend is ready: http://localhost:8080/api/health"

# ─────────────────────────────────────────────
# 6. React Frontend
# ─────────────────────────────────────────────
echo ""
echo -e "${BOLD}Starting React frontend (port 3000)...${NC}"

cd web

if [ ! -d node_modules ]; then
  echo "  Installing npm dependencies..."
  npm install --silent
  ok "Dependencies installed"
else
  ok "node_modules exists (skipped install)"
fi

echo ""
echo -e "${BOLD}${GREEN}"
echo "  ======================================="
echo "  TRPG Simulation — Development Ready"
echo "  Frontend:  http://localhost:3000"
echo "  Backend:   http://localhost:8080"
echo "  Database:  localhost:5432"
if [ -n "${ALLOWED_ORIGINS:-}" ]; then
echo "  External:  $ALLOWED_ORIGINS"
fi
echo "  ======================================="
echo -e "${NC}"
echo -e "  Press ${BOLD}Ctrl+C${NC} to stop all services."
echo ""

npm run dev &
FRONTEND_PID=$!

# Wait for either process to exit
wait "$FRONTEND_PID" 2>/dev/null || true
cleanup
