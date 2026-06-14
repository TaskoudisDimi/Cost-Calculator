#!/usr/bin/env bash
set -e

cleanup() {
  echo ""
  echo "Stopping..."
  kill $BACKEND_PID $FRONTEND_PID 2>/dev/null
  wait $BACKEND_PID $FRONTEND_PID 2>/dev/null
}
trap cleanup EXIT INT TERM

ROOT="$(cd "$(dirname "$0")" && pwd)"
KEY="$ROOT/backend/serviceAccountKey.json"

if [ ! -f "$KEY" ]; then
  echo "ERROR: $KEY not found."
  echo "Download it from Firebase Console → Project Settings → Service accounts → Generate new private key"
  exit 1
fi

echo "Starting backend..."
(cd "$ROOT/backend" && GOOGLE_APPLICATION_CREDENTIALS="$KEY" go run ./cmd/server) &
BACKEND_PID=$!

echo "Starting frontend..."
(cd "$ROOT/frontend" && npm run dev) &
FRONTEND_PID=$!

echo ""
echo "Backend:  http://localhost:8080"
echo "Frontend: http://localhost:5173"
echo ""
echo "Press Ctrl+C to stop."

wait $BACKEND_PID $FRONTEND_PID
