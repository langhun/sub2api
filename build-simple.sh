#!/bin/bash
set -e

VERSION=$(cat VERSION)
COMMIT=$(git rev-parse --short HEAD)
DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || date +"%Y-%m-%dT%H:%M:%SZ")
BUILD_TYPE="production"

echo "📦 Version: $VERSION | Commit: $COMMIT"

# 清理并创建目录
rm -rf dist/upload
mkdir -p dist/upload

# 构建后端
echo "🔨 构建后端 (Linux amd64)..."
cd backend
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build \
    -tags embed \
    -ldflags "-s -w -X 'main.Version=$VERSION' -X 'main.Commit=$COMMIT' -X 'main.Date=$DATE' -X 'main.BuildType=$BUILD_TYPE'" \
    -o ../dist/upload/sub2api-linux-amd64 \
    ./cmd/server
cd ..

# 复制 VERSION
cp VERSION dist/upload/

# 创建压缩包
echo "📦 创建 tar.zst..."
cd dist/upload
tar -cf - sub2api-linux-amd64 VERSION | zstd -19 -o sub2api-${VERSION}-linux-amd64.tar.zst

# 计算 hash
sha256sum sub2api-linux-amd64 > sub2api-linux-amd64.sha256
sha256sum sub2api-${VERSION}-linux-amd64.tar.zst > sub2api-${VERSION}-linux-amd64.tar.zst.sha256

echo ""
echo "✅ 构建完成！"
echo ""
ls -lh
echo ""
echo "SHA256:"
cat *.sha256

