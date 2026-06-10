<template>
  <div class="w-full rounded-[16px] border border-white/75 bg-[linear-gradient(135deg,rgba(255,255,255,0.94),rgba(239,250,250,0.98))] px-2 py-1 shadow-[0_6px_16px_rgba(15,118,110,0.045)] backdrop-blur sm:px-2.5">
    <div class="flex flex-col gap-1">
      <div class="flex flex-wrap items-center gap-1">
        <div class="flex min-w-0 items-center gap-1.5">
          <div class="flex h-6 w-6 items-center justify-center rounded-lg bg-teal-500/10 text-teal-700 dark:bg-teal-400/15 dark:text-teal-200">
            <Icon name="filter" size="sm" />
          </div>
          <div class="min-w-0">
            <div class="text-[12px] font-semibold text-gray-900 dark:text-white">筛选条件</div>
          </div>
        </div>
      </div>

      <div class="grid gap-1 md:grid-cols-2 xl:grid-cols-5">
      <SearchInput
        :model-value="searchQuery"
        :placeholder="t('admin.accounts.searchAccounts')"
        class="min-w-0 w-full md:col-span-2 xl:col-span-1 2xl:col-span-1"
        @update:model-value="$emit('update:searchQuery', $event)"
        @search="$emit('change')"
      />
      <Select :model-value="filters.platform" class="w-full" :options="pOpts" @update:model-value="updatePlatform" @change="$emit('change')" />
      <Select :model-value="filters.main_status" class="w-full" :options="mainStatusOpts" @update:model-value="updateMainStatus" @change="$emit('change')" />
      <Select :model-value="filters.runtime_status" class="w-full" :options="runtimeStatusOpts" @update:model-value="updateRuntimeStatus" @change="$emit('change')" />
      <Select :model-value="filters.group" class="w-full" :options="gOpts" @update:model-value="updateGroup" @change="$emit('change')" />
      <Select :model-value="filters.tier" class="w-full" :options="tierOpts" @update:model-value="updateTier" @change="$emit('change')" />
      <Select :model-value="filters.type" class="w-full" :options="tOpts" @update:model-value="updateType" @change="$emit('change')" />
      <Select :model-value="filters.scheduling_status" class="w-full" :options="schedulingStatusOpts" @update:model-value="updateSchedulingStatus" @change="$emit('change')" />
      <Select :model-value="filters.privacy_mode" class="w-full" :options="privacyOpts" @update:model-value="updatePrivacyMode" @change="$emit('change')" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Select from '@/components/common/Select.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import Icon from '@/components/icons/Icon.vue'
import type { AdminGroup } from '@/types'

const props = defineProps<{ searchQuery: string; filters: Record<string, any>; groups?: AdminGroup[] }>()
const emit = defineEmits(['update:searchQuery', 'update:filters', 'change', 'status-guide'])
const { t } = useI18n()

type TierOption = {
  value: string
  label: string
  platform?: string
  kind?: 'group'
  disabled?: boolean
}

const tierOptionsByPlatform = computed<Record<string, TierOption[]>>(() => ({
  openai: [
    { value: 'openai:free', label: t('admin.accounts.tier.free'), platform: 'openai' },
    { value: 'openai:plus', label: t('admin.accounts.tier.plus'), platform: 'openai' },
    { value: 'openai:team', label: t('admin.accounts.tier.team'), platform: 'openai' },
    { value: 'openai:pro', label: t('admin.accounts.tier.pro'), platform: 'openai' },
    { value: 'openai:enterprise', label: t('admin.accounts.tier.enterprise'), platform: 'openai' }
  ],
  gemini: [
    { value: 'gemini:google_one_free', label: t('admin.accounts.tier.googleOneFree'), platform: 'gemini' },
    { value: 'gemini:google_ai_pro', label: t('admin.accounts.tier.googleAIPro'), platform: 'gemini' },
    { value: 'gemini:google_ai_ultra', label: t('admin.accounts.tier.googleAIUltra'), platform: 'gemini' },
    { value: 'gemini:gcp_standard', label: t('admin.accounts.tier.gcpStandard'), platform: 'gemini' },
    { value: 'gemini:gcp_enterprise', label: t('admin.accounts.tier.gcpEnterprise'), platform: 'gemini' },
    { value: 'gemini:aistudio_free', label: t('admin.accounts.tier.aiStudioFree'), platform: 'gemini' },
    { value: 'gemini:aistudio_paid', label: t('admin.accounts.tier.aiStudioPaid'), platform: 'gemini' },
    { value: 'gemini:google_one_unknown', label: t('admin.accounts.tier.unknown'), platform: 'gemini' }
  ],
  antigravity: [
    { value: 'antigravity:free-tier', label: t('admin.accounts.tier.free'), platform: 'antigravity' },
    { value: 'antigravity:g1-pro-tier', label: t('admin.accounts.tier.pro'), platform: 'antigravity' },
    { value: 'antigravity:g1-ultra-tier', label: t('admin.accounts.tier.ultra'), platform: 'antigravity' }
  ]
}))

const tierGroupLabels: Record<string, string> = {
  openai: 'OpenAI',
  gemini: 'Gemini',
  antigravity: 'Antigravity'
}

const toFilterValue = (value: string | number | boolean | null) => String(value ?? '')

const tierPlatform = (tier: string) => {
  const [platform] = tier.split(':', 1)
  return platform || ''
}

const isTierCompatibleWithPlatform = (tier: string, platform: string) => {
  if (!tier || !platform) return true
  return tierPlatform(tier) === platform
}

const updatePlatform = (value: string | number | boolean | null) => {
  const platform = toFilterValue(value)
  const nextFilters: Record<string, any> = { ...props.filters, platform }
  if (!isTierCompatibleWithPlatform(toFilterValue(nextFilters.tier), platform)) {
    nextFilters.tier = ''
  }
  emit('update:filters', nextFilters)
}
const updateTier = (value: string | number | boolean | null) => { emit('update:filters', { ...props.filters, tier: toFilterValue(value) }) }
const updateType = (value: string | number | boolean | null) => { emit('update:filters', { ...props.filters, type: value }) }
const updateMainStatus = (value: string | number | boolean | null) => { emit('update:filters', { ...props.filters, main_status: toFilterValue(value) }) }
const updateRuntimeStatus = (value: string | number | boolean | null) => { emit('update:filters', { ...props.filters, runtime_status: toFilterValue(value) }) }
const updateSchedulingStatus = (value: string | number | boolean | null) => { emit('update:filters', { ...props.filters, scheduling_status: toFilterValue(value) }) }
const updatePrivacyMode = (value: string | number | boolean | null) => { emit('update:filters', { ...props.filters, privacy_mode: value }) }
const updateGroup = (value: string | number | boolean | null) => { emit('update:filters', { ...props.filters, group: value }) }
const pOpts = computed(() => [{ value: '', label: t('admin.accounts.allPlatforms') }, { value: 'anthropic', label: 'Anthropic' }, { value: 'openai', label: 'OpenAI' }, { value: 'gemini', label: 'Gemini' }, { value: 'antigravity', label: 'Antigravity' }])
const tierOpts = computed<TierOption[]>(() => {
  const selectedPlatform = toFilterValue(props.filters.platform)
  const allOption = { value: '', label: t('admin.accounts.tier.all') }
  if (selectedPlatform) {
    return [allOption, ...(tierOptionsByPlatform.value[selectedPlatform] || [])]
  }
  return [
    allOption,
    ...Object.entries(tierOptionsByPlatform.value).flatMap(([platform, options]) => [
      { value: `__group_${platform}`, label: tierGroupLabels[platform] || platform, kind: 'group' as const, disabled: true },
      ...options
    ])
  ]
})
const tOpts = computed(() => [{ value: '', label: t('admin.accounts.allTypes') }, { value: 'oauth', label: t('admin.accounts.oauthType') }, { value: 'setup-token', label: t('admin.accounts.setupToken') }, { value: 'apikey', label: t('admin.accounts.apiKey') }, { value: 'bedrock', label: 'AWS Bedrock' }])
const mainStatusOpts = computed(() => [
  { value: '', label: t('admin.accounts.statusFilters.allMain') },
  { value: 'active', label: t('admin.accounts.status.mainActive') },
  { value: 'inactive', label: t('admin.accounts.status.mainInactive') },
  { value: 'error', label: t('admin.accounts.status.mainError') }
])
const runtimeStatusOpts = computed(() => [
  { value: '', label: t('admin.accounts.statusFilters.allRuntime') },
  { value: 'normal', label: t('admin.accounts.status.runtimeNormal') },
  { value: 'rate_limited', label: t('admin.accounts.status.runtimeRateLimited') },
  { value: 'overloaded', label: t('admin.accounts.status.runtimeOverloaded') },
  { value: 'temp_unschedulable', label: t('admin.accounts.statusFilters.tempUnschedulable') }
])
const schedulingStatusOpts = computed(() => [
  { value: '', label: t('admin.accounts.statusFilters.allScheduling') },
  { value: 'enabled', label: t('admin.accounts.status.scheduleEnabled') },
  { value: 'paused', label: t('admin.accounts.statusFilters.unschedulable') }
])
const privacyOpts = computed(() => [
  { value: '', label: t('admin.accounts.allPrivacyModes') },
  { value: '__unset__', label: t('admin.accounts.privacyUnset') },
  { value: 'training_off', label: 'Privacy' },
  { value: 'training_set_cf_blocked', label: 'CF' },
  { value: 'training_set_failed', label: 'Fail' }
])
const gOpts = computed(() => [
  { value: '', label: t('admin.accounts.allGroups') },
  { value: 'ungrouped', label: t('admin.accounts.ungroupedGroup') },
  ...(props.groups || []).map(g => ({ value: String(g.id), label: g.name }))
])
</script>
