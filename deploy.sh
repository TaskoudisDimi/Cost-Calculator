#!/usr/bin/env bash
# Usage:
#   ./deploy.sh cloudrun   — Firebase Hosting + Cloud Run (requires GCP billing)
#   ./deploy.sh railway    — Firebase Hosting + Railway backend (no GCP billing needed)
set -e

MODE="${1:-}"
PROJECT_ID="billcalculator-fb2dd"
REGION="europe-west1"
SERVICE="cost-calculator-backend"
ROOT="$(cd "$(dirname "$0")" && pwd)"
KEY="$ROOT/backend/serviceAccountKey.json"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'
ok()   { echo -e "${GREEN}✓${NC} $1"; }
warn() { echo -e "${YELLOW}!${NC} $1"; }
die()  { echo -e "${RED}✗ $1${NC}"; exit 1; }
step() { echo -e "\n${YELLOW}──── $1 ────${NC}"; }

if [ -z "$MODE" ]; then
  echo -e "${CYAN}Choose deployment backend:${NC}"
  echo "  1) Cloud Run  (GCP billing required, same Google account)"
  echo "  2) Railway    (no GCP billing, free tier)"
  read -rp "Enter 1 or 2: " choice
  case "$choice" in
    1) MODE="cloudrun" ;;
    2) MODE="railway" ;;
    *) die "Invalid choice" ;;
  esac
fi

# ── Shared pre-flight ──────────────────────────────────────────────────────────
step "Pre-flight checks"
command -v firebase >/dev/null || die "Firebase CLI not found. Run: npm install -g firebase-tools"
command -v npm      >/dev/null || die "npm not found"
[ -f "$KEY" ] || die "serviceAccountKey.json not found at $KEY"
ok "Firebase CLI found"

# ═══════════════════════════════════════════════════════════════════════════════
# PATH A: Cloud Run
# ═══════════════════════════════════════════════════════════════════════════════
if [ "$MODE" = "cloudrun" ]; then

  command -v gcloud >/dev/null || die "gcloud CLI not found. Install: https://cloud.google.com/sdk/docs/install"
  GCLOUD_ACCOUNT=$(gcloud config get-value account 2>/dev/null)
  [ -n "$GCLOUD_ACCOUNT" ] || die "Not logged in to gcloud. Run: gcloud auth login"
  ok "gcloud account: $GCLOUD_ACCOUNT"
  gcloud config set project "$PROJECT_ID" --quiet

  step "Enabling Cloud APIs (requires billing on the project)"
  gcloud services enable \
    run.googleapis.com \
    secretmanager.googleapis.com \
    cloudbuild.googleapis.com \
    artifactregistry.googleapis.com \
    --project "$PROJECT_ID" --quiet
  ok "APIs enabled"

  step "Storing service account key in Secret Manager"
  if gcloud secrets describe firebase-service-account --project "$PROJECT_ID" &>/dev/null; then
    warn "Secret exists — updating version..."
    gcloud secrets versions add firebase-service-account --data-file="$KEY" --project "$PROJECT_ID"
  else
    gcloud secrets create firebase-service-account --data-file="$KEY" --project "$PROJECT_ID"
    ok "Secret created"
  fi

  if [ -n "$ANTHROPIC_API_KEY" ]; then
    if gcloud secrets describe anthropic-api-key --project "$PROJECT_ID" &>/dev/null; then
      echo -n "$ANTHROPIC_API_KEY" | gcloud secrets versions add anthropic-api-key --data-file=- --project "$PROJECT_ID"
    else
      echo -n "$ANTHROPIC_API_KEY" | gcloud secrets create anthropic-api-key --data-file=- --project "$PROJECT_ID"
    fi
    ok "Anthropic key stored"
    ANTHROPIC_FLAG="--update-secrets=ANTHROPIC_API_KEY=anthropic-api-key:latest"
  else
    warn "ANTHROPIC_API_KEY not set — bill scanning disabled"
    ANTHROPIC_FLAG=""
  fi

  step "Deploying backend to Cloud Run"
  gcloud run deploy "$SERVICE" \
    --source "$ROOT/backend/" \
    --project "$PROJECT_ID" \
    --region "$REGION" \
    --allow-unauthenticated \
    --port 8080 \
    --memory 256Mi \
    --set-env-vars "FIREBASE_PROJECT_ID=$PROJECT_ID,GIN_MODE=release" \
    --update-secrets "/secrets/sa.json=firebase-service-account:latest" \
    --set-env-vars "GOOGLE_APPLICATION_CREDENTIALS=/secrets/sa.json" \
    $ANTHROPIC_FLAG \
    --quiet
  ok "Backend deployed"

  BACKEND_URL=$(gcloud run services describe "$SERVICE" \
    --region "$REGION" --project "$PROJECT_ID" \
    --format "value(status.url)")
  ok "Backend URL: $BACKEND_URL"

  step "Building frontend"
  (cd "$ROOT/frontend" && npm ci --prefer-offline && npm run build)

  step "Deploying to Firebase Hosting (with Cloud Run rewrite)"
  # Temporarily patch firebase.json to add the /api/** → Cloud Run rewrite
  ORIG_FIREBASE="$ROOT/firebase.json"
  TMP_FIREBASE="$ROOT/firebase.cloudrun.json"
  python3 - <<PYEOF
import json, sys
with open('$ORIG_FIREBASE') as f:
    cfg = json.load(f)
cfg['hosting']['rewrites'] = [
    {"source": "/api/**", "run": {"serviceId": "$SERVICE", "region": "$REGION"}},
    {"source": "**", "destination": "/index.html"}
]
with open('$TMP_FIREBASE', 'w') as f:
    json.dump(cfg, f, indent=2)
PYEOF
  firebase deploy --only hosting --config "$TMP_FIREBASE" --project "$PROJECT_ID"
  rm -f "$TMP_FIREBASE"

  echo -e "\n${GREEN}═══ Deploy complete! (Cloud Run) ═══${NC}"
  echo -e "  App      : https://${PROJECT_ID}.web.app"
  echo -e "  Backend  : $BACKEND_URL"
  echo -e "${GREEN}════════════════════════════════════${NC}"

# ═══════════════════════════════════════════════════════════════════════════════
# PATH B: Railway
# ═══════════════════════════════════════════════════════════════════════════════
elif [ "$MODE" = "railway" ]; then

  echo -e "\n${CYAN}Railway deployment — manual steps:${NC}"
  echo ""
  echo "  1. Go to https://railway.app and sign in with GitHub"
  echo "  2. New Project → Deploy from GitHub repo"
  echo "     Root directory:  backend/"
  echo "     (Railway auto-detects the Dockerfile)"
  echo ""
  echo "  3. Set these environment variables in the Railway dashboard:"
  echo -e "     ${YELLOW}FIREBASE_PROJECT_ID${NC}           = $PROJECT_ID"
  echo -e "     ${YELLOW}GIN_MODE${NC}                      = release"
  echo -e "     ${YELLOW}GOOGLE_APPLICATION_CREDENTIALS${NC} = /etc/secrets/sa.json  (after step 4)"
  if [ -n "$ANTHROPIC_API_KEY" ]; then
  echo -e "     ${YELLOW}ANTHROPIC_API_KEY${NC}              = $ANTHROPIC_API_KEY"
  fi
  echo ""
  echo "  4. Add the service account key as a Railway Secret File:"
  echo "     Path: /etc/secrets/sa.json"
  echo "     Content: (paste the contents of backend/serviceAccountKey.json)"
  echo ""
  echo "  5. After Railway deploys, copy the public URL (e.g. https://xxx.up.railway.app)"
  echo "     and run this script again:"
  echo ""
  echo -e "     ${GREEN}RAILWAY_URL=https://xxx.up.railway.app ./deploy.sh railway-frontend${NC}"
  echo ""

elif [ "$MODE" = "railway-frontend" ]; then

  [ -n "$RAILWAY_URL" ] || die "Set RAILWAY_URL=https://xxx.up.railway.app before running"

  FIREBASE_HOSTING_URL="https://${PROJECT_ID}.web.app"
  API_URL="$RAILWAY_URL/api"

  step "Building frontend (pointing to Railway backend)"
  (cd "$ROOT/frontend" && npm ci --prefer-offline && VITE_API_URL="$API_URL" npm run build)

  step "Deploying frontend to Firebase Hosting"
  firebase deploy --only hosting --project "$PROJECT_ID"

  step "Setting ALLOWED_ORIGINS on Railway backend"
  echo ""
  warn "One more step: add this env var in Railway dashboard:"
  echo -e "  ${YELLOW}ALLOWED_ORIGINS${NC} = $FIREBASE_HOSTING_URL"
  echo ""
  echo -e "${GREEN}═══ Deploy complete! (Railway) ════${NC}"
  echo -e "  App     : $FIREBASE_HOSTING_URL"
  echo -e "  Backend : $RAILWAY_URL"
  echo -e "${GREEN}══════════════════════════════════${NC}"

else
  die "Unknown mode '$MODE'. Use: cloudrun | railway | railway-frontend"
fi
