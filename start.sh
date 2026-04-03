#!/usr/bin/env bash
set -e

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "Starting backend: http://localhost:8000"
(
  cd "$PROJECT_ROOT/backend"
  go run .
) &
BACKEND_PID=$!

echo "Starting frontend: http://localhost:3000"
(
  cd "$PROJECT_ROOT/frontend"
  npm start
) &
FRONTEND_PID=$!

cleanup() {
  echo "Stopping services..."
  kill "$BACKEND_PID" "$FRONTEND_PID" 2>/dev/null || true
}

trap cleanup EXIT INT TERM
wait
