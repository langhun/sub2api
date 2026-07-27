<template>
  <div class="card">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
        {{ t('admin.settings.balanceFeatures.entrySwitchesTitle') }}
      </h2>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
        {{ t('admin.settings.balanceFeatures.entrySwitchesDescription') }}
      </p>
    </div>
    <div class="grid gap-x-8 gap-y-5 p-6 md:grid-cols-2">
      <ActivityUsageSettingsPanel
        :model-value="usageSettings"
        @update:model-value="updateUsageSettings"
      />
      <ActivityLeaderboardSettingsPanel
        :model-value="leaderboardSettings"
        @update:model-value="updateLeaderboardSettings"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import ActivityLeaderboardSettingsPanel from './ActivityLeaderboardSettingsPanel.vue'
import ActivityUsageSettingsPanel from './ActivityUsageSettingsPanel.vue'

interface ActivityEntrySwitchesSettings {
  usage_query_enabled: boolean
  leaderboard_enabled: boolean
  leaderboard_balance_enabled: boolean
  leaderboard_consumption_enabled: boolean
  leaderboard_checkin_enabled: boolean
  leaderboard_transfer_enabled: boolean
  leaderboard_include_admin: boolean
}

type ActivityUsageSettings = Pick<ActivityEntrySwitchesSettings, 'usage_query_enabled'>
type ActivityLeaderboardSettings = Omit<ActivityEntrySwitchesSettings, 'usage_query_enabled'>

const props = defineProps<{ modelValue: ActivityEntrySwitchesSettings }>()
const emit = defineEmits<{ 'update:modelValue': [value: ActivityEntrySwitchesSettings] }>()
const { t } = useI18n()
const form = reactive<ActivityEntrySwitchesSettings>({ ...props.modelValue })

const usageSettings = computed<ActivityUsageSettings>(() => ({
  usage_query_enabled: form.usage_query_enabled,
}))

const leaderboardSettings = computed<ActivityLeaderboardSettings>(() => ({
  leaderboard_enabled: form.leaderboard_enabled,
  leaderboard_balance_enabled: form.leaderboard_balance_enabled,
  leaderboard_consumption_enabled: form.leaderboard_consumption_enabled,
  leaderboard_checkin_enabled: form.leaderboard_checkin_enabled,
  leaderboard_transfer_enabled: form.leaderboard_transfer_enabled,
  leaderboard_include_admin: form.leaderboard_include_admin,
}))

function hasSameSettings(left: ActivityEntrySwitchesSettings, right: ActivityEntrySwitchesSettings): boolean {
  return JSON.stringify(left) === JSON.stringify(right)
}

function emitUpdate(): void {
  emit('update:modelValue', { ...form })
}

function updateUsageSettings(value: ActivityUsageSettings): void {
  if (form.usage_query_enabled === value.usage_query_enabled) return
  form.usage_query_enabled = value.usage_query_enabled
  emitUpdate()
}

function updateLeaderboardSettings(value: ActivityLeaderboardSettings): void {
  const next = { ...form, ...value }
  if (hasSameSettings(form, next)) return
  Object.assign(form, value)
  emitUpdate()
}

watch(() => props.modelValue, (value) => {
  if (!hasSameSettings(form, value)) {
    Object.assign(form, value)
  }
}, { deep: true })
</script>
