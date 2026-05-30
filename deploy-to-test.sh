#!/usr/bin/env bash
set -euo pipefail

VERSION=${VERSION:-$(tr -d '\r\n' < backend/cmd/server/VERSION)}
PACKAGE=${PACKAGE:-sub2api-${VERSION}-linux-amd64.tar.zst}
REMOTE_HOST=${REMOTE_HOST:?Set REMOTE_HOST to the target server host}
REMOTE_PORT=${REMOTE_PORT:-22}
REMOTE_USER=${REMOTE_USER:-root}
REMOTE="${REMOTE_USER}@${REMOTE_HOST}"
UPLOAD_DIR=${UPLOAD_DIR:-dist/upload}
PACKAGE_PATH="${UPLOAD_DIR}/${PACKAGE}"
BINARY_PATH="${UPLOAD_DIR}/sub2api-linux-amd64"

if [ ! -f "$PACKAGE_PATH" ]; then
  echo "Missing package: $PACKAGE_PATH" >&2
  exit 1
fi

if [ ! -f "$BINARY_PATH" ]; then
  echo "Missing binary: $BINARY_PATH" >&2
  exit 1
fi

LOCAL_HASH=$(sha256sum "$BINARY_PATH" | awk '{print $1}')

echo "Deploying Sub2API candidate to isolated test instance"
echo "Version: $VERSION"
echo "Package: $PACKAGE_PATH"
echo "Local binary SHA256: $LOCAL_HASH"
echo "Remote: $REMOTE port $REMOTE_PORT"

echo "Uploading package..."
scp -P "$REMOTE_PORT" "$PACKAGE_PATH" "$REMOTE:/tmp/"

echo "Extracting and verifying remote candidate..."
ssh -p "$REMOTE_PORT" "$REMOTE" "PACKAGE='$PACKAGE' LOCAL_HASH='$LOCAL_HASH' bash -s" <<'ENDSSH'
set -euo pipefail
cd /tmp
rm -f sub2api-linux-amd64 VERSION config.example.yaml
case "$PACKAGE" in
  *.tar.zst) tar --zstd -xf "$PACKAGE" ;;
  *.tar.gz|*.tgz) tar -xzf "$PACKAGE" ;;
  *) echo "Unsupported package format: $PACKAGE" >&2; exit 1 ;;
esac
chmod +x sub2api-linux-amd64
REMOTE_HASH=$(sha256sum sub2api-linux-amd64 | awk '{print $1}')
echo "Remote candidate SHA256: $REMOTE_HASH"
test "$REMOTE_HASH" = "$LOCAL_HASH"
./sub2api-linux-amd64 -version
ENDSSH

echo "Refreshing sub2api-test.service on port 18808..."
ssh -p "$REMOTE_PORT" "$REMOTE" <<'ENDSSH'
set -euo pipefail
systemctl stop sub2api-test.service || true
if [ -f /opt/sub2api-test/sub2api ]; then
  cp /opt/sub2api-test/sub2api /opt/sub2api-test/sub2api.backup
fi
cp /tmp/sub2api-linux-amd64 /opt/sub2api-test/sub2api
chmod +x /opt/sub2api-test/sub2api
systemctl start sub2api-test.service
sleep 3
systemctl is-active sub2api-test.service
ENDSSH

echo "Verifying test instance..."
ssh -p "$REMOTE_PORT" "$REMOTE" <<'ENDSSH'
set -euo pipefail
systemctl status sub2api-test.service --no-pager | head -20
TEST_HASH=$(sha256sum /opt/sub2api-test/sub2api | awk '{print $1}')
echo "Test binary SHA256: $TEST_HASH"
/opt/sub2api-test/sub2api -version
curl -fsS http://127.0.0.1:18808/health
curl -fsS http://127.0.0.1:18808/setup/status | grep -q '"needs_setup":false'
curl -fsS http://127.0.0.1:18808/api/v1/public/pricing >/dev/null
curl -fsS http://127.0.0.1:18808/api/v1/monitoring/summary >/dev/null
journalctl -u sub2api-test.service -n 50 --no-pager
ENDSSH

echo "Test instance validation complete. Promote with:"
echo "  REMOTE_HOST=$REMOTE_HOST REMOTE_PORT=$REMOTE_PORT REMOTE_USER=$REMOTE_USER ./deploy-to-production.sh"
