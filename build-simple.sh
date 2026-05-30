#!/usr/bin/env bash
set -euo pipefail

VERSION_FILE="backend/cmd/server/VERSION"
UPLOAD_DIR="dist/upload"
BINARY_NAME="sub2api-linux-amd64"

if [ ! -f "$VERSION_FILE" ]; then
  echo "Missing version file: $VERSION_FILE" >&2
  exit 1
fi

VERSION=$(tr -d '\r\n' < "$VERSION_FILE")
COMMIT=$(git rev-parse --short HEAD)
DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || date +"%Y-%m-%dT%H:%M:%SZ")
BUILD_TYPE="production"
PACKAGE_BASENAME="sub2api-${VERSION}-linux-amd64"

echo "Version: $VERSION | Commit: $COMMIT"

rm -rf "$UPLOAD_DIR"
mkdir -p "$UPLOAD_DIR"

echo "Building backend for linux/amd64 without embedded frontend refresh..."
(
  cd backend
  GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build \
    -tags embed \
    -ldflags "-s -w -X 'main.Version=$VERSION' -X 'main.Commit=$COMMIT' -X 'main.Date=$DATE' -X 'main.BuildType=$BUILD_TYPE'" \
    -trimpath \
    -o "../$UPLOAD_DIR/$BINARY_NAME" \
    ./cmd/server
)

cp "$VERSION_FILE" "$UPLOAD_DIR/VERSION"

(
  cd "$UPLOAD_DIR"
  if command -v zstd >/dev/null 2>&1; then
    PACKAGE="${PACKAGE_BASENAME}.tar.zst"
    tar -cf - "$BINARY_NAME" VERSION | zstd -19 -o "$PACKAGE"
  else
    PACKAGE="${PACKAGE_BASENAME}.tar.gz"
    tar -czf "$PACKAGE" "$BINARY_NAME" VERSION
  fi
  sha256sum "$BINARY_NAME" > "$BINARY_NAME.sha256"
  sha256sum "$PACKAGE" > "$PACKAGE.sha256"
)

echo "Build complete. Artifacts:"
ls -lh "$UPLOAD_DIR"
echo "SHA256:"
cat "$UPLOAD_DIR"/*.sha256
