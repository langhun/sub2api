# 组件和工具文档

本文档列出了项目中所有可复用的组件和 composables，提供快速参考和使用指南。

## 目录

- [组件](#组件)
  - [SettingsCard](#settingscard)
  - [ProxyBulkActionsBar](#proxybulkactionsbar)
- [Composables](#composables)
  - [useSettingsCard](#usesettingscard)
  - [useAccountBulkOperations](#useaccountbulkoperations)
  - [useProxyTesting](#useproxytesting)
- [工具函数](#工具函数)
  - [accountFilters](#accountfilters)

---

## 组件

### SettingsCard

**路径**: `src/components/admin/settings/SettingsCard.vue`

**功能**: 标准化的设置卡片容器组件，提供统一的加载状态、保存按钮和布局。

**使用场景**:
- 系统设置页面
- 配置管理界面
- 任何需要加载/保存数据的表单容器

**示例**:
```vue
<SettingsCard
  :title="t('admin.settings.general')"
  :description="t('admin.settings.generalDesc')"
  :loading="loading"
  :saving="saving"
  :show-save-button="true"
  @save="handleSave"
>
  <div class="space-y-4">
    <!-- 你的表单内容 -->
  </div>
</SettingsCard>
```

**详细文档**: 见 [SettingsCard 组件文档](#settingscard-详细文档)

---

### ProxyBulkActionsBar

**路径**: `src/components/admin/proxy/ProxyBulkActionsBar.vue`

**功能**: 代理批量操作工具栏，提供测试、质量检查、分配、删除等批量操作。

**使用场景**:
- 代理管理页面
- 需要批量操作的列表视图

**示例**:
```vue
<ProxyBulkActionsBar
  :selected-count="selectedProxies.length"
  :batch-testing="batchTesting"
  :batch-quality-checking="batchQualityChecking"
  @test="handleBatchTest"
  @quality-check="handleBatchQualityCheck"
  @enable-pool="handleEnablePool"
  @disable-pool="handleDisablePool"
  @clear-cooldown="handleClearCooldown"
  @assign="handleAssign"
  @unassign="handleUnassign"
  @delete="handleDelete"
  @clear="clearSelection"
/>
```

**详细文档**: 见 [ProxyBulkActionsBar 组件文档](#proxybulkactionsbar-详细文档)

---

## Composables

### useSettingsCard

**路径**: `src/composables/useSettingsCard.ts`

**功能**: 简化设置卡片的加载和保存逻辑，提供统一的状态管理。

**使用场景**:
- 配合 SettingsCard 组件使用
- 任何需要加载/保存配置的场景

**示例**:
```typescript
const { loading, saving, form, load, save } = useSettingsCard({
  loadFn: () => adminAPI.settings.getGeneral(),
  saveFn: (data) => adminAPI.settings.updateGeneral(data),
  successMessage: t('admin.settings.saveSuccess'),
  errorMessage: t('admin.settings.saveFailed')
})

onMounted(() => load())
```

**详细文档**: 见 [useSettingsCard 详细文档](#usesettingscard-详细文档)

---

### useAccountBulkOperations

**路径**: `src/composables/useAccountBulkOperations.ts`

**功能**: 处理账户批量操作的通用逻辑，包括确认、错误处理和结果反馈。

**使用场景**:
- 账户批量启用/禁用
- 批量删除
- 批量更新配置

**示例**:
```typescript
const { operating, handleBulkOperation } = useAccountBulkOperations()

async function handleBatchDelete() {
  await handleBulkOperation(
    () => adminAPI.accounts.batchDelete(selectedIds.value),
    {
      confirmMessage: t('admin.accounts.confirmDelete'),
      successMessage: (result) => 
        t('admin.accounts.deleteSuccess', { count: result.success }),
      errorMessage: (result) => 
        t('admin.accounts.deleteFailed', { count: result.failed })
    },
    {
      onSuccess: () => {
        clearSelection()
        loadAccounts()
      }
    }
  )
}
```

**详细文档**: 见 [useAccountBulkOperations 详细文档](#useaccountbulkoperations-详细文档)

---

### useProxyTesting

**路径**: `src/composables/useProxyTesting.ts`

**功能**: 代理测试和质量检查的核心逻辑，支持单个和批量操作，带并发控制。

**使用场景**:
- 代理连接测试
- 代理质量检查
- 批量代理验证

**示例**:
```typescript
const {
  testingProxyIds,
  qualityCheckingProxyIds,
  testSingleProxy,
  testMultipleProxies,
  checkSingleProxyQuality,
  checkMultipleProxiesQuality
} = useProxyTesting()

// 单个测试
const result = await testSingleProxy(proxyId)

// 批量测试（并发控制）
await testMultipleProxies(selectedIds.value, 5)

// 质量检查
const summary = await checkMultipleProxiesQuality(selectedIds.value, 3)
console.log(`健康: ${summary.healthy}, 警告: ${summary.warn}`)
```

**详细文档**: 见 [useProxyTesting 详细文档](#useproxytesting-详细文档)

---

## 工具函数

### accountFilters

**路径**: `src/utils/accountFilters.ts`

**功能**: 账户过滤逻辑集合，支持平台、类型、状态、分组、隐私模式等多维度过滤。

**使用场景**:
- 账户列表过滤
- 搜索功能
- 条件筛选

**示例**:
```typescript
import {
  matchesPlatform,
  matchesType,
  matchesStatus,
  matchesTier,
  matchesGroup,
  matchesSearch
} from '@/utils/accountFilters'

const filtered = accounts.filter(account => {
  return matchesPlatform(account, 'openai') &&
         matchesType(account, 'session') &&
         matchesStatus(account, 'active', '', '', { nowMs: Date.now() }) &&
         matchesTier(account, 'plus', 'openai') &&
         matchesSearch(account, searchTerm)
})
```

**详细文档**: 见 [accountFilters 详细文档](#accountfilters-详细文档)

---

## 详细文档

### SettingsCard 详细文档

#### Props

| 属性 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| title | string | 是 | - | 卡片标题 |
| description | string | 否 | - | 卡片描述文本 |
| loading | boolean | 否 | false | 是否显示加载状态 |
| saving | boolean | 否 | false | 是否正在保存 |
| showSaveButton | boolean | 否 | false | 是否显示保存按钮 |

#### Events

| 事件名 | 参数 | 说明 |
|--------|------|------|
| save | - | 点击保存按钮时触发 |

#### Slots

| 插槽名 | 说明 |
|--------|------|
| default | 卡片内容区域 |

#### 完整示例

```vue
<script setup lang="ts">
import { onMounted } from 'vue'
import SettingsCard from '@/components/admin/settings/SettingsCard.vue'
import { useSettingsCard } from '@/composables/useSettingsCard'
import { adminAPI } from '@/api/admin'

const { loading, saving, form, load, save } = useSettingsCard({
  loadFn: () => adminAPI.settings.getGeneral(),
  saveFn: (data) => adminAPI.settings.updateGeneral(data)
})

onMounted(() => load())
</script>

<template>
  <SettingsCard
    :title="$t('admin.settings.general')"
    :description="$t('admin.settings.generalDesc')"
    :loading="loading"
    :saving="saving"
    :show-save-button="true"
    @save="save"
  >
    <div class="space-y-4">
      <div>
        <label class="label">{{ $t('admin.settings.siteName') }}</label>
        <input v-model="form.site_name" type="text" class="input" />
      </div>
      <div>
        <label class="label">{{ $t('admin.settings.siteUrl') }}</label>
        <input v-model="form.site_url" type="url" class="input" />
      </div>
    </div>
  </SettingsCard>
</template>
```

---

### ProxyBulkActionsBar 详细文档

#### Props

| 属性 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| selectedCount | number | 是 | - | 已选中的代理数量 |
| batchTesting | boolean | 否 | false | 是否正在批量测试 |
| batchQualityChecking | boolean | 否 | false | 是否正在批量质量检查 |

#### Events

| 事件名 | 说明 |
|--------|------|
| test | 批量测试连接 |
| quality-check | 批量质量检查 |
| enable-pool | 批量启用代理池 |
| disable-pool | 批量禁用代理池 |
| clear-cooldown | 批量清除冷却时间 |
| assign | 分配账户 |
| unassign | 取消分配账户 |
| delete | 批量删除 |
| clear | 清除选择 |

#### 完整示例

```vue
<script setup lang="ts">
import { ref } from 'vue'
import ProxyBulkActionsBar from '@/components/admin/proxy/ProxyBulkActionsBar.vue'
import { useProxyTesting } from '@/composables/useProxyTesting'
import { adminAPI } from '@/api/admin'

const selectedProxies = ref<number[]>([])
const batchTesting = ref(false)
const batchQualityChecking = ref(false)

const { testMultipleProxies, checkMultipleProxiesQuality } = useProxyTesting()

async function handleBatchTest() {
  batchTesting.value = true
  try {
    await testMultipleProxies(selectedProxies.value, 5)
  } finally {
    batchTesting.value = false
  }
}

async function handleBatchQualityCheck() {
  batchQualityChecking.value = true
  try {
    const summary = await checkMultipleProxiesQuality(selectedProxies.value, 3)
    console.log('Quality check summary:', summary)
  } finally {
    batchQualityChecking.value = false
  }
}
</script>

<template>
  <ProxyBulkActionsBar
    v-if="selectedProxies.length > 0"
    :selected-count="selectedProxies.length"
    :batch-testing="batchTesting"
    :batch-quality-checking="batchQualityChecking"
    @test="handleBatchTest"
    @quality-check="handleBatchQualityCheck"
    @clear="selectedProxies = []"
  />
</template>
```

#### 注意事项

- 组件使用 sticky 定位，会固定在顶部
- 下拉菜单会自动处理点击外部关闭
- 所有操作按钮在执行时会显示加载状态



### useSettingsCard 详细文档

#### 类型定义

```typescript
interface SettingsCardOptions<T> {
  loadFn: () => Promise<T>
  saveFn: (data: T) => Promise<void>
  successMessage?: string
  errorMessage?: string
}
```

#### 参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| options | SettingsCardOptions<T> | 是 | 配置选项 |
| options.loadFn | () => Promise<T> | 是 | 加载数据的函数 |
| options.saveFn | (data: T) => Promise<void> | 是 | 保存数据的函数 |
| options.successMessage | string | 否 | 保存成功提示 |
| options.errorMessage | string | 否 | 错误提示 |

#### 返回值

| 属性 | 类型 | 说明 |
|------|------|------|
| loading | Ref<boolean> | 是否正在加载 |
| saving | Ref<boolean> | 是否正在保存 |
| form | Reactive<T> | 表单数据（响应式） |
| load | () => Promise<void> | 加载数据方法 |
| save | () => Promise<void> | 保存数据方法 |

#### 特性

- 自动错误处理和提示
- 统一的加载/保存状态管理
- 响应式表单数据
- 类型安全

#### 完整示例

```typescript
import { onMounted } from 'vue'
import { useSettingsCard } from '@/composables/useSettingsCard'
import { adminAPI } from '@/api/admin'
import { useI18n } from 'vue-i18n'

interface GeneralSettings {
  site_name: string
  site_url: string
  enable_registration: boolean
}

const { t } = useI18n()

const { loading, saving, form, load, save } = useSettingsCard<GeneralSettings>({
  loadFn: async () => {
    const response = await adminAPI.settings.getGeneral()
    return response.data
  },
  saveFn: async (data) => {
    await adminAPI.settings.updateGeneral(data)
  },
  successMessage: t('admin.settings.saveSuccess'),
  errorMessage: t('admin.settings.saveFailed')
})

onMounted(() => load())
```


### useAccountBulkOperations 详细文档

#### 类型定义

```typescript
interface BulkOperationResult {
  success: number
  failed: number
  skipped?: number
  success_ids?: number[]
  failed_ids?: number[]
}

interface BulkOperationOptions {
  confirmMessage: string
  successMessage: string | ((result: BulkOperationResult) => string)
  errorMessage?: string | ((result: BulkOperationResult) => string)
  skipConfirm?: boolean
}
```

#### 返回值

| 属性 | 类型 | 说明 |
|------|------|------|
| operating | Ref<boolean> | 是否正在执行操作 |
| handleBulkOperation | Function | 执行批量操作的方法 |

#### handleBulkOperation 参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| operation | () => Promise<T> | 是 | 要执行的批量操作函数 |
| options | BulkOperationOptions | 是 | 操作配置 |
| callbacks | Object | 否 | 回调函数 |
| callbacks.onSuccess | (result: T) => void | 否 | 成功回调 |
| callbacks.onError | (error: any) => void | 否 | 错误回调 |
| callbacks.onFinally | () => void | 否 | 完成回调 |

#### 特性

- 自动确认对话框（可选）
- 部分成功处理（区分成功和失败的项）
- 统一的错误处理和提示
- 支持动态消息（函数形式）
- 回调钩子支持

#### 完整示例

```typescript
import { useAccountBulkOperations } from '@/composables/useAccountBulkOperations'
import { adminAPI } from '@/api/admin'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const { operating, handleBulkOperation } = useAccountBulkOperations()

// 批量启用账户
async function handleBatchEnable() {
  const result = await handleBulkOperation(
    () => adminAPI.accounts.batchEnable(selectedIds.value),
    {
      confirmMessage: t('admin.accounts.confirmEnable', { count: selectedIds.value.length }),
      successMessage: (result) => 
        t('admin.accounts.enableSuccess', { count: result.success }),
      errorMessage: (result) => 
        t('admin.accounts.enablePartialFailed', { 
          success: result.success, 
          failed: result.failed 
        })
    },
    {
      onSuccess: (result) => {
        console.log('Enabled accounts:', result.success_ids)
        clearSelection()
        loadAccounts()
      },
      onError: (error) => {
        console.error('Batch enable failed:', error)
      }
    }
  )
}

// 跳过确认的操作
async function handleQuickAction() {
  await handleBulkOperation(
    () => adminAPI.accounts.quickAction(selectedIds.value),
    {
      confirmMessage: '', // 不会使用
      successMessage: t('common.success'),
      skipConfirm: true
    }
  )
}
```

#### 注意事项

- 如果用户取消确认，返回 null
- 部分成功时会显示错误消息而非成功消息
- 操作期间 operating 为 true，可用于禁用按钮


### useProxyTesting 详细文档

#### 类型定义

```typescript
interface ProxyTestResult {
  success: boolean
  latency_ms?: number
  message?: string
  ip_address?: string
  country?: string
  country_code?: string
  region?: string
  city?: string
}

interface ProxyQualityCheckResult {
  challenge_count: number
  failed_count: number
  warn_count: number
  // ... 其他字段
}

interface QualityCheckSummary {
  total: number
  healthy: number
  warn: number
  challenge: number
  failed: number
}
```

#### 返回值

| 属性 | 类型 | 说明 |
|------|------|------|
| testingProxyIds | Ref<Set<number>> | 正在测试的代理 ID 集合 |
| qualityCheckingProxyIds | Ref<Set<number>> | 正在质量检查的代理 ID 集合 |
| testSingleProxy | (proxyId: number) => Promise<ProxyTestResult \| null> | 测试单个代理 |
| testMultipleProxies | (ids: number[], concurrency?: number) => Promise<void> | 批量测试代理 |
| checkSingleProxyQuality | (proxyId: number) => Promise<ProxyQualityCheckResult \| null> | 检查单个代理质量 |
| checkMultipleProxiesQuality | (ids: number[], concurrency?: number) => Promise<QualityCheckSummary> | 批量质量检查 |
| summarizeQualityStatus | (result: ProxyQualityCheckResult) => Proxy['quality_status'] | 汇总质量状态 |

#### 方法详解

##### testSingleProxy

测试单个代理的连接性。

```typescript
const result = await testSingleProxy(123)
if (result?.success) {
  console.log(`延迟: ${result.latency_ms}ms`)
  console.log(`IP: ${result.ip_address}`)
  console.log(`位置: ${result.country} ${result.city}`)
}
```

##### testMultipleProxies

批量测试多个代理，支持并发控制。

```typescript
// 使用默认并发数 5
await testMultipleProxies([1, 2, 3, 4, 5])

// 自定义并发数
await testMultipleProxies(selectedIds.value, 10)
```

##### checkSingleProxyQuality

检查单个代理的质量（检测是否被封禁、需要验证码等）。

```typescript
const result = await checkSingleProxyQuality(123)
if (result) {
  const status = summarizeQualityStatus(result)
  console.log(`质量状态: ${status}`) // 'healthy' | 'warn' | 'challenge' | 'failed'
}
```

##### checkMultipleProxiesQuality

批量质量检查，返回汇总统计。

```typescript
const summary = await checkMultipleProxiesQuality(selectedIds.value, 3)
console.log(`总计: ${summary.total}`)
console.log(`健康: ${summary.healthy}`)
console.log(`警告: ${summary.warn}`)
console.log(`需验证: ${summary.challenge}`)
console.log(`失败: ${summary.failed}`)
```

#### 完整示例

```vue
<script setup lang="ts">
import { ref, computed } from 'vue'
import { useProxyTesting } from '@/composables/useProxyTesting'

const selectedProxies = ref<number[]>([1, 2, 3, 4, 5])

const {
  testingProxyIds,
  qualityCheckingProxyIds,
  testSingleProxy,
  testMultipleProxies,
  checkMultipleProxiesQuality
} = useProxyTesting()

// 检查某个代理是否正在测试
const isProxyTesting = (id: number) => testingProxyIds.value.has(id)

// 批量测试
async function handleBatchTest() {
  await testMultipleProxies(selectedProxies.value, 5)
  console.log('所有测试完成')
}

// 批量质量检查
async function handleBatchQualityCheck() {
  const summary = await checkMultipleProxiesQuality(selectedProxies.value, 3)
  
  if (summary.failed > 0) {
    alert(`发现 ${summary.failed} 个失败的代理`)
  }
}

// 单个测试
async function testProxy(id: number) {
  const result = await testSingleProxy(id)
  if (result?.success) {
    console.log(`代理 ${id} 测试成功，延迟 ${result.latency_ms}ms`)
  } else {
    console.log(`代理 ${id} 测试失败: ${result?.message}`)
  }
}
</script>

<template>
  <div>
    <button @click="handleBatchTest">批量测试</button>
    <button @click="handleBatchQualityCheck">批量质量检查</button>
    
    <div v-for="id in selectedProxies" :key="id">
      <span>代理 {{ id }}</span>
      <span v-if="isProxyTesting(id)">测试中...</span>
      <button @click="testProxy(id)" :disabled="isProxyTesting(id)">
        测试
      </button>
    </div>
  </div>
</template>
```

#### 注意事项

- 并发控制：默认测试并发为 5，质量检查并发为 3
- 测试状态：通过 testingProxyIds 和 qualityCheckingProxyIds 跟踪正在进行的操作
- 错误处理：单个代理失败不会影响其他代理的测试
- 性能考虑：大量代理测试时建议适当调整并发数


### accountFilters 详细文档

#### 导出的函数

| 函数名 | 参数 | 返回值 | 说明 |
|--------|------|--------|------|
| matchesPlatform | (account, platform) | boolean | 匹配平台 |
| matchesType | (account, type) | boolean | 匹配账户类型 |
| matchesStatus | (account, mainStatus, runtimeStatus, schedulingStatus, options) | boolean | 匹配状态 |
| matchesTier | (account, selectedTier, fallbackPlatform) | boolean | 匹配套餐等级 |
| matchesGroup | (account, group) | boolean | 匹配分组 |
| matchesPrivacyMode | (account, privacyMode) | boolean | 匹配隐私模式 |
| matchesSearch | (account, search) | boolean | 匹配搜索关键词 |
| getAntigravityTierFromAccount | (account) | string \| null | 获取 Antigravity 账户的套餐等级 |

#### 类型定义

```typescript
interface AccountFilterOptions {
  platform?: string
  tier?: string
  type?: string
  main_status?: string
  runtime_status?: string
  scheduling_status?: string
  privacy_mode?: string
  group?: string
  search?: string
}

interface StatusEvalOptions {
  nowMs: number
}
```

#### 常量

```typescript
// 未分组的特殊值
export const ACCOUNT_UNGROUPED_GROUP_QUERY_VALUE = 'ungrouped'

// 隐私模式未设置的特殊值
export const ACCOUNT_PRIVACY_MODE_UNSET_QUERY_VALUE = '__unset__'
```

#### 函数详解

##### matchesPlatform

匹配账户平台。

```typescript
matchesPlatform(account, 'openai') // true if account.platform === 'openai'
matchesPlatform(account, '') // true (空字符串匹配所有)
```

##### matchesType

匹配账户类型。

```typescript
matchesType(account, 'session') // true if account.type === 'session'
matchesType(account, 'api_key') // true if account.type === 'api_key'
```

##### matchesStatus

匹配账户状态（主状态、运行时状态、调度状态）。

```typescript
matchesStatus(
  account,
  'active',           // 主状态
  'running',          // 运行时状态
  'scheduled',        // 调度状态
  { nowMs: Date.now() }
)
```

##### matchesTier

匹配套餐等级，支持平台前缀。

```typescript
// 基本用法
matchesTier(account, 'plus', 'openai')

// 带平台前缀
matchesTier(account, 'openai:plus', '')

// OpenAI 套餐别名
matchesTier(account, 'chatgpt_plus', 'openai') // 等同于 'plus'
matchesTier(account, 'plus_plan', 'openai')    // 等同于 'plus'

// Gemini 套餐别名
matchesTier(account, 'ai_premium', 'gemini')   // 等同于 'google_ai_pro'
matchesTier(account, 'pro', 'gemini')          // 等同于 'google_ai_pro'
```

##### matchesGroup

匹配账户分组。

```typescript
// 匹配特定分组
matchesGroup(account, '123') // 分组 ID 为 123

// 匹配未分组账户
matchesGroup(account, ACCOUNT_UNGROUPED_GROUP_QUERY_VALUE)
```

##### matchesPrivacyMode

匹配隐私模式。

```typescript
// 匹配特定隐私模式
matchesPrivacyMode(account, 'strict')

// 匹配未设置隐私模式的账户
matchesPrivacyMode(account, ACCOUNT_PRIVACY_MODE_UNSET_QUERY_VALUE)
```

##### matchesSearch

匹配搜索关键词（在账户名称中搜索）。

```typescript
matchesSearch(account, 'test') // 账户名称包含 'test'（不区分大小写）
```

#### 完整示例

```typescript
import {
  matchesPlatform,
  matchesType,
  matchesStatus,
  matchesTier,
  matchesGroup,
  matchesPrivacyMode,
  matchesSearch,
  ACCOUNT_UNGROUPED_GROUP_QUERY_VALUE,
  type AccountFilterOptions
} from '@/utils/accountFilters'

// 过滤器配置
const filters: AccountFilterOptions = {
  platform: 'openai',
  type: 'session',
  main_status: 'active',
  runtime_status: '',
  scheduling_status: '',
  tier: 'plus',
  group: '',
  privacy_mode: '',
  search: 'test'
}

// 应用过滤
const filteredAccounts = accounts.filter(account => {
  const statusOptions = { nowMs: Date.now() }
  
  return (
    matchesPlatform(account, filters.platform || '') &&
    matchesType(account, filters.type || '') &&
    matchesStatus(
      account,
      filters.main_status || '',
      filters.runtime_status || '',
      filters.scheduling_status || '',
      statusOptions
    ) &&
    matchesTier(account, filters.tier || '', filters.platform || '') &&
    matchesGroup(account, filters.group || '') &&
    matchesPrivacyMode(account, filters.privacy_mode || '') &&
    matchesSearch(account, filters.search || '')
  )
})

// 查找未分组的 OpenAI Plus 账户
const ungroupedPlusAccounts = accounts.filter(account => {
  return (
    matchesPlatform(account, 'openai') &&
    matchesTier(account, 'plus', 'openai') &&
    matchesGroup(account, ACCOUNT_UNGROUPED_GROUP_QUERY_VALUE)
  )
})
```

#### 套餐等级别名

##### OpenAI

| 别名 | 标准值 |
|------|--------|
| free, free_plan, chatgpt_free | free |
| plus, plus_plan, chatgpt_plus | plus |
| team, team_plan, chatgpt_team, business | team |
| pro, pro_plan, chatgpt_pro | pro |
| enterprise, enterprise_plan, chatgpt_enterprise | enterprise |

##### Gemini

| 别名 | 标准值 |
|------|--------|
| free, google_one_basic, google_one_standard | google_one_free |
| ai_premium, pro | google_ai_pro |
| ultra | google_ai_ultra |

##### Antigravity

| 别名 | 标准值 |
|------|--------|
| free, free_tier | free-tier |
| pro, g1_pro_tier | g1-pro-tier |
| ultra, g1_ultra_tier | g1-ultra-tier |

#### 注意事项

- 所有过滤函数对空字符串或 undefined 返回 true（表示不过滤）
- 套餐等级匹配会自动处理别名和大小写
- 搜索功能不区分大小写
- 状态匹配需要提供当前时间戳用于时间相关的状态判断

---

## 最佳实践

### 组件使用

1. **SettingsCard + useSettingsCard 组合**
   - 始终在 onMounted 中调用 load()
   - 使用 TypeScript 定义表单数据类型
   - 提供清晰的成功/错误消息

2. **ProxyBulkActionsBar**
   - 配合 useProxyTesting 使用
   - 维护独立的 loading 状态
   - 操作完成后刷新列表

### Composables 使用

1. **useAccountBulkOperations**
   - 使用动态消息函数提供详细反馈
   - 在 onSuccess 回调中清理选择和刷新数据
   - 对危险操作保留确认对话框

2. **useProxyTesting**
   - 根据代理数量调整并发数
   - 使用 testingProxyIds 显示加载状态
   - 批量操作后显示汇总结果

3. **useSettingsCard**
   - 为不同设置页面创建独立的类型定义
   - 在 loadFn 中处理数据转换
   - 在 saveFn 中进行数据验证

### 工具函数使用

1. **accountFilters**
   - 组合多个过滤函数实现复杂过滤
   - 使用常量而非硬编码特殊值
   - 缓存 nowMs 避免重复计算

---

## 更新日志

- 2026-05-21: 初始文档创建
  - 添加 SettingsCard 组件文档
  - 添加 ProxyBulkActionsBar 组件文档
  - 添加 useSettingsCard composable 文档
  - 添加 useAccountBulkOperations composable 文档
  - 添加 useProxyTesting composable 文档
  - 添加 accountFilters 工具函数文档

