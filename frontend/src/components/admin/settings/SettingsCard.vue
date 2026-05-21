<!--
  SettingsCard 组件

  标准化的设置卡片容器组件，提供统一的加载状态、保存按钮和布局。

  Props:
  - title: 卡片标题
  - description: 卡片描述文本（可选）
  - loading: 是否显示加载状态（可选）
  - saving: 是否正在保存（可选）
  - showSaveButton: 是否显示保存按钮（可选）

  Events:
  - save: 点击保存按钮时触发

  Slots:
  - default: 卡片内容区域

  使用示例:
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
-->
<template>
  <div class="card">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
        {{ title }}
      </h2>
      <p v-if="description" class="mt-1 text-sm text-gray-500 dark:text-gray-400">
        {{ description }}
      </p>
    </div>
    <div class="space-y-5 p-6">
      <!-- Loading State -->
      <div v-if="loading" class="flex items-center gap-2 text-gray-500">
        <div class="h-4 w-4 animate-spin rounded-full border-b-2 border-primary-600"></div>
        {{ t('common.loading') }}
      </div>

      <!-- Content -->
      <template v-else>
        <slot />

        <!-- Save Button -->
        <div
          v-if="showSaveButton"
          class="flex justify-end border-t border-gray-100 pt-4 dark:border-dark-700"
        >
          <button
            type="button"
            @click="$emit('save')"
            :disabled="saving"
            class="btn btn-primary btn-sm"
          >
            <svg
              v-if="saving"
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
            {{ saving ? t('common.saving') : t('common.save') }}
          </button>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineProps<{
  title: string
  description?: string
  loading?: boolean
  saving?: boolean
  showSaveButton?: boolean
}>()

defineEmits<{
  save: []
}>()

const { t } = useI18n()
</script>
