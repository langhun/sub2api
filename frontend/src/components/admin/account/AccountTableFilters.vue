<template>
  <div class="w-full rounded-[16px] border border-white/75 bg-[linear-gradient(135deg,rgba(255,255,255,0.94),rgba(239,250,250,0.98))] px-2 py-1 shadow-[0_6px_16px_rgba(15,118,110,0.045)] backdrop-blur dark:border-dark-700/80 dark:bg-[linear-gradient(135deg,rgba(15,23,42,0.94),rgba(17,24,39,0.98))] dark:shadow-none sm:px-2.5">
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

      <div class="grid gap-1 md:grid-cols-2 xl:grid-cols-5 2xl:grid-cols-9">
        <SearchInput
          :model-value="searchQuery"
          :placeholder="t('admin.accounts.searchAccounts')"
          class="min-w-0 w-full md:col-span-2 xl:col-span-2 2xl:col-span-1"
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
import { computed } from 'vue'; import { useI18n } from 'vue-i18n'; import Select from '@/components/common/Select.vue'; import SearchInput from '@/components/common/SearchInput.vue'
import type { AdminGroup } from '@/types'
const props = defineProps<{ searchQuery: string; filters: Record<string, any>; groups?: AdminGroup[] }>()
const emit = defineEmits(['update:searchQuery', 'update:filters', 'change']); const { t } = useI18n()
const updatePlatform = (value: string | number | boolean | null) => { emit('update:filters', { ...props.filters, platform: value }) }
const updateType = (value: string | number | boolean | null) => { emit('update:filters', { ...props.filters, type: value }) }
const updateStatus = (value: string | number | boolean | null) => { emit('update:filters', { ...props.filters, status: value }) }
const updatePrivacyMode = (value: string | number | boolean | null) => { emit('update:filters', { ...props.filters, privacy_mode: value }) }
const updateGroup = (value: string | number | boolean | null) => { emit('update:filters', { ...props.filters, group: value }) }
const pOpts = computed(() => [{ value: '', label: t('admin.accounts.allPlatforms') }, { value: 'anthropic', label: 'Anthropic' }, { value: 'openai', label: 'OpenAI' }, { value: 'gemini', label: 'Gemini' }, { value: 'antigravity', label: 'Antigravity' }])
const tOpts = computed(() => [{ value: '', label: t('admin.accounts.allTypes') }, { value: 'oauth', label: t('admin.accounts.oauthType') }, { value: 'setup-token', label: t('admin.accounts.setupToken') }, { value: 'apikey', label: t('admin.accounts.apiKey') }, { value: 'bedrock', label: 'AWS Bedrock' }])
const sOpts = computed(() => [{ value: '', label: t('admin.accounts.allStatus') }, { value: 'active', label: t('admin.accounts.status.active') }, { value: 'inactive', label: t('admin.accounts.status.inactive') }, { value: 'error', label: t('admin.accounts.status.error') }, { value: 'rate_limited', label: t('admin.accounts.status.rateLimited') }, { value: 'temp_unschedulable', label: t('admin.accounts.status.tempUnschedulable') }, { value: 'unschedulable', label: t('admin.accounts.status.unschedulable') }])
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
