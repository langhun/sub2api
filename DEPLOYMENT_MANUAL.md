# 🚀 Sub2API 生产部署手册

## 📦 构建信息

- **版本**: v1.0.0
- **提交**: f5ac91d9
- **构建时间**: 2026-05-21
- **包文件**: `dist/upload/sub2api-v1.0.0-linux-amd64.tar.gz`
- **SHA256**: `308a19e9c6ddb39971a377d16ae6e5f50e10947e3c54735abf41147389b54604`
- **二进制大小**: 133MB
- **压缩包大小**: 42MB

## 🖥️ 服务器信息

**SSH 连接**:
```bash
ssh -p 5522 root@45.136.13.37
# Password: Wolfsoul10356
```

**测试环境 (18808)**:
- Service: sub2api-test.service
- Binary: /opt/sub2api-test/sub2api
- DATA_DIR: /opt/sub2api-test/data
- Port: 18808

**生产环境 (8808)**:
- Service: sub2api.service
- Binary: /opt/sub2api/sub2api
- DATA_DIR: /app/data
- Port: 8808

## 📋 部署步骤

### 第一步：上传文件到服务器

```bash
# 在本地执行
cd D:/CodeSpace/sub2api
scp -P 5522 dist/upload/sub2api-v1.0.0-linux-amd64.tar.gz root@45.136.13.37:/tmp/
```

### 第二步：SSH 登录服务器

```bash
ssh -p 5522 root@45.136.13.37
# 输入密码: Wolfsoul10356
```

### 第三步：解压并验证

```bash
cd /tmp
tar -xzf sub2api-v1.0.0-linux-amd64.tar.gz

# 验证 hash
sha256sum sub2api-linux-amd64
# 应该输出: 308a19e9c6ddb39971a377d16ae6e5f50e10947e3c54735abf41147389b54604

# 检查版本
chmod +x sub2api-linux-amd64
./sub2api-linux-amd64 -version
```

### 第四步：部署到测试环境 (18808)

```bash
# 停止测试服务
systemctl stop sub2api-test.service

# 备份旧版本
cp /opt/sub2api-test/sub2api /opt/sub2api-test/sub2api.backup.$(date +%Y%m%d_%H%M%S)

# 复制新版本
cp /tmp/sub2api-linux-amd64 /opt/sub2api-test/sub2api
chmod +x /opt/sub2api-test/sub2api

# 启动测试服务
systemctl start sub2api-test.service

# 检查状态
systemctl status sub2api-test.service
```

### 第五步：验证测试环境 (18808)

```bash
# 1. 检查服务状态
systemctl is-active sub2api-test.service
# 应该输出: active

# 2. 检查健康状态
curl http://127.0.0.1:18808/health
# 应该输出: {"status":"ok"}

# 3. 检查 setup 状态
curl http://127.0.0.1:18808/setup/status
# 应该输出: {"needs_setup":false}

# 4. 检查 API
curl http://127.0.0.1:18808/api/v1/public/pricing
# 应该返回 JSON

# 5. 检查监控 API
curl http://127.0.0.1:18808/api/v1/monitoring/summary
# 应该返回 JSON

# 6. 查看日志
journalctl -u sub2api-test.service -n 50 --no-pager

# 7. 检查端口监听
ss -lntp | grep 18808
```

### 第六步：推广到生产环境 (8808)

**⚠️ 确认测试环境验证通过后再执行！**

```bash
# 停止生产服务
systemctl stop sub2api.service

# 备份旧版本
cp /opt/sub2api/sub2api /opt/sub2api/sub2api.backup.$(date +%Y%m%d_%H%M%S)

# 复制测试环境验证过的二进制文件
cp /opt/sub2api-test/sub2api /opt/sub2api/sub2api
chmod +x /opt/sub2api/sub2api

# 启动生产服务
systemctl start sub2api.service

# 检查状态
systemctl status sub2api.service
```

### 第七步：验证生产环境 (8808)

```bash
# 1. 检查服务状态
systemctl is-active sub2api.service
# 应该输出: active

# 2. 检查健康状态
curl http://127.0.0.1:8808/health
# 应该输出: {"status":"ok"}

# 3. 检查 setup 状态
curl http://127.0.0.1:8808/setup/status
# 应该输出: {"needs_setup":false}

# 4. 检查 API
curl http://127.0.0.1:8808/api/v1/public/pricing
# 应该返回 JSON

# 5. 检查监控 API
curl http://127.0.0.1:8808/api/v1/monitoring/summary
# 应该返回 JSON

# 6. 查看日志
journalctl -u sub2api.service -n 50 --no-pager

# 7. 检查端口监听
ss -lntp | grep 8808
```

### 第八步：停止测试服务

```bash
# 生产环境验证通过后，停止并禁用测试服务
systemctl stop sub2api-test.service
systemctl disable sub2api-test.service

echo "✅ 测试服务已停止并禁用"
```

### 第九步：清理临时文件

```bash
rm -f /tmp/sub2api-v1.0.0-linux-amd64.tar.gz
rm -f /tmp/sub2api-linux-amd64
rm -f /tmp/VERSION

echo "✅ 临时文件已清理"
```

## ⚠️ 故障处理

### 问题 1: 服务状态显示 status=203/EXEC

**原因**: 上传了错误平台的二进制文件

**解决**: 重新构建 Linux amd64 版本并上传

### 问题 2: systemctl 显示 active 但应用未响应

**检查**:
```bash
# 检查端口监听
ss -lntp | grep 18808  # 或 8808

# 检查进程
ps aux | grep sub2api

# 查看详细日志
journalctl -u sub2api-test.service -f  # 或 sub2api.service
```

### 问题 3: 健康检查失败

**检查**:
```bash
# 查看配置文件
cat /opt/sub2api-test/config.yaml  # 或 /opt/sub2api/config.yaml

# 检查数据库连接
# 检查日志中的错误信息
```

## 📊 验证清单

### 测试环境 (18808) ✅

- [ ] 二进制文件 hash 匹配
- [ ] `/opt/sub2api-test/sub2api -version` 正确
- [ ] `systemctl is-active sub2api-test.service` 显示 active
- [ ] `http://127.0.0.1:18808/health` 返回 OK
- [ ] `http://127.0.0.1:18808/setup/status` 返回 needs_setup=false
- [ ] `http://127.0.0.1:18808/api/v1/public/pricing` 返回 JSON
- [ ] `http://127.0.0.1:18808/api/v1/monitoring/summary` 返回 JSON
- [ ] journalctl 日志无错误
- [ ] 首页资源文件名已更新（证明前端已更新）

### 生产环境 (8808) ✅

- [ ] 使用与测试环境完全相同的二进制文件
- [ ] Hash 仍然匹配
- [ ] `/opt/sub2api/sub2api -version` 正确
- [ ] `systemctl is-active sub2api.service` 显示 active
- [ ] `http://127.0.0.1:8808/health` 返回 OK
- [ ] `http://127.0.0.1:8808/setup/status` 返回 needs_setup=false
- [ ] `http://127.0.0.1:8808/api/v1/public/pricing` 返回 JSON
- [ ] `http://127.0.0.1:8808/api/v1/monitoring/summary` 返回 JSON
- [ ] journalctl 日志无错误
- [ ] 测试服务已停止并禁用

## 🔄 回滚步骤

如果生产环境出现问题，立即回滚：

```bash
# 停止服务
systemctl stop sub2api.service

# 恢复备份
cp /opt/sub2api/sub2api.backup.YYYYMMDD_HHMMSS /opt/sub2api/sub2api
chmod +x /opt/sub2api/sub2api

# 启动服务
systemctl start sub2api.service

# 验证
systemctl status sub2api.service
curl http://127.0.0.1:8808/health
```

## 📝 部署后检查

1. **监控关键指标**:
   - API 响应时间
   - 错误率
   - 数据库查询性能
   - 系统资源使用

2. **检查日志**:
   ```bash
   # 持续监控日志
   journalctl -u sub2api.service -f
   
   # 查找错误
   journalctl -u sub2api.service | grep -i error
   ```

3. **用户反馈**:
   - 监控用户报告的问题
   - 检查关键功能是否正常

## 🎉 部署完成

部署完成后，确认：
- ✅ 测试环境 (18808) 验证通过
- ✅ 生产环境 (8808) 运行正常
- ✅ 测试服务已停止并禁用
- ✅ 临时文件已清理
- ✅ 备份文件已保留

---

**部署日期**: 2026-05-21  
**版本**: v1.0.0  
**提交**: f5ac91d9  
**部署人**: Claude Opus 4.7
