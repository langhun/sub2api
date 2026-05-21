<template>
  <BaseDialog
    :show="show"
    :title="editing ? t('admin.proxies.subscriptions.editTitle') : t('admin.proxies.subscriptions.createTitle')"
    width="normal"
    @close="$emit('close')"
  >
    <form id="create-subscription-form" class="space-y-4" @submit.prevent="$emit('submit')">
      <div>
        <label class="input-label">{{ t('admin.proxies.subscriptions.fields.name') }}</label>
        <input :value="form.name" type="text" class="input" required @input="updateTextField('name', $event)" />
      </div>
      <div>
        <label class="input-label">{{ t('admin.proxies.subscriptions.fields.url') }}</label>
        <input :value="form.url" type="url" class="input" required @input="updateTextField('url', $event)" />
      </div>
      <div>
        <label class="input-label">{{ t('admin.proxies.subscriptions.fields.format') }}</label>
        <Select
          :model-value="form.source_format"
          :options="formatOptions"
          @update:model-value="updateField('source_format', String($event))"
        />
      </div>
      <div class="grid grid-cols-2 gap-4">
        <div>
          <label class="input-label">{{ t('admin.proxies.subscriptions.fields.refreshIntervalHours') }}</label>
          <input
            :value="form.refresh_interval_hours"
            type="number"
            min="1"
            class="input"
            @input="updateNumberField('refresh_interval_hours', $event)"
          />
        </div>
        <div>
          <label class="input-label">{{ t('admin.proxies.subscriptions.fields.targetEntryCount') }}</label>
          <input
            :value="form.target_entry_count"
            type="number"
            min="1"
            max="10"
            class="input"
            @input="updateNumberField('target_entry_count', $event)"
          />
        </div>
      </div>
      <div class="grid grid-cols-2 gap-4">
        <label class="flex items-center gap-2 pt-8 text-sm text-gray-700 dark:text-gray-300">
          <input
            :checked="form.enabled"
            type="checkbox"
            class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
            @change="updateCheckboxField('enabled', $event)"
          />
          {{ t('common.enabled') }}
        </label>
      </div>
      <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
        <input
          :checked="form.auto_add_to_pool"
          type="checkbox"
          class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
          @change="updateCheckboxField('auto_add_to_pool', $event)"
        />
        {{ t('admin.proxies.subscriptions.fields.autoAddToPool') }}
      </label>
    </form>
    <template #footer>
      <div class="flex justify-end gap-3">
        <button class="btn btn-secondary" type="button" @click="$emit('close')">{{ t('common.cancel') }}</button>
        <button class="btn btn-primary" type="submit" form="create-subscription-form" :disabled="submitting">
          {{ submitting ? t('common.submitting') : editing ? t('common.save') : t('common.create') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import type { ProxySubscriptionSource } from '@/types'

const { t } = useI18n()

const props = defineProps<{
  show: boolean
  editing: boolean
  submitting: boolean
  form: {
    name: string
    url: string
    source_format: ProxySubscriptionSource['source_format']
    enabled: boolean
    refresh_interval_hours: number
    target_entry_count: number
    auto_add_to_pool: boolean
  }
  formatOptions: Array<{ value: ProxySubscriptionSource['source_format']; label: string }>
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'submit'): void
  (e: 'update:form', value: {
    name: string
    url: string
    source_format: ProxySubscriptionSource['source_format']
    enabled: boolean
    refresh_interval_hours: number
    target_entry_count: number
    auto_add_to_pool: boolean
  }): void
}>()

const updateField = (
  key: keyof typeof props.form,
  value: string | number | boolean | ProxySubscriptionSource['source_format']
) => {
  emit('update:form', {
    ...props.form,
    [key]: value
  })
}

const updateTextField = (key: 'name' | 'url', event: Event) => {
  const target = event.target as HTMLInputElement
  updateField(key, target.value)
}

const updateNumberField = (key: 'refresh_interval_hours' | 'target_entry_count', event: Event) => {
  const target = event.target as HTMLInputElement
  updateField(key, Number(target.value))
}

const updateCheckboxField = (key: 'enabled' | 'auto_add_to_pool', event: Event) => {
  const target = event.target as HTMLInputElement
  updateField(key, target.checked)
}
</script>
