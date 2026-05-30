#!/usr/bin/env bash
set -euo pipefail

REMOTE_HOST=${REMOTE_HOST:?Set REMOTE_HOST to the target server host}
REMOTE_PORT=${REMOTE_PORT:-22}
REMOTE_USER=${REMOTE_USER:-root}
REMOTE="${REMOTE_USER}@${REMOTE_HOST}"

read -r -p "Confirm isolated test instance on 18808 passed validation (yes/no): " confirm
if [ "$confirm" != "yes" ]; then
  echo "Deployment cancelled"
  exit 1
fi

echo "Promoting verified test binary to production on port 8808..."
ssh -p "$REMOTE_PORT" "$REMOTE" <<'ENDSSH'
set -euo pipefail
TEST_HASH=$(sha256sum /opt/sub2api-test/sub2api | awk '{print $1}')
echo "Verified test binary SHA256: $TEST_HASH"
/opt/sub2api-test/sub2api -version
systemctl stop sub2api.service
if [ -f /opt/sub2api/sub2api ]; then
  cp /opt/sub2api/sub2api /opt/sub2api/sub2api.backup.$(date +%Y%m%d_%H%M%S)
fi
cp /opt/sub2api-test/sub2api /opt/sub2api/sub2api
chmod +x /opt/sub2api/sub2api
PROD_HASH=$(sha256sum /opt/sub2api/sub2api | awk '{print $1}')
echo "Production binary SHA256: $PROD_HASH"
test "$PROD_HASH" = "$TEST_HASH"
systemctl start sub2api.service
sleep 3
systemctl is-active sub2api.service
ENDSSH

echo "Verifying production instance..."
ssh -p "$REMOTE_PORT" "$REMOTE" <<'ENDSSH'
set -euo pipefail
systemctl status sub2api.service --no-pager | head -20
/opt/sub2api/sub2api -version
curl -fsS http://127.0.0.1:8808/health
curl -fsS http://127.0.0.1:8808/setup/status | grep -q '"needs_setup":false'
curl -fsS http://127.0.0.1:8808/api/v1/public/pricing >/dev/null
curl -fsS http://127.0.0.1:8808/api/v1/monitoring/summary >/dev/null
journalctl -u sub2api.service -n 50 --no-pager
ENDSSH

echo "Stopping temporary test service..."
ssh -p "$REMOTE_PORT" "$REMOTE" <<'ENDSSH'
set -euo pipefail
systemctl stop sub2api-test.service
systemctl disable sub2api-test.service
ENDSSH

echo "Production deployment complete."
