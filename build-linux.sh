#!/usr/bin/env bash
set -euo pipefail

echo "Building Sub2API Linux amd64 production binary..."

VERSION_FILE="backend/cmd/server/VERSION"
CONFIG_EXAMPLE="deploy/config.example.yaml"
UPLOAD_DIR="dist/upload"
BINARY_NAME="sub2api-linux-amd64"

if [ ! -f "$VERSION_FILE" ]; then
  echo "Missing version file: $VERSION_FILE" >&2
  exit 1
fi

if [ ! -f "$CONFIG_EXAMPLE" ]; then
  echo "Missing config example: $CONFIG_EXAMPLE" >&2
  exit 1
fi

VERSION=$(tr -d '\r\n' < "$VERSION_FILE")
COMMIT=$(git rev-parse --short HEAD)
DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
BUILD_TYPE="production"
PACKAGE_BASENAME="sub2api-${VERSION}-linux-amd64"

echo "Version: $VERSION"
echo "Commit: $COMMIT"
echo "Date: $DATE"
echo "BuildType: $BUILD_TYPE"

echo "Cleaning old upload artifacts..."
rm -rf "$UPLOAD_DIR"
mkdir -p "$UPLOAD_DIR"

echo "Building frontend into backend/internal/web/dist..."
if command -v pnpm >/dev/null 2>&1; then
  pnpm --dir frontend run build
else
  corepack pnpm --dir frontend run build
fi

echo "Building backend for linux/amd64..."
(
  cd backend
  GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build \
    -tags embed \
    -ldflags "-s -w -X 'main.Version=$VERSION' -X 'main.Commit=$COMMIT' -X 'main.Date=$DATE' -X 'main.BuildType=$BUILD_TYPE'" \
    -trimpath \
    -o "../$UPLOAD_DIR/$BINARY_NAME" \
    ./cmd/server
)

echo "Copying release metadata and example config..."
cp "$VERSION_FILE" "$UPLOAD_DIR/VERSION"
cp "$CONFIG_EXAMPLE" "$UPLOAD_DIR/config.example.yaml"

echo "Creating release package..."
(
  cd "$UPLOAD_DIR"
  if command -v zstd >/dev/null 2>&1; then
    PACKAGE="${PACKAGE_BASENAME}.tar.zst"
    tar -cf - "$BINARY_NAME" VERSION config.example.yaml | zstd -19 -o "$PACKAGE"
  else
    PACKAGE="${PACKAGE_BASENAME}.tar.gz"
    tar -czf "$PACKAGE" "$BINARY_NAME" VERSION config.example.yaml
  fi
  sha256sum "$BINARY_NAME" > "$BINARY_NAME.sha256"
  sha256sum "$PACKAGE" > "$PACKAGE.sha256"
)

echo "Build complete. Artifacts:"
ls -lh "$UPLOAD_DIR"
echo "SHA256:"
cat "$UPLOAD_DIR/$BINARY_NAME.sha256"
cat "$UPLOAD_DIR/${PACKAGE_BASENAME}".*.sha256
