# SettingsView 重构指南

## 目标

将 SettingsView.vue 中的独立设置项重构为使用 `SettingsCard` 组件和 `useSettingsCard` composable，以减少重复代码并提高可维护性。

## 重构步骤

### 步骤 1: 导入新组件和 Composable

在 `<script setup>` 部分添加导入：

```typescript
import SettingsCard from '@/components/admin/settings/SettingsCard.vue'
import { useSettingsCard } from '@/composables/useSettingsCard'
```

### 步骤 2: 重构 Overload Cooldown 设置项

#### 原代码（第 206-305 行）

```vue
<!-- Overload Cooldown (529) Settings -->
<div class="card">
  <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
    <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
      {{ t("admin.settings.overloadCooldown.title") }}
    </h2>
    <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
      {{ t("admin.settings.overloadCooldown.description") }}
    </p>
  </div>
  <div class="space-y-5 p-6">
    <div v-if="overloadCooldownLoading" class="flex items-center gap-2 text-gray-500">
      <div class="h-4 w-4 animate-spin rounded-full border-b-2 border-primary-600"></div>
      {{ t("common.loading") }}
    </div>

    <template v-else>
      <div class="flex items-center justify-between">
        <div>
          <label class="font-medium text-gray-900 dark:text-white">
            {{ t("admin.settings.overloadCooldown.enabled") }}
          </label>
          <p class="text-sm text-gray-500 dark:text-gray-400">
            {{ t("admin.settings.overloadCooldown.enabledHint") }}
          </p>
        </div>
        <Toggle v-model="overloadCooldownForm.enabled" />
      </div>

      <div
        v-if="overloadCooldownForm.enabled"
        class="space-y-4 border-t border-gray-100 pt-4 dark:border-dark-700"
      >
        <div>
          <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
            {{ t("admin.settings.overloadCooldown.cooldownMinutes") }}
          </label>
          <input
            v-model.number="overloadCooldownForm.cooldown_minutes"
            type="number"
            min="1"
            max="120"
            class="input w-32"
          />
          <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
            {{ t("admin.settings.overloadCooldown.cooldownMinutesHint") }}
          </p>
        </div>
      </div>

      <div class="flex justify-end border-t border-gray-100 pt-4 dark:border-dark-700">
        <button
          type="button"
          @click="saveOverloadCooldownSettings"
          :disabled="overloadCooldownSaving"
          class="btn btn-primary btn-sm"
        >
          <svg v-if="overloadCooldownSaving" class="mr-1 h-4 w-4 animate-spin" ...>...</svg>
          {{ overloadCooldownSaving ? t("common.saving") : t("common.save") }}
        </button>
      </div>
    </template>
  </div>
</div>
```

#### 新代码

**模板部分：**

```vue
<!-- Overload Cooldown (529) Settings -->
<SettingsCard
  :title="t('admin.settings.overloadCooldown.title')"
  :description="t('admin.settings.overloadCooldown.description')"
  :loading="overloadCooldown.loading"
  :saving="overloadCooldown.saving"
  :show-save-button="true"
  @save="overloadCooldown.save"
>
  <div class="flex items-center justify-between">
    <div>
      <label class="font-medium text-gray-900 dark:text-white">
        {{ t("admin.settings.overloadCooldown.enabled") }}
      </label>
      <p class="text-sm text-gray-500 dark:text-gray-400">
        {{ t("admin.settings.overloadCooldown.enabledHint") }}
      </p>
    </div>
    <Toggle v-model="overloadCooldown.form.enabled" />
  </div>

  <div
    v-if="overloadCooldown.form.enabled"
    class="space-y-4 border-t border-gray-100 pt-4 dark:border-dark-700"
  >
    <div>
      <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
        {{ t("admin.settings.overloadCooldown.cooldownMinutes") }}
      </label>
      <input
        v-model.number="overloadCooldown.form.cooldown_minutes"
        type="number"
        min="1"
        max="120"
        class="input w-32"
      />
      <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
        {{ t("admin.settings.overloadCooldown.cooldownMinutesHint") }}
      </p>
    </div>
  </div>
</SettingsCard>
```

**脚本部分（替换第 7366-7372 行和 9285-9316 行）：**

```typescript
// 删除旧的状态定义
// const overloadCooldownLoading = ref(true);
// const overloadCooldownSaving = ref(false);
// const overloadCooldownForm = reactive({
//   enabled: true,
//   cooldown_minutes: 10,
// });

// 删除旧的方法
// async function loadOverloadCooldownSettings() { ... }
// async function saveOverloadCooldownSettings() { ... }

// 使用新的 composable
const overloadCooldown = useSettingsCard({
  loadFn: () => adminAPI.settings.getOverloadCooldownSettings(),
  saveFn: (data) => adminAPI.settings.updateOverloadCooldownSettings({
    enabled: data.enabled,
    cooldown_minutes: data.cooldown_minutes
  }),
  successMessage: t('admin.settings.overloadCooldown.saved'),
  errorMessage: t('admin.settings.overloadCooldown.saveFailed')
})

// 在 onMounted 中调用
onMounted(async () => {
  // ... 其他初始化代码
  await overloadCooldown.load()
  // ... 其他初始化代码
})
```

### 步骤 3: 重构其他独立设置项

按照相同的模式重构以下设置项：

#### 3.1 Rate Limit 429 Cooldown（第 307-412 行）

```typescript
const rateLimit429Cooldown = useSettingsCard({
  loadFn: () => adminAPI.settings.getRateLimit429CooldownSettings(),
  saveFn: (data) => adminAPI.settings.updateRateLimit429CooldownSettings(data),
  successMessage: t('admin.settings.rateLimit429Cooldown.saved'),
  errorMessage: t('admin.settings.rateLimit429Cooldown.saveFailed')
})
```

#### 3.2 Stream Timeout（第 414-592 行）

```typescript
const streamTimeout = useSettingsCard({
  loadFn: () => adminAPI.settings.getStreamTimeoutSettings(),
  saveFn: (data) => adminAPI.settings.updateStreamTimeoutSettings(data),
  successMessage: t('admin.settings.streamTimeout.saved'),
  errorMessage: t('admin.settings.streamTimeout.saveFailed')
})
```

#### 3.3 Request Rectifier（第 594-792 行）

```typescript
const rectifier = useSettingsCard({
  loadFn: () => adminAPI.settings.getRectifierSettings(),
  saveFn: (data) => adminAPI.settings.updateRectifierSettings(data),
  successMessage: t('admin.settings.rectifier.saved'),
  errorMessage: t('admin.settings.rectifier.saveFailed')
})
```

#### 3.4 Beta Policy（第 793-1070 行）

```typescript
const betaPolicy = useSettingsCard({
  loadFn: () => adminAPI.settings.getBetaPolicySettings(),
  saveFn: (data) => adminAPI.settings.updateBetaPolicySettings(data),
  successMessage: t('admin.settings.betaPolicy.saved'),
  errorMessage: t('admin.settings.betaPolicy.saveFailed')
})
```

### 步骤 4: 添加设置分组标题

在 Gateway 标签页中添加视觉分组：

```vue
<div v-show="activeTab === 'gateway'" class="space-y-6">
  <!-- Error Handling Group -->
  <div class="space-y-6">
    <div class="flex items-center gap-2 border-b border-gray-200 pb-2 dark:border-dark-600">
      <Icon name="exclamationTriangle" size="md" class="text-amber-500" />
      <h3 class="text-base font-semibold text-gray-900 dark:text-white">
        {{ t('admin.settings.gateway.errorHandling') }}
      </h3>
    </div>
    
    <!-- Overload Cooldown -->
    <SettingsCard ...>...</SettingsCard>
    
    <!-- Rate Limit 429 Cooldown -->
    <SettingsCard ...>...</SettingsCard>
    
    <!-- Stream Timeout -->
    <SettingsCard ...>...</SettingsCard>
  </div>

  <!-- Request Processing Group -->
  <div class="space-y-6">
    <div class="flex items-center gap-2 border-b border-gray-200 pb-2 dark:border-dark-600">
      <Icon name="cog" size="md" class="text-blue-500" />
      <h3 class="text-base font-semibold text-gray-900 dark:text-white">
        {{ t('admin.settings.gateway.requestProcessing') }}
      </h3>
    </div>
    
    <!-- Request Rectifier -->
    <SettingsCard ...>...</SettingsCard>
  </div>

  <!-- Policy Management Group -->
  <div class="space-y-6">
    <div class="flex items-center gap-2 border-b border-gray-200 pb-2 dark:border-dark-600">
      <Icon name="shield" size="md" class="text-violet-500" />
      <h3 class="text-base font-semibold text-gray-900 dark:text-white">
        {{ t('admin.settings.gateway.policyManagement') }}
      </h3>
    </div>
    
    <!-- Beta Policy -->
    <SettingsCard ...>...</SettingsCard>
    
    <!-- OpenAI Fast/Flex Policy -->
    <SettingsCard ...>...</SettingsCard>
  </div>

  <!-- Advanced Settings Group -->
  <div class="space-y-6">
    <div class="flex items-center gap-2 border-b border-gray-200 pb-2 dark:border-dark-600">
      <Icon name="adjustments" size="md" class="text-gray-500" />
      <h3 class="text-base font-semibold text-gray-900 dark:text-white">
        {{ t('admin.settings.gateway.advancedSettings') }}
      </h3>
    </div>
    
    <!-- Claude Code Settings -->
    <div class="card">...</div>
    
    <!-- Gateway Scheduling -->
    <div class="card">...</div>
    
    <!-- Gateway Forwarding -->
    <div class="card">...</div>
    
    <!-- Web Search Emulation -->
    <div class="card">...</div>
  </div>
</div>
```

### 步骤 5: 添加国际化文本

在 `frontend/src/i18n/locales/zh.ts` 和 `en.ts` 中添加：

```typescript
admin: {
  settings: {
    gateway: {
      errorHandling: '错误处理',
      requestProcessing: '请求处理',
      policyManagement: '策略管理',
      advancedSettings: '高级设置'
    }
  }
}
```

## 预期效果

### 代码减少

- **删除的代码行数**: 约 2000 行
  - 重复的加载状态 UI: ~600 行
  - 重复的保存按钮 UI: ~400 行
  - 重复的加载/保存方法: ~1000 行

- **新增的代码行数**: 约 200 行
  - SettingsCard 组件: 60 行
  - useSettingsCard composable: 60 行
  - 使用 composable 的代码: ~80 行

- **净减少**: 约 1800 行（17%）

### 可维护性提升

1. **统一的模式**: 所有独立设置项使用相同的结构
2. **更少的重复**: 加载/保存逻辑集中在 composable 中
3. **更容易测试**: composable 可以独立测试
4. **更好的类型安全**: TypeScript 类型推导更准确

### 用户体验改善

1. **一致的交互**: 所有设置项的加载和保存行为一致
2. **更清晰的组织**: 分组标题帮助用户快速找到相关设置
3. **更好的视觉层次**: 使用图标和颜色区分不同类型的设置

## 注意事项

1. **保持向后兼容**: 确保 API 调用参数和返回值格式不变
2. **测试所有设置项**: 重构后测试每个设置项的加载和保存功能
3. **检查错误处理**: 确保错误消息正确显示
4. **验证表单验证**: 确保输入验证规则仍然有效

## 完整示例

参考 `frontend/src/components/admin/settings/SettingsCard.vue` 和 `frontend/src/composables/useSettingsCard.ts` 的实现。

## 后续步骤

1. ✅ 创建 SettingsCard 组件
2. ✅ 创建 useSettingsCard composable
3. ⏳ 重构 Overload Cooldown 设置项
4. ⏳ 重构 Rate Limit 429 Cooldown 设置项
5. ⏳ 重构 Stream Timeout 设置项
6. ⏳ 重构 Request Rectifier 设置项
7. ⏳ 重构 Beta Policy 设置项
8. ⏳ 添加设置分组标题
9. ⏳ 测试所有设置项
10. ⏳ 更新文档
