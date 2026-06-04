<!--
  ProxyBulkActionsBar 组件

  代理批量操作工具栏，提供测试、质量检查、分配、删除等批量操作。

  Props:
  - selectedCount: 已选中的代理数量
  - batchTesting: 是否正在批量测试（可选）
  - batchQualityChecking: 是否正在批量质量检查（可选）

  Events:
  - test: 批量测试连接
  - quality-check: 批量质量检查
  - enable-pool: 批量启用代理池
  - disable-pool: 批量禁用代理池
  - clear-cooldown: 批量清除冷却时间
  - assign: 分配账户
  - unassign: 取消分配账户
  - delete: 批量删除
  - clear: 清除选择

  特性:
  - 使用 sticky 定位，固定在顶部
  - 下拉菜单自动处理点击外部关闭
  - 所有操作按钮在执行时显示加载状态

  使用示例:
  <ProxyBulkActionsBar
    :selected-count="selectedProxies.length"
    :batch-testing="batchTesting"
    :batch-quality-checking="batchQualityChecking"
    @test="handleBatchTest"
    @quality-check="handleBatchQualityCheck"
    @delete="handleDelete"
    @clear="clearSelection"
  />
-->
<template>
  <div
    class="sticky top-0 z-10 flex items-center justify-between gap-3 border-b border-gray-200 bg-white px-4 py-3 dark:border-dark-600 dark:bg-dark-800"
  >
    <div class="flex items-center gap-3">
      <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
        {{ t('admin.proxies.selectedCount', { count: selectedCount }) }}
      </span>
      <button
        @click="$emit('clear')"
        class="text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
      >
        {{ t('common.clearSelection') }}
      </button>
    </div>

    <div class="flex flex-wrap items-center gap-2">
      <!-- Test Connection -->
      <button
        @click="$emit('test')"
        :disabled="batchTesting"
        class="btn btn-sm btn-secondary"
      >
        <svg
          v-if="batchTesting"
          class="mr-1 h-4 w-4 animate-spin"
          fill="none"
          viewBox="0 0 24 24"
        >
          <circle
            class="opacity-25"
            cx="12"
            cy="12"
            r="10"
            stroke="currentColor"
            stroke-width="4"
          ></circle>
          <path
            class="opacity-75"
            fill="currentColor"
            d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
          ></path>
        </svg>
        <Icon v-else name="checkCircle" size="sm" class="mr-1" />
        <span class="hidden sm:inline">{{ t('admin.proxies.batchTest') }}</span>
      </button>

      <!-- Quality Check -->
      <button
        @click="$emit('quality-check')"
        :disabled="batchQualityChecking"
        class="btn btn-sm btn-secondary"
      >
        <svg
          v-if="batchQualityChecking"
          class="mr-1 h-4 w-4 animate-spin"
          fill="none"
          viewBox="0 0 24 24"
        >
          <circle
            class="opacity-25"
            cx="12"
            cy="12"
            r="10"
            stroke="currentColor"
            stroke-width="4"
          ></circle>
          <path
            class="opacity-75"
            fill="currentColor"
            d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
          ></path>
        </svg>
        <Icon v-else name="shield" size="sm" class="mr-1" />
        <span class="hidden sm:inline">{{ t('admin.proxies.batchQualityCheck') }}</span>
      </button>

      <!-- More Actions Dropdown -->
      <div class="relative">
        <button
          @click="showDropdown = !showDropdown"
          class="btn btn-sm btn-secondary"
        >
          <Icon name="more" size="sm" class="mr-1" />
          <span class="hidden sm:inline">{{ t('common.more') }}</span>
        </button>

        <div
          v-if="showDropdown"
          class="absolute right-0 top-full z-50 mt-1 w-48 overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-600 dark:bg-dark-800"
          @click.stop
        >
          <div class="py-1">
            <button
              @click="handleAction('enable-pool')"
              class="flex w-full items-center gap-2 px-4 py-2 text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700"
            >
              <Icon name="plus" size="sm" class="text-gray-500" />
              {{ t('admin.proxies.poolBatchEnable') }}
            </button>
            <button
              @click="handleAction('disable-pool')"
              class="flex w-full items-center gap-2 px-4 py-2 text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700"
            >
              <Icon name="x" size="sm" class="text-gray-500" />
              {{ t('admin.proxies.poolBatchDisable') }}
            </button>
            <button
              @click="handleAction('clear-cooldown')"
              class="flex w-full items-center gap-2 px-4 py-2 text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700"
            >
              <Icon name="refresh" size="sm" class="text-amber-500" />
              {{ t('admin.proxies.clearCooldownBatch') }}
            </button>
            <div class="my-1 border-t border-gray-100 dark:border-dark-700"></div>
            <button
              @click="handleAction('assign')"
              class="flex w-full items-center gap-2 px-4 py-2 text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700"
            >
              <Icon name="link" size="sm" class="text-gray-700" />
              {{ t('admin.proxies.assignAccounts') }}
            </button>
            <button
              @click="handleAction('unassign')"
              class="flex w-full items-center gap-2 px-4 py-2 text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700"
            >
              <Icon name="xCircle" size="sm" class="text-gray-500" />
              {{ t('admin.proxies.quickUnassign') }}
            </button>
            <div class="my-1 border-t border-gray-100 dark:border-dark-700"></div>
            <button
              @click="handleAction('delete')"
              class="flex w-full items-center gap-2 px-4 py-2 text-sm text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/20"
            >
              <Icon name="trash" size="sm" />
              {{ t('common.delete') }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

defineProps<{
  selectedCount: number
  batchTesting?: boolean
  batchQualityChecking?: boolean
}>()

const emit = defineEmits<{
  test: []
  'quality-check': []
  'enable-pool': []
  'disable-pool': []
  'clear-cooldown': []
  assign: []
  unassign: []
  delete: []
  clear: []
}>()

const { t } = useI18n()
const showDropdown = ref(false)

function handleAction(action: string) {
  showDropdown.value = false
  emit(action as any)
}

function handleClickOutside(event: MouseEvent) {
  const target = event.target as HTMLElement
  if (!target.closest('.relative')) {
    showDropdown.value = false
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})
</script>
