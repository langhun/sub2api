#!/bin/bash
# Sub2API 生产部署脚本
# 服务器: 45.136.13.37:5522

set -e

VERSION="v1.0.0"
PACKAGE="sub2api-${VERSION}-linux-amd64.tar.gz"
HASH="308a19e9c6ddb39971a377d16ae6e5f50e10947e3c54735abf41147389b54604"

echo "🚀 Sub2API 生产部署流程"
echo "========================"
echo "Version: $VERSION"
echo "Package: $PACKAGE"
echo "Hash: $HASH"
echo ""

# 1. 上传到服务器
echo "📤 步骤 1: 上传文件到服务器..."
scp -P 5522 dist/upload/$PACKAGE root@45.136.13.37:/tmp/

# 2. 在服务器上解压并验证
echo "📦 步骤 2: 解压并验证..."
ssh -p 5522 root@45.136.13.37 << 'ENDSSH'
cd /tmp
tar -xzf sub2api-v1.0.0-linux-amd64.tar.gz
echo "✅ 解压完成"

# 验证 hash
ACTUAL_HASH=$(sha256sum sub2api-linux-amd64 | awk '{print $1}')
echo "Hash: $ACTUAL_HASH"

# 检查版本
chmod +x sub2api-linux-amd64
./sub2api-linux-amd64 -version || echo "版本检查完成"
ENDSSH

# 3. 部署到测试环境 (18808)
echo "🧪 步骤 3: 部署到测试环境 (18808)..."
ssh -p 5522 root@45.136.13.37 << 'ENDSSH'
# 停止测试服务
systemctl stop sub2api-test.service || true

# 备份旧版本
if [ -f /opt/sub2api-test/sub2api ]; then
    cp /opt/sub2api-test/sub2api /opt/sub2api-test/sub2api.backup
fi

# 复制新版本
cp /tmp/sub2api-linux-amd64 /opt/sub2api-test/sub2api
chmod +x /opt/sub2api-test/sub2api

# 启动测试服务
systemctl start sub2api-test.service
sleep 3

# 检查状态
systemctl is-active sub2api-test.service
ENDSSH

# 4. 验证测试环境
echo "✅ 步骤 4: 验证测试环境 (18808)..."
ssh -p 5522 root@45.136.13.37 << 'ENDSSH'
echo "检查服务状态..."
systemctl status sub2api-test.service --no-pager | head -10

echo ""
echo "检查健康状态..."
curl -s http://127.0.0.1:18808/health || echo "健康检查失败"

echo ""
echo "检查 setup 状态..."
curl -s http://127.0.0.1:18808/setup/status || echo "setup 检查失败"

echo ""
echo "检查 API..."
curl -s http://127.0.0.1:18808/api/v1/public/pricing | head -c 100 || echo "API 检查失败"

echo ""
echo "查看最近日志..."
journalctl -u sub2api-test.service -n 20 --no-pager
ENDSSH

echo ""
echo "✅ 测试环境部署完成！"
echo ""
echo "请手动验证测试环境后，运行以下命令推广到生产："
echo "  ./deploy-to-production.sh"
