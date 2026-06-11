#!/usr/bin/env bash
set -euo pipefail

REMOTE_HOST=${REMOTE_HOST:?Set REMOTE_HOST to the target server host}
REMOTE_PORT=${REMOTE_PORT:-22}
REMOTE_USER=${REMOTE_USER:-root}
REMOTE="${REMOTE_USER}@${REMOTE_HOST}"
REMOTE_UPLOAD_DIR=${REMOTE_UPLOAD_DIR:-/opt/sub2api-rollout}
REMOTE_SCRIPT_LOCAL=${REMOTE_SCRIPT_LOCAL:-deploy/remote-production-flow.sh}
REMOTE_SCRIPT_NAME=${REMOTE_SCRIPT_NAME:-remote-production-flow.sh}
UPLOAD_DIR=${UPLOAD_DIR:-dist/upload}
BINARY_NAME=${BINARY_NAME:-sub2api-linux-amd64}
PACKAGE=${PACKAGE:-}
TEST_SERVICE=${TEST_SERVICE:-sub2api-test.service}
PRODUCTION_SERVICE=${PRODUCTION_SERVICE:-sub2api.service}
TEST_BINARY_PATH=${TEST_BINARY_PATH:-/opt/sub2api-test/sub2api}
PRODUCTION_BINARY_PATH=${PRODUCTION_BINARY_PATH:-/opt/sub2api/sub2api}
TEST_CONFIG_PATH=${TEST_CONFIG_PATH:-/opt/sub2api-test/data/config.yaml}
PRODUCTION_CONFIG_PATH=${PRODUCTION_CONFIG_PATH:-/app/data/config.yaml}
TEST_PORT=${TEST_PORT:-18808}
PRODUCTION_PORT=${PRODUCTION_PORT:-8808}

quote() {
  printf "'%s'" "$(printf '%s' "$1" | sed "s/'/'\"'\"'/g")"
}

require_file() {
  local path="$1"

  [ -f "$path" ] || { echo "Missing file: $path" >&2; exit 1; }
}

resolve_package_path() {
  if [ -n "$PACKAGE" ]; then
    printf '%s\n' "${UPLOAD_DIR}/${PACKAGE}"
    return
  fi

  if [ -f "${UPLOAD_DIR}/${BINARY_NAME}.tar.zst" ]; then
    printf '%s\n' "${UPLOAD_DIR}/${BINARY_NAME}.tar.zst"
    return
  fi

  if [ -f "${UPLOAD_DIR}/${BINARY_NAME}.tar.gz" ]; then
    printf '%s\n' "${UPLOAD_DIR}/${BINARY_NAME}.tar.gz"
    return
  fi

  echo "Missing rollout package under ${UPLOAD_DIR}" >&2
  exit 1
}

PACKAGE_PATH="$(resolve_package_path)"
BINARY_PATH="${UPLOAD_DIR}/${BINARY_NAME}"
REMOTE_PACKAGE_PATH="${REMOTE_UPLOAD_DIR%/}/$(basename "$PACKAGE_PATH")"
REMOTE_SCRIPT_PATH="${REMOTE_UPLOAD_DIR%/}/${REMOTE_SCRIPT_NAME}"

require_file "$PACKAGE_PATH"
require_file "$BINARY_PATH"
require_file "$REMOTE_SCRIPT_LOCAL"

LOCAL_HASH=$(sha256sum "$BINARY_PATH" | awk '{print $1}')
PACKAGE_HASH=$(sha256sum "$PACKAGE_PATH" | awk '{print $1}')

echo "Deploying Sub2API candidate to isolated test instance"
echo "Package: $PACKAGE_PATH"
echo "Local binary SHA256: $LOCAL_HASH"
echo "Remote: $REMOTE port $REMOTE_PORT"

echo "Uploading package and remote flow script..."
ssh -p "$REMOTE_PORT" "$REMOTE" "mkdir -p $(quote "$REMOTE_UPLOAD_DIR")"
scp -P "$REMOTE_PORT" "$PACKAGE_PATH" "$REMOTE_SCRIPT_LOCAL" "$REMOTE:$REMOTE_UPLOAD_DIR/"

REMOTE_CMD="chmod +x $(quote "$REMOTE_SCRIPT_PATH") && env MODE='test' STAGE_DIR=$(quote "$REMOTE_UPLOAD_DIR") PACKAGE_PATH=$(quote "$REMOTE_PACKAGE_PATH") EXPECTED_BINARY_SHA256=$(quote "$LOCAL_HASH") EXPECTED_PACKAGE_SHA256=$(quote "$PACKAGE_HASH") BINARY_NAME=$(quote "$BINARY_NAME") TEST_SERVICE=$(quote "$TEST_SERVICE") PRODUCTION_SERVICE=$(quote "$PRODUCTION_SERVICE") TEST_BINARY_PATH=$(quote "$TEST_BINARY_PATH") PRODUCTION_BINARY_PATH=$(quote "$PRODUCTION_BINARY_PATH") TEST_CONFIG_PATH=$(quote "$TEST_CONFIG_PATH") PRODUCTION_CONFIG_PATH=$(quote "$PRODUCTION_CONFIG_PATH") TEST_PORT=$(quote "$TEST_PORT") PRODUCTION_PORT=$(quote "$PRODUCTION_PORT") bash $(quote "$REMOTE_SCRIPT_PATH") test"
ssh -p "$REMOTE_PORT" "$REMOTE" "$REMOTE_CMD"

echo "Test instance validation complete. Promote with:"
echo "  REMOTE_HOST=$REMOTE_HOST REMOTE_PORT=$REMOTE_PORT REMOTE_USER=$REMOTE_USER REMOTE_UPLOAD_DIR=$REMOTE_UPLOAD_DIR ./deploy-to-production.sh"
