# API 变更日志

本文档记录 Sub2API 的 API 版本历史、重大变更和迁移指南。

## 版本规范

Sub2API 遵循语义化版本控制：
- **主版本号**：不兼容的 API 变更
- **次版本号**：向后兼容的功能新增
- **修订号**：向后兼容的问题修复

## 当前版本

### v1.0.0 (2024-01-15)

当前稳定版本，提供完整的 API 功能。

#### 核心功能
- 用户认证和授权（JWT + API Key）
- 多模型 AI 网关（OpenAI、Anthropic、Gemini 等）
- 余额管理和转账系统
- 红包功能
- 使用统计和监控
- 管理员后台接口

#### API 端点
- `/api/auth/*` - 认证相关
- `/api/user/*` - 用户管理
- `/api/balance/*` - 余额和转账
- `/v1/chat/completions` - OpenAI 兼容接口
- `/v1/messages` - Anthropic 兼容接口
- `/admin/*` - 管理员接口

## 历史版本

### v0.9.0 (2023-12-01) - Beta 版本

#### 新增功能
- 初始 API 实现
- 基础用户系统
- OpenAI 网关支持

#### 已知问题
- 响应格式不统一
- 缺少完整的错误处理

## 重大变更

### 响应格式标准化 (v1.0.0)

**变更时间**: 2024-01-15

**影响**: 所有 API 端点

**变更内容**:
统一所有 API 响应格式为标准信封格式：

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

**迁移指南**:
1. 更新客户端代码，从 `response.data` 获取实际数据
2. 检查 `response.code` 而不是 HTTP 状态码
3. 错误响应包含 `reason` 和 `metadata` 字段

**示例**:

旧格式：
```json
{
  "items": [...],
  "total": 100
}
```

新格式：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [...],
    "total": 100,
    "page": 1,
    "page_size": 20,
    "pages": 5
  }
}
```

### 分页参数标准化 (v1.0.0)

**变更时间**: 2024-01-15

**影响**: 所有分页端点

**变更内容**:
- 统一使用 `page` 和 `page_size` 参数
- 响应包含 `pages` 字段（总页数）
- 支持 `limit` 作为 `page_size` 的别名

**迁移指南**:
更新分页请求参数：
- `offset` → `page`
- `limit` → `page_size`（或继续使用 `limit`）

### 错误处理增强 (v1.0.0)

**变更时间**: 2024-01-15

**影响**: 所有 API 端点

**变更内容**:
错误响应现在包含结构化信息：

```json
{
  "code": 400,
  "message": "Invalid request",
  "reason": "INVALID_AMOUNT",
  "metadata": {
    "field": "amount",
    "constraint": "must be positive"
  }
}
```

**迁移指南**:
1. 使用 `reason` 字段进行错误类型判断
2. 从 `metadata` 获取详细错误信息
3. 不要依赖 `message` 的具体文本（可能变化）

## 废弃通知

### 即将废弃

目前没有计划废弃的 API。

### 已废弃

#### 直接使用 gin.H 的响应格式 (v0.9.0)

**废弃时间**: 2024-01-15  
**移除时间**: 已移除

所有端点已迁移到标准 response 包。

## 迁移指南

### 从 v0.9.0 迁移到 v1.0.0

#### 1. 更新响应处理

**JavaScript/TypeScript 示例**:

```typescript
// 旧代码
const response = await fetch('/api/balance/transfer/history');
const data = await response.json();
console.log(data.items); // 直接访问

// 新代码
const response = await fetch('/api/balance/transfer/history');
const result = await response.json();
if (result.code === 0) {
  console.log(result.data.items); // 从 data 字段访问
} else {
  console.error(result.message, result.reason);
}
```

#### 2. 更新错误处理

**Go 示例**:

```go
// 旧代码
if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    return
}

// 新代码
if err != nil {
    response.ErrorFrom(c, err)
    return
}
```

#### 3. 更新分页处理

```typescript
// 旧代码
const params = new URLSearchParams({
  offset: '0',
  limit: '20'
});

// 新代码
const params = new URLSearchParams({
  page: '1',
  page_size: '20'
});
```

## 最佳实践

### 1. 版本控制

虽然当前 API 不使用 URL 版本号，但建议：
- 在客户端记录使用的 API 版本
- 订阅变更通知
- 在测试环境验证新版本

### 2. 错误处理

```typescript
function handleApiError(result: ApiResponse) {
  switch (result.reason) {
    case 'INSUFFICIENT_BALANCE':
      showBalanceWarning();
      break;
    case 'INVALID_AMOUNT':
      showValidationError(result.metadata);
      break;
    default:
      showGenericError(result.message);
  }
}
```

### 3. 向后兼容

客户端应该：
- 忽略未知的响应字段
- 为新字段提供默认值
- 不依赖字段顺序

## 联系方式

如有 API 相关问题：
- GitHub Issues: https://github.com/Wei-Shaw/sub2api/issues
- 文档: https://github.com/Wei-Shaw/sub2api/wiki



