#!/bin/bash
# Sub2API 推广到生产环境脚本

set -e

echo "🚀 推广到生产环境 (8808)"
echo "========================"
echo ""

read -p "确认测试环境 (18808) 已验证通过？(yes/no): " confirm
if [ "$confirm" != "yes" ]; then
    echo "❌ 取消部署"
    exit 1
fi

echo "📤 推广到生产环境..."
ssh -p 5522 root@45.136.13.37 << 'ENDSSH'
# 停止生产服务
systemctl stop sub2api.service

# 备份旧版本
if [ -f /opt/sub2api/sub2api ]; then
    cp /opt/sub2api/sub2api /opt/sub2api/sub2api.backup.$(date +%Y%m%d_%H%M%S)
fi

# 复制测试环境验证过的二进制文件
cp /opt/sub2api-test/sub2api /opt/sub2api/sub2api
chmod +x /opt/sub2api/sub2api

# 启动生产服务
systemctl start sub2api.service
sleep 3

# 检查状态
systemctl is-active sub2api.service
ENDSSH

echo "✅ 步骤 1: 验证生产环境 (8808)..."
ssh -p 5522 root@45.136.13.37 << 'ENDSSH'
echo "检查服务状态..."
systemctl status sub2api.service --no-pager | head -10

echo ""
echo "检查健康状态..."
curl -s http://127.0.0.1:8808/health || echo "健康检查失败"

echo ""
echo "检查 setup 状态..."
curl -s http://127.0.0.1:8808/setup/status || echo "setup 检查失败"

echo ""
echo "检查 API..."
curl -s http://127.0.0.1:8808/api/v1/public/pricing | head -c 100 || echo "API 检查失败"

echo ""
echo "查看最近日志..."
journalctl -u sub2api.service -n 20 --no-pager
ENDSSH

echo ""
echo "✅ 步骤 2: 停止测试服务..."
ssh -p 5522 root@45.136.13.37 << 'ENDSSH'
systemctl stop sub2api-test.service
systemctl disable sub2api-test.service
echo "✅ 测试服务已停止并禁用"
ENDSSH

echo ""
echo "🎉 生产环境部署完成！"
echo ""
echo "请访问 https://your-domain.com 验证"
