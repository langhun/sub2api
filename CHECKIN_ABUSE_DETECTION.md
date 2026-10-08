# 签到异常行为检测报告

## 检测目标

检查用户 `jichun29@163.com` 是否存在绕过Cloudflare保护的签到行为，以及系统中是否存在其他异常签到用户。

## 签到保护机制分析

根据代码分析，系统实现了多层签到保护：

### 1. 验证码保护 (protection.go)
- **Cloudflare Turnstile**: 主要验证码方案
- **腾讯天御验证码**: 备选方案
- **阿里云验证码**: 备选方案
- 验证码验证会检查 `action` 参数，签到必须使用 `checkin` action

### 2. 频率限制 (protection.go:19-26)

```go
userAttemptLimit     = 5        // 每个用户10分钟内最多5次
userAttemptWindow    = 10分钟

ipAttemptLimit       = 5        // 每个IP 10分钟内最多5次  
ipAttemptWindow      = 10分钟

ipDailyLimit         = 10       // 每个IP 24小时内最多10次
ipDailyWindow        = 24小时
```

### 3. IP识别策略 (protection.go:145-155)
- 过滤私有IP和环回地址
- IPv4：使用完整地址
- IPv6：使用/64子网掩码
- IP经过SHA256哈希处理后存储

### 4. 幂等性保护 (service.go:464-466)
```go
func checkinIdempotencyKey(userID int64, today time.Time) string {
    return fmt.Sprintf("checkin:%d:%s", userID, today.Format("2006-01-02"))
}
```
每个用户每天只能签到一次（通过数据库唯一索引强制）

## 可能的绕过方式

### 方式1: 未启用验证码
- 如果系统配置中 `Server.Mode != "release"` 或 `Turnstile.Required = false`
- 验证码验证会被跳过 (auth_service.go:661)

### 方式2: 代理IP轮换
- 使用代理池轮换IP，绕过IP频率限制
- 但无法绕过用户级别的限制和每日一次的数据库约束

### 方式3: 自动化脚本 + 有效验证码
- 使用自动化工具，但正常通过验证码
- 这种情况下时间会相对固定（脚本定时执行）

## 检测方法

运行检测脚本查看：

1. **同一时间签到模式**: 如果用户每天都在相同时间（精确到秒）签到，高度可疑
2. **连续签到天数**: 异常高的连续签到（如50+天）可能表示自动化
3. **签到时间分布**: 正常用户签到时间应该有随机性

## 使用检测脚本

### 前置条件
```bash
pip install pymysql
```

### 设置数据库连接
```bash
export DB_HOST=your_db_host
export DB_PORT=3306
export DB_NAME=sub2api
export DB_USER=your_user
export DB_PASS=your_password
```

### 运行检测
```bash
python check_checkin.py
```

## 检测指标

### 🚩 高风险指标
- 同一精确时间签到 >= 10次
- 连续签到天数 >= 60天
- 签到时间方差 < 60秒

### ⚠️  中风险指标
- 同一小时签到 >= 80%
- 连续签到天数 >= 30天
- 总签到奖励异常高

### ✅ 正常指标
- 签到时间分布随机
- 偶尔中断连续签到
- 不同类型签到（normal + luck）混合

## 建议处置方案

### 如果确认绕过保护：

1. **立即措施**:
   - 暂停该账号签到权限
   - 审查同IP段其他账号
   - 检查服务器配置是否正确启用验证码

2. **调查措施**:
   - 查看Redis限流记录
   - 检查验证码服务日志
   - 分析是否有API直接调用

3. **长期措施**:
   - 增加设备指纹检测
   - 实施签到时间随机化（避免整点签到）
   - 添加行为分析（鼠标轨迹、停留时间）
   - 考虑引入机器学习模型识别自动化

## 配置检查清单

检查以下配置是否正确：

```yaml
# config.yaml
server:
  mode: release  # ← 必须是 release

turnstile:
  required: true  # ← 必须启用
  secret_key: "xxx"  # ← 配置有效密钥
```

或在设置中心检查：
- Turnstile是否启用
- 腾讯天御/阿里云验证码是否正确配置
- Redis连接是否正常（限流依赖Redis）
