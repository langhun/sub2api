# 签到异常检测工具使用指南

## 问题描述

用户 `jichun29@163.com` 疑似每天同一时间签到，怀疑绕过了Cloudflare保护。

## 检测工具

提供了三种检测方式：

### 方式1: Shell脚本（推荐，最简单）

```bash
# 设置数据库连接信息
export DB_HOST=your_host
export DB_PORT=3306
export DB_NAME=sub2api
export DB_USER=your_user
export DB_PASS=your_password

# 运行检测
./check_checkin.sh
```

或者创建 `.env` 文件：
```bash
DB_HOST=localhost
DB_PORT=3306
DB_NAME=sub2api
DB_USER=root
DB_PASS=your_password
```

然后直接运行：
```bash
./check_checkin.sh
```

### 方式2: Python脚本（更详细的分析）

```bash
# 安装依赖
pip install pymysql

# 设置环境变量（同上）
export DB_HOST=...

# 运行
python check_checkin.py
```

### 方式3: Go程序（如果你喜欢Go）

```bash
# 设置环境变量
export DATABASE_URL="user:pass@tcp(host:port)/dbname"

# 编译运行
go run check_checkin_abuse.go
```

## 检测内容

### 1. 用户签到历史
- 查看最近60次签到记录
- 分析签到时间分布
- 检查时间模式

### 2. 时间模式分析
- 统计每个精确时间的签到次数
- 如果某个时间出现>=10次，标记为可疑
- 计算时间分布的集中度

### 3. 全局异常用户
- 查找所有同一时间签到>=10次的用户
- 查找连续签到>=30天的用户
- 查找总签到>=50次的用户

### 4. 账户信息
- 用户状态
- 累计奖励
- 余额情况

## 判断标准

### 🚨 高度可疑（自动化脚本）
- ✅ 同一精确时间（精确到秒）签到 >= 15次
- ✅ 签到时间集中度 > 80%（80%以上签到在同一时间）
- ✅ 连续签到 >= 60天
- ✅ 时间方差 < 60秒

### ⚠️ 中度可疑
- 同一小时签到 >= 90%
- 连续签到 30-59天
- 同一时间签到 10-14次

### ✅ 正常
- 签到时间随机分布
- 有中断（不是每天都签到）
- 时间分散在不同小时

## 系统保护机制

根据代码分析，系统有以下保护：

### 1. 验证码验证
- Cloudflare Turnstile
- 腾讯天御验证码
- 阿里云验证码

**检查点**: `backend/internal/service/auth_service.go:660-702`

### 2. 频率限制
- 用户限制: 10分钟内最多5次尝试
- IP限制: 10分钟内最多5次，24小时最多10次

**检查点**: `backend/internal/custom/modules/activity/checkin/protection.go:19-26`

### 3. 每日唯一性
- 数据库唯一索引: `(user_id, checkin_date)`
- 每天只能签到一次

## 可能的绕过方式

### 1. 配置问题
```yaml
# 检查 config.yaml
server:
  mode: release  # 如果不是 release，验证码可能被跳过

turnstile:
  required: true  # 如果是 false，验证码不强制
```

### 2. 验证码服务未启用
检查管理后台设置，确认验证码服务已正确配置。

### 3. 使用代理IP池
- 轮换IP绕过IP限制
- 但无法绕过用户限制和每日唯一性

### 4. 合法自动化（最可能）
- 使用浏览器自动化工具（Selenium, Playwright等）
- 正常通过验证码
- 定时触发，所以时间固定

## 处置建议

### 如果确认是自动化：

#### 1. 立即措施
```sql
-- 暂停签到权限（设置用户状态或添加黑名单）
UPDATE users SET status = 'suspended' WHERE email = 'jichun29@163.com';

-- 或者添加到签到黑名单（如果有相关表）
```

#### 2. 调查措施
- 检查该用户的其他行为
- 查看同IP的其他账户
- 检查Redis限流记录

#### 3. 加固措施
- 确认验证码配置正确
- 添加设备指纹检测
- 增加行为分析（签到前页面停留时间）
- 考虑添加签到时间随机性要求

## 快速诊断命令

```bash
# 直接查看该用户最近签到时间
mysql -h host -u user -p dbname -e "
SELECT TIME(created_at) as time, COUNT(*) as cnt
FROM checkins c JOIN users u ON c.user_id = u.id  
WHERE u.email = 'jichun29@163.com'
GROUP BY TIME(created_at)
ORDER BY cnt DESC LIMIT 5;
"
```

如果返回某个时间的cnt > 10，基本确认是自动化脚本。

## 技术细节

详细的代码分析和保护机制说明，请查看：`CHECKIN_ABUSE_DETECTION.md`
