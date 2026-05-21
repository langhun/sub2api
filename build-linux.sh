#!/bin/bash
set -e

echo "🚀 开始构建 Sub2API Linux amd64 版本..."

# 读取版本信息
VERSION=$(cat VERSION)
COMMIT=$(git rev-parse --short HEAD)
DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
BUILD_TYPE="production"

echo "📦 版本信息:"
echo "  Version: $VERSION"
echo "  Commit: $COMMIT"
echo "  Date: $DATE"
echo "  BuildType: $BUILD_TYPE"

# 清理旧的构建产物
echo "🧹 清理旧的构建产物..."
rm -rf dist/upload
mkdir -p dist/upload

# 等待前端构建完成
echo "⏳ 等待前端构建完成..."
while [ ! -d "frontend/dist" ]; do
    sleep 2
done
echo "✅ 前端构建完成"

# 复制前端构建产物到后端
echo "📋 复制前端构建产物..."
rm -rf backend/internal/web/dist
cp -r frontend/dist backend/internal/web/

# 构建后端 (Linux amd64)
echo "🔨 构建后端 (Linux amd64)..."
cd backend

GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build \
    -tags embed \
    -ldflags "-s -w \
        -X 'main.Version=$VERSION' \
        -X 'main.Commit=$COMMIT' \
        -X 'main.Date=$DATE' \
        -X 'main.BuildType=$BUILD_TYPE'" \
    -o ../dist/upload/sub2api-linux-amd64 \
    ./cmd/server

cd ..

# 复制配置文件和 VERSION
echo "📄 复制配置文件..."
cp VERSION dist/upload/
cp config.example.yaml dist/upload/

# 创建 tar.zst 压缩包
echo "📦 创建 tar.zst 压缩包..."
cd dist/upload
tar -cf - sub2api-linux-amd64 VERSION config.example.yaml | zstd -19 -o sub2api-${VERSION}-linux-amd64.tar.zst
cd ../..

# 计算 hash
echo "🔐 计算文件 hash..."
cd dist/upload
sha256sum sub2api-linux-amd64 > sub2api-linux-amd64.sha256
sha256sum sub2api-${VERSION}-linux-amd64.tar.zst > sub2api-${VERSION}-linux-amd64.tar.zst.sha256
cd ../..

# 显示构建结果
echo ""
echo "✅ 构建完成！"
echo ""
echo "📦 构建产物:"
ls -lh dist/upload/
echo ""
echo "🔐 SHA256:"
cat dist/upload/sub2api-linux-amd64.sha256
cat dist/upload/sub2api-${VERSION}-linux-amd64.tar.zst.sha256
echo ""
echo "📤 上传文件: dist/upload/sub2api-${VERSION}-linux-amd64.tar.zst"
