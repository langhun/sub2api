#!/usr/bin/env bash
set -euo pipefail

REMOTE_HOST=${REMOTE_HOST:?Set REMOTE_HOST to the target server host}
REMOTE_PORT=${REMOTE_PORT:-22}
REMOTE_USER=${REMOTE_USER:-root}
REMOTE="${REMOTE_USER}@${REMOTE_HOST}"
REMOTE_UPLOAD_DIR=${REMOTE_UPLOAD_DIR:-/opt/sub2api-rollout}
REMOTE_SCRIPT_LOCAL=${REMOTE_SCRIPT_LOCAL:-deploy/remote-production-flow.sh}
REMOTE_SCRIPT_NAME=${REMOTE_SCRIPT_NAME:-remote-production-flow.sh}
TEST_SERVICE=${TEST_SERVICE:-sub2api-test.service}
PRODUCTION_SERVICE=${PRODUCTION_SERVICE:-sub2api.service}
TEST_BINARY_PATH=${TEST_BINARY_PATH:-/opt/sub2api-test/sub2api}
PRODUCTION_BINARY_PATH=${PRODUCTION_BINARY_PATH:-/opt/sub2api/sub2api}
TEST_CONFIG_PATH=${TEST_CONFIG_PATH:-/opt/sub2api-test/data/config.yaml}
PRODUCTION_CONFIG_PATH=${PRODUCTION_CONFIG_PATH:-/app/data/config.yaml}
TEST_PORT=${TEST_PORT:-18808}
PRODUCTION_PORT=${PRODUCTION_PORT:-8808}
KEEP_TEST_SERVICE=${KEEP_TEST_SERVICE:-0}
AUTO_APPROVE=${AUTO_APPROVE:-0}

quote() {
  printf "'%s'" "$(printf '%s' "$1" | sed "s/'/'\"'\"'/g")"
}

require_file() {
  local path="$1"

  [ -f "$path" ] || { echo "Missing file: $path" >&2; exit 1; }
}

REMOTE_SCRIPT_PATH="${REMOTE_UPLOAD_DIR%/}/${REMOTE_SCRIPT_NAME}"

if [ "$AUTO_APPROVE" != "1" ]; then
  read -r -p "Confirm isolated test instance on 18808 passed validation (yes/no): " confirm
  if [ "$confirm" != "yes" ]; then
    echo "Deployment cancelled"
    exit 1
  fi
fi

require_file "$REMOTE_SCRIPT_LOCAL"

echo "Uploading remote flow script..."
ssh -p "$REMOTE_PORT" "$REMOTE" "mkdir -p $(quote "$REMOTE_UPLOAD_DIR")"
scp -P "$REMOTE_PORT" "$REMOTE_SCRIPT_LOCAL" "$REMOTE:$REMOTE_UPLOAD_DIR/"

echo "Promoting verified test binary to production on port 8808..."
REMOTE_CMD="chmod +x $(quote "$REMOTE_SCRIPT_PATH") && env MODE='promote' STAGE_DIR=$(quote "$REMOTE_UPLOAD_DIR") KEEP_TEST_SERVICE=$(quote "$KEEP_TEST_SERVICE") TEST_SERVICE=$(quote "$TEST_SERVICE") PRODUCTION_SERVICE=$(quote "$PRODUCTION_SERVICE") TEST_BINARY_PATH=$(quote "$TEST_BINARY_PATH") PRODUCTION_BINARY_PATH=$(quote "$PRODUCTION_BINARY_PATH") TEST_CONFIG_PATH=$(quote "$TEST_CONFIG_PATH") PRODUCTION_CONFIG_PATH=$(quote "$PRODUCTION_CONFIG_PATH") TEST_PORT=$(quote "$TEST_PORT") PRODUCTION_PORT=$(quote "$PRODUCTION_PORT") bash $(quote "$REMOTE_SCRIPT_PATH") promote"
ssh -p "$REMOTE_PORT" "$REMOTE" "$REMOTE_CMD"

echo "Production deployment complete."
