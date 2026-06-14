#!/usr/bin/env bash
# Build the mobile app (iOS / Android) using the production Cloud Run backend
set -e

ROOT="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ID="billcalculator-fb2dd"
REGION="europe-west1"
SERVICE="cost-calculator-backend"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
ok()  { echo -e "${GREEN}✓${NC} $1"; }
die() { echo -e "${RED}✗ $1${NC}"; exit 1; }
step(){ echo -e "\n${YELLOW}──── $1 ────${NC}"; }

# ── Get backend URL ────────────────────────────────────────────────────────────
step "Fetching Cloud Run backend URL"
BACKEND_URL=$(gcloud run services describe "$SERVICE" \
  --region "$REGION" --project "$PROJECT_ID" \
  --format "value(status.url)" 2>/dev/null || echo "")

[ -n "$BACKEND_URL" ] || die "Could not find Cloud Run service '$SERVICE'. Run ./deploy.sh first."
ok "Backend: $BACKEND_URL"

# ── Build frontend with production API URL ─────────────────────────────────────
step "Building frontend"
(cd "$ROOT/frontend" && VITE_API_URL="$BACKEND_URL/api" npm run build)
ok "Frontend built"

# ── Sync Capacitor ─────────────────────────────────────────────────────────────
step "Syncing Capacitor"
(cd "$ROOT/frontend" && npx cap sync)
ok "Capacitor synced"

echo ""
echo -e "${GREEN}═══════════════════════════════════════${NC}"
echo -e "${GREEN}  Mobile build ready!${NC}"
echo ""
echo "  Open in Xcode (iOS):"
echo "    cd frontend && npx cap open ios"
echo ""
echo "  Open in Android Studio:"
echo "    cd frontend && npx cap open android"
echo ""
echo "  In Xcode/Android Studio, select your device and press Run."
echo -e "${GREEN}═══════════════════════════════════════${NC}"
