#!/usr/bin/env bash
set -euo pipefail

echo "Building Sub2API Linux amd64 production binary..."

VERSION_FILE="backend/cmd/server/VERSION"
CONFIG_EXAMPLE="deploy/config.example.yaml"
UPLOAD_DIR="dist/upload"
BINARY_NAME="sub2api-linux-amd64"
PACKAGE_BASENAME="$BINARY_NAME"

require_file() {
  local path="$1"

  if [ ! -f "$path" ]; then
    echo "Missing file: $path" >&2
    exit 1
  fi
}

build_frontend() {
  if command -v pnpm >/dev/null 2>&1; then
    pnpm --dir frontend run build
    return
  fi

  if command -v corepack >/dev/null 2>&1; then
    corepack pnpm --dir frontend run build
    return
  fi

  if [ -x "frontend/node_modules/.bin/vite" ] || [ -f "frontend/node_modules/.bin/vite" ]; then
    (
      cd frontend
      ./node_modules/.bin/vite build
    )
    return
  fi

  echo "Missing pnpm/corepack and frontend/node_modules/.bin/vite was not found." >&2
  exit 1
}

require_file "$VERSION_FILE"
require_file "$CONFIG_EXAMPLE"

VERSION=$(tr -d '\r\n' < "$VERSION_FILE")
COMMIT=$(git rev-parse --short HEAD)
DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
BUILD_TYPE="production"

echo "Version: $VERSION"
echo "Commit: $COMMIT"
echo "Date: $DATE"
echo "BuildType: $BUILD_TYPE"

echo "Cleaning old upload artifacts..."
rm -rf "$UPLOAD_DIR"
mkdir -p "$UPLOAD_DIR"

echo "Building frontend into backend/internal/web/dist..."
build_frontend

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
cd - >/dev/null

echo "Build complete. Artifacts:"
ls -lh "$UPLOAD_DIR"
echo "SHA256:"
cat "$UPLOAD_DIR/$BINARY_NAME.sha256"
cat "$UPLOAD_DIR/$PACKAGE.sha256"
