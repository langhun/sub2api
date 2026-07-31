<template>
  <div class="card">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.settings.balanceFeatures.checkinTitle') }}</h2>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.settings.balanceFeatures.checkinDescription') }}</p>
    </div>
    <div class="space-y-5 p-6">
      <div class="flex items-center justify-between">
        <span class="font-medium text-gray-900 dark:text-white">{{ t('admin.settings.balanceFeatures.normalCheckin') }}</span>
        <Toggle v-model="form.checkin_enabled" />
      </div>
      <div v-if="form.checkin_enabled" class="grid gap-4 md:grid-cols-2">
        <label class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.settings.balanceFeatures.minReward') }}<input v-model.number="form.checkin_min_balance" type="number" min="0" step="0.01" class="input mt-1" /></label>
        <label class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.settings.balanceFeatures.maxReward') }}<input v-model.number="form.checkin_max_balance" type="number" min="0" step="0.01" class="input mt-1" /></label>
      </div>
      <div class="flex items-center justify-between border-t border-gray-100 pt-4 dark:border-dark-700">
        <span class="font-medium text-gray-900 dark:text-white">{{ t('admin.settings.balanceFeatures.luckCheckin') }}</span>
        <Toggle v-model="form.checkin_luck_enabled" />
      </div>
      <div v-if="form.checkin_luck_enabled" class="grid gap-4 md:grid-cols-2">
        <label class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.settings.balanceFeatures.minMultiplier') }}<input v-model.number="form.checkin_luck_min_multiplier" type="number" min="0" step="0.1" class="input mt-1" /></label>
        <label class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.settings.balanceFeatures.maxMultiplier') }}<input v-model.number="form.checkin_luck_max_multiplier" type="number" min="0" step="0.1" class="input mt-1" /></label>
      </div>
      <div class="flex items-center justify-between border-t border-gray-100 pt-4 dark:border-dark-700">
        <span class="font-medium text-gray-900 dark:text-white">{{ t('admin.settings.balanceFeatures.redpacketEnabled') }}</span>
        <Toggle v-model="form.redpacket_enabled" />
      </div>
      <div v-if="form.redpacket_enabled" class="grid gap-4 md:grid-cols-2">
        <label class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.settings.balanceFeatures.redpacketMaxCount') }}<input v-model.number="form.redpacket_max_count" type="number" min="1" class="input mt-1" /></label>
        <label class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.settings.balanceFeatures.redpacketExpireHours') }}<input v-model.number="form.redpacket_expire_hours" type="number" min="1" class="input mt-1" /></label>
      </div>
    </div>
  </div>

  <BlindboxPrizePoolCard
    v-model:enabled="form.checkin_blindbox_enabled"
    v-model:trigger-type="form.checkin_blindbox_trigger_type"
    v-model:interval="form.checkin_blindbox_interval"
  />

  <RewardDeliveryOpsPanel />

  <CodeFormatSettingsEditor v-model="form.code_format_settings" @validity="form.code_format_settings_valid = $event" />
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { CodeFormatSettings } from '../settings'
import CodeFormatSettingsEditor from './CodeFormatSettingsEditor.vue'
import Toggle from '@/components/common/Toggle.vue'
import BlindboxPrizePoolCard from './components/BlindboxPrizePoolCard.vue'
import RewardDeliveryOpsPanel from './components/RewardDeliveryOpsPanel.vue'

interface ActivitySettings {
  checkin_enabled: boolean
  checkin_min_balance: number
  checkin_max_balance: number
  checkin_luck_enabled: boolean
  checkin_luck_min_multiplier: number
  checkin_luck_max_multiplier: number
  checkin_blindbox_enabled: boolean
  checkin_blindbox_trigger_type: string
  checkin_blindbox_interval: number
  redpacket_enabled: boolean
  redpacket_max_count: number
  redpacket_expire_hours: number
  code_format_settings: CodeFormatSettings
  code_format_settings_valid: boolean
}

const props = defineProps<{ modelValue: ActivitySettings }>()
const emit = defineEmits<{ 'update:modelValue': [value: ActivitySettings] }>()
const { t } = useI18n()

function cloneSettings(value: ActivitySettings): ActivitySettings {
  return {
    ...value,
    code_format_settings: Object.fromEntries(
      Object.entries(value.code_format_settings).map(([key, rule]) => [key, { ...rule }]),
    ) as CodeFormatSettings,
  }
}

const form = reactive<ActivitySettings>(cloneSettings(props.modelValue))

function hasSameSettings(left: ActivitySettings, right: ActivitySettings): boolean {
  return JSON.stringify(left) === JSON.stringify(right)
}

watch(() => props.modelValue, (value) => {
  const next = cloneSettings(value)
  if (!hasSameSettings(form, next)) {
    Object.assign(form, next)
  }
}, { deep: true })

watch(form, (value) => {
  const next = cloneSettings(value)
  if (!hasSameSettings(next, props.modelValue)) {
    emit('update:modelValue', next)
  }
}, { deep: true })
</script>
