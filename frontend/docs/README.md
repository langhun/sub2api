# 前端文档

本目录包含前端项目的技术文档。

## 文档列表

### [COMPONENTS.md](./COMPONENTS.md)

组件和工具函数的完整文档，包括：

- **组件**
  - SettingsCard - 标准化设置卡片容器
  - ProxyBulkActionsBar - 代理批量操作工具栏

- **Composables**
  - useSettingsCard - 设置加载和保存逻辑
  - useAccountBulkOperations - 批量操作处理
  - useProxyTesting - 代理测试和质量检查

- **工具函数**
  - accountFilters - 账户过滤逻辑集合

每个组件和工具都包含：
- 功能描述
- 完整的 API 文档
- 使用示例
- 最佳实践
- 注意事项

## 快速开始

1. 查看 [COMPONENTS.md](./COMPONENTS.md) 了解可用的组件和工具
2. 在源代码中查看 JSDoc 注释获取详细的类型信息
3. 参考文档中的示例代码快速上手

## 贡献

添加新组件或工具时，请：
1. 在源代码中添加完整的 JSDoc 注释
2. 在 COMPONENTS.md 中添加相应的文档条目
3. 提供至少一个完整的使用示例
4. 更新 COMPONENTS.md 的更新日志

## 文档规范

- 所有公开的函数和接口必须有 JSDoc 注释
- 组件必须在文件顶部添加 HTML 注释说明
- 示例代码必须是可运行的完整代码
- 使用中文编写文档内容
