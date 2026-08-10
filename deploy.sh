#!/usr/bin/env bash
# Usage: ./deploy.sh
set -e

PROJECT_ID="billcalculator-fb2dd"
REGION="europe-west1"
SERVICE="cost-calculator-backend"
ROOT="$(cd "$(dirname "$0")" && pwd)"
KEY="$ROOT/backend/serviceAccountKey.json"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
ok()   { echo -e "${GREEN}✓${NC} $1"; }
warn() { echo -e "${YELLOW}!${NC} $1"; }
die()  { echo -e "${RED}✗ $1${NC}"; exit 1; }
step() { echo -e "\n${YELLOW}──── $1 ────${NC}"; }

# ── Pre-flight checks ──────────────────────────────────────────────────────────
step "Pre-flight checks"
command -v firebase >/dev/null || die "Firebase CLI not found. Run: npm install -g firebase-tools"
command -v gcloud   >/dev/null || die "gcloud CLI not found. Install: https://cloud.google.com/sdk/docs/install"
command -v npm      >/dev/null || die "npm not found"
[ -f "$KEY" ] || die "serviceAccountKey.json not found at $KEY"

GCLOUD_ACCOUNT=$(gcloud config get-value account 2>/dev/null)
[ -n "$GCLOUD_ACCOUNT" ] || die "Not logged in to gcloud. Run: gcloud auth login"
ok "Firebase CLI found"
ok "gcloud account: $GCLOUD_ACCOUNT"
gcloud config set project "$PROJECT_ID" --quiet

# ── Enable Cloud APIs ──────────────────────────────────────────────────────────
step "Enabling Cloud APIs (requires billing on the project)"
gcloud services enable \
  run.googleapis.com \
  secretmanager.googleapis.com \
  cloudbuild.googleapis.com \
  artifactregistry.googleapis.com \
  --project "$PROJECT_ID" --quiet
ok "APIs enabled"

# ── IAM permissions ────────────────────────────────────────────────────────────
step "Granting Cloud Build permissions to default service account"
PROJECT_NUMBER=$(gcloud projects describe "$PROJECT_ID" --format="value(projectNumber)")
SA="${PROJECT_NUMBER}-compute@developer.gserviceaccount.com"
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:$SA" --role="roles/cloudbuild.builds.builder" \
  --condition=None --quiet
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:$SA" --role="roles/storage.objectAdmin" \
  --condition=None --quiet
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:$SA" --role="roles/secretmanager.secretAccessor" \
  --condition=None --quiet
ok "Permissions granted to $SA"

# ── Secrets ────────────────────────────────────────────────────────────────────
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

# ── Deploy backend ─────────────────────────────────────────────────────────────
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

# ── Build & deploy frontend ────────────────────────────────────────────────────
step "Building frontend"
(cd "$ROOT/frontend" && npm ci --prefer-offline && npm run build)

step "Deploying to Firebase Hosting"
ORIG_FIREBASE="$ROOT/firebase.json"
TMP_FIREBASE="$ROOT/firebase.cloudrun.json"
python3 - <<PYEOF
import json
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

echo -e "\n${GREEN}═══ Deploy complete! ═══${NC}"
echo -e "  App     : https://${PROJECT_ID}.web.app"
echo -e "  Backend : $BACKEND_URL"
echo -e "${GREEN}════════════════════════${NC}"
