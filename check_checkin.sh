#!/bin/bash
# 签到异常行为快速检测脚本
# 使用 MySQL 命令行工具直接查询

# 从 .env 文件或环境变量读取数据库配置
if [ -f .env ]; then
    source .env
fi

DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-3306}"
DB_NAME="${DB_NAME:-sub2api}"
DB_USER="${DB_USER:-root}"
DB_PASS="${DB_PASS}"

# 构建MySQL连接命令
MYSQL_CMD="mysql -h${DB_HOST} -P${DB_PORT} -u${DB_USER}"
if [ -n "$DB_PASS" ]; then
    MYSQL_CMD="$MYSQL_CMD -p${DB_PASS}"
fi
MYSQL_CMD="$MYSQL_CMD ${DB_NAME}"

echo "======================================================================"
echo "签到异常行为检测 - jichun29@163.com"
echo "======================================================================"

# 1. 查询特定用户的签到记录
echo -e "\n[1] 用户 jichun29@163.com 最近60次签到记录:"
$MYSQL_CMD <<SQL
SELECT
    u.email,
    c.checkin_date,
    TIME(c.created_at) as time,
    c.reward_amount,
    c.checkin_type,
    c.streak_days
FROM checkins c
JOIN users u ON c.user_id = u.id
WHERE u.email = 'jichun29@163.com'
ORDER BY c.checkin_date DESC
LIMIT 60;
SQL

# 2. 分析该用户的时间模式
echo -e "\n[2] 该用户签到时间分布:"
$MYSQL_CMD <<SQL
SELECT
    TIME(c.created_at) as exact_time,
    COUNT(*) as count,
    ROUND(COUNT(*) * 100.0 / (SELECT COUNT(*) FROM checkins c2
        JOIN users u2 ON c2.user_id = u2.id
        WHERE u2.email = 'jichun29@163.com'), 1) as percentage
FROM checkins c
JOIN users u ON c.user_id = u.id
WHERE u.email = 'jichun29@163.com'
GROUP BY TIME(c.created_at)
ORDER BY count DESC
LIMIT 10;
SQL

# 3. 查找所有疑似自动化签到的用户
echo -e "\n[3] 所有疑似自动化签到用户（同一时间>=10次）:"
$MYSQL_CMD <<SQL
SELECT
    u.email,
    TIME(c.created_at) as time,
    COUNT(*) as same_time_count,
    DATE(MIN(c.checkin_date)) as first_date,
    DATE(MAX(c.checkin_date)) as last_date,
    MAX(c.streak_days) as max_streak
FROM checkins c
JOIN users u ON c.user_id = u.id
WHERE c.checkin_date >= DATE_SUB(CURDATE(), INTERVAL 60 DAY)
GROUP BY u.id, TIME(c.created_at)
HAVING same_time_count >= 10
ORDER BY same_time_count DESC
LIMIT 30;
SQL

# 4. 查找连续签到天数异常高的用户
echo -e "\n[4] 连续签到天数TOP20:"
$MYSQL_CMD <<SQL
SELECT
    u.email,
    u.status,
    MAX(c.streak_days) as max_streak,
    COUNT(*) as total_checkins,
    ROUND(SUM(c.reward_amount), 2) as total_rewards,
    DATE(MIN(c.checkin_date)) as first_checkin,
    DATE(MAX(c.checkin_date)) as last_checkin,
    DATEDIFF(MAX(c.checkin_date), MIN(c.checkin_date)) as span_days
FROM checkins c
JOIN users u ON c.user_id = u.id
GROUP BY u.id
HAVING max_streak >= 30 OR total_checkins >= 50
ORDER BY max_streak DESC, total_checkins DESC
LIMIT 20;
SQL

# 5. 检查该用户的账户信息
echo -e "\n[5] 用户账户信息:"
$MYSQL_CMD <<SQL
SELECT
    u.id,
    u.email,
    u.status,
    u.role,
    u.balance,
    u.created_at,
    (SELECT COUNT(*) FROM checkins WHERE user_id = u.id) as total_checkins,
    (SELECT MAX(streak_days) FROM checkins WHERE user_id = u.id) as max_streak,
    (SELECT SUM(reward_amount) FROM checkins WHERE user_id = u.id) as total_checkin_rewards
FROM users u
WHERE u.email = 'jichun29@163.com';
SQL

echo -e "\n======================================================================"
echo "检测完成"
echo "======================================================================"
