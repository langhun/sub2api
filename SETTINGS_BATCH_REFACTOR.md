# 批量重构 SettingsView 设置项脚本

本文档记录了如何批量重构 SettingsView.vue 中的其他独立设置项。

## 已完成

✅ Overload Cooldown (第 206-305 行)

## 待重构列表

### 1. Rate Limit 429 Cooldown (第 307-412 行)

**模板替换**:
```vue
<!-- 旧代码 -->
<div class="card">
  <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
    <h2>{{ t("admin.settings.rateLimit429Cooldown.title") }}</h2>
    <p>{{ t("admin.settings.rateLimit429Cooldown.description") }}</p>
  </div>
  <div class="space-y-5 p-6">
    <div v-if="rateLimit429CooldownLoading">...</div>
    <template v-else>
      <!-- 内容 -->
      <div class="flex justify-end">
        <button @click="saveRateLimit429CooldownSettings">...</button>
      </div>
    </template>
  </div>
</div>

<!-- 新代码 -->
<SettingsCard
  :title="t('admin.settings.rateLimit429Cooldown.title')"
  :description="t('admin.settings.rateLimit429Cooldown.description')"
  :loading="rateLimit429Cooldown.loading"
  :saving="rateLimit429Cooldown.saving"
  :show-save-button="true"
  @save="rateLimit429Cooldown.save"
>
  <!-- 内容，将 rateLimit429CooldownForm 替换为 rateLimit429Cooldown.form -->
</SettingsCard>
```

**状态替换**:
```typescript
// 删除旧代码（第 7374-7380 行）
// const rateLimit429CooldownLoading = ref(true);
// const rateLimit429CooldownSaving = ref(false);
// const rateLimit429CooldownForm = reactive({
//   enabled: true,
//   cooldown_seconds: 5,
// });

// 添加新代码
const rateLimit429Cooldown = useSettingsCard({
  loadFn: () => adminAPI.settings.getRateLimit429CooldownSettings(),
  saveFn: (data) => adminAPI.settings.updateRateLimit429CooldownSettings(data),
  successMessage: t('admin.settings.rateLimit429Cooldown.saved'),
  errorMessage: t('admin.settings.rateLimit429Cooldown.saveFailed')
})
```

**删除方法**（第 9267-9299 行）:
- `loadRateLimit429CooldownSettings()`
- `saveRateLimit429CooldownSettings()`

**更新 onMounted**:
```typescript
// 替换
loadRateLimit429CooldownSettings();
// 为
rateLimit429Cooldown.load();
```

### 2. Stream Timeout (第 414-592 行)

**状态替换**:
```typescript
const streamTimeout = useSettingsCard({
  loadFn: () => adminAPI.settings.getStreamTimeoutSettings(),
  saveFn: (data) => adminAPI.settings.updateStreamTimeoutSettings(data),
  successMessage: t('admin.settings.streamTimeout.saved'),
  errorMessage: t('admin.settings.streamTimeout.saveFailed')
})
```

**删除方法**（第 9301-9333 行）:
- `loadStreamTimeoutSettings()`
- `saveStreamTimeoutSettings()`

### 3. Request Rectifier (第 594-792 行)

**状态替换**:
```typescript
const rectifier = useSettingsCard({
  loadFn: () => adminAPI.settings.getRectifierSettings(),
  saveFn: (data) => adminAPI.settings.updateRectifierSettings(data),
  successMessage: t('admin.settings.rectifier.saved'),
  errorMessage: t('admin.settings.rectifier.saveFailed')
})
```

**删除方法**（第 9335-9367 行）:
- `loadRectifierSettings()`
- `saveRectifierSettings()`

### 4. Beta Policy (第 793-1070 行)

**状态替换**:
```typescript
const betaPolicy = useSettingsCard({
  loadFn: () => adminAPI.settings.getBetaPolicySettings(),
  saveFn: (data) => adminAPI.settings.updateBetaPolicySettings(data),
  successMessage: t('admin.settings.betaPolicy.saved'),
  errorMessage: t('admin.settings.betaPolicy.saveFailed')
})
```

**删除方法**（第 9369-9401 行）:
- `loadBetaPolicySettings()`
- `saveBetaPolicySettings()`

## 预期效果

### 代码减少统计

| 设置项 | 旧代码行数 | 新代码行数 | 减少行数 |
|--------|-----------|-----------|---------|
| Overload Cooldown | 100 | 45 | 55 |
| Rate Limit 429 | 106 | 45 | 61 |
| Stream Timeout | 179 | 50 | 129 |
| Request Rectifier | 199 | 55 | 144 |
| Beta Policy | 278 | 60 | 218 |
| **总计** | **862** | **255** | **607** |

### 方法减少统计

| 设置项 | 删除的方法 | 行数 |
|--------|-----------|------|
| Overload Cooldown | load + save | 33 |
| Rate Limit 429 | load + save | 33 |
| Stream Timeout | load + save | 33 |
| Request Rectifier | load + save | 33 |
| Beta Policy | load + save | 33 |
| **总计** | **10 个方法** | **165** |

### 总减少

- **模板代码**: 607 行
- **方法代码**: 165 行
- **状态定义**: 约 30 行
- **总计**: 约 800 行（7.6%）

## 实施步骤

1. ✅ 重构 Overload Cooldown
2. ⏳ 重构 Rate Limit 429 Cooldown
3. ⏳ 重构 Stream Timeout
4. ⏳ 重构 Request Rectifier
5. ⏳ 重构 Beta Policy
6. ⏳ 测试所有设置项
7. ⏳ 提交代码

## 注意事项

1. **保持表单字段名称一致**: 确保 `form.xxx` 的字段名与原来的 `xxxForm.xxx` 一致
2. **检查条件渲染**: 确保 `v-if` 条件正确更新为使用新的 form 对象
3. **测试保存功能**: 每个设置项重构后都要测试保存功能
4. **检查错误处理**: 确保错误消息正确显示

## 测试清单

- [ ] Overload Cooldown 加载正常
- [ ] Overload Cooldown 保存正常
- [ ] Overload Cooldown 错误处理正常
- [ ] Rate Limit 429 加载正常
- [ ] Rate Limit 429 保存正常
- [ ] Rate Limit 429 错误处理正常
- [ ] Stream Timeout 加载正常
- [ ] Stream Timeout 保存正常
- [ ] Stream Timeout 错误处理正常
- [ ] Request Rectifier 加载正常
- [ ] Request Rectifier 保存正常
- [ ] Request Rectifier 错误处理正常
- [ ] Beta Policy 加载正常
- [ ] Beta Policy 保存正常
- [ ] Beta Policy 错误处理正常
