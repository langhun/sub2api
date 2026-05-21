# Mihomo 集成错误处理和日志改进总结

## 改进概述

本次改进针对 Mihomo 集成的错误处理和日志记录进行了全面优化，增强了系统的可观测性和调试能力。

## 主要改进

### 1. 端口冲突处理改进

**文件**: `backend/internal/repository/proxy_subscription_mihomo_runtime_manager.go:350-360`

**改进内容**:
- 添加了详细的错误日志，记录冲突的端点和运行时 ID
- 在检测到端口冲突时，将状态标记为 Stopped
- 简化了错误消息，使其更加清晰

```go
if previousRuntimeID, exists := listenerEndpoints[endpoint]; exists {
    slog.Error("duplicate mihomo listener endpoint detected",
        "endpoint", endpoint,
        "runtime_id", runtimeID,
        "conflicting_runtime_id", previousRuntimeID)
    
    state.Stopped = true
    return nil, fmt.Errorf("duplicate endpoint %s", endpoint)
}
```

### 2. 订阅源获取失败日志增强

**文件**: `backend/internal/service/proxy_subscription_service.go:182-189`

**改进内容**:
- 添加了包含 source_id 和 source_url 的错误日志
- 便于快速定位失败的订阅源

```go
payload, err := s.fetchSubscriptionContent(ctx, source.URL)
if err != nil {
    slog.Error("proxy_subscription.fetch_failed",
        "source_id", id,
        "source_url", source.URL,
        "error", err)
    // ... 错误处理
}
```

### 3. 运行时管理调试日志

**文件**: `backend/internal/repository/proxy_subscription_mihomo_runtime_manager.go`

**新增调试日志**:

#### UpsertRuntime 操作
- 运行时创建开始
- 端口分配成功/失败
- 配置应用失败
- 监听器未就绪
- 运行时创建成功

#### DeleteRuntime 操作
- 删除开始
- 运行时未找到
- 配置应用失败
- 删除成功

#### 端口分配
- 重用现有端口
- 使用首选端口
- 首选端口不可用
- 分配新端口
- 无可用端口

### 4. 订阅服务调试日志

**文件**: `backend/internal/service/proxy_subscription_service.go`

**新增调试日志**:

#### RefreshSource 操作
- 刷新开始
- 载荷解析完成（包含格式、节点数、错误数）
- 节点过滤完成（包含运行时节点数、过滤错误数）

#### materializeRuntimeProxy 操作
- 物化开始（包含节点详情）
- 运行时未配置警告
- 运行时 upsert 失败
- 运行时 upsert 成功（包含监听器信息）

### 5. 错误消息格式统一

所有错误消息现在都包含:
- 操作上下文（runtime_id, source_id 等）
- 相关资源标识（endpoint, port, url 等）
- 详细的错误信息

### 6. 资源清理改进

**文件**: `backend/internal/repository/proxy_subscription_mihomo_runtime_manager.go:265-283`

**改进内容**:
- 删除运行时时添加文件清理错误日志
- 区分文件不存在和其他错误
- 添加删除成功的调试日志

## 日志级别使用

- **DEBUG**: 正常操作流程的详细信息（开始、成功、端口分配等）
- **WARN**: 非致命问题（文件清理失败、运行时未配置等）
- **ERROR**: 操作失败（端口冲突、配置应用失败、监听器未就绪等）

## 统计数据

- 运行时管理器新增 19 条日志语句
- 订阅服务新增 9 条日志语句
- 总计新增 177 行代码
- 删除 22 行旧代码

## 影响范围

改进涉及以下关键路径:
1. Mihomo 运行时生命周期管理
2. 订阅源刷新流程
3. 代理节点物化过程
4. 端口分配和冲突检测
5. 资源清理和错误恢复

## 后续建议

1. 在生产环境中启用 DEBUG 日志级别进行初步验证
2. 监控日志输出，确保没有性能影响
3. 根据实际运行情况调整日志级别和内容
4. 考虑添加指标收集（metrics）以补充日志
