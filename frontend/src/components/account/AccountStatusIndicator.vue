<template>
  <div class="flex flex-col items-start gap-1.5">
    <div class="space-y-1">
      <div class="flex items-center gap-2">
        <span class="w-[64px] shrink-0 text-[11px] text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.statusLayers.main') }}
        </span>
        <span :class="['badge text-xs', mainStatusClass]">
          {{ mainStatusText }}
        </span>

        <div v-if="hasError && account.error_message" class="group/error relative">
          <svg
            class="h-4 w-4 cursor-help text-red-500 transition-colors hover:text-red-600 dark:text-red-400 dark:hover:text-red-300"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            stroke-width="2"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              d="M9.879 7.519c1.171-1.025 3.071-1.025 4.242 0 1.172 1.025 1.172 2.687 0 3.712-.203.179-.43.326-.67.442-.745.361-1.45.999-1.45 1.827v.75M21 12a9 9 0 11-18 0 9 9 0 0118 0zm-9 5.25h.008v.008H12v-.008z"
            />
          </svg>
          <div
            class="invisible absolute left-0 top-full z-[100] mt-1.5 min-w-[200px] max-w-[300px] rounded-lg bg-gray-800 px-3 py-2 text-xs text-white opacity-0 shadow-xl transition-all duration-200 group-hover/error:visible group-hover/error:opacity-100 dark:bg-gray-900"
          >
            <div class="whitespace-pre-wrap break-words leading-relaxed text-gray-300">
              {{ account.error_message }}
            </div>
            <div
              class="absolute bottom-full left-3 border-[6px] border-transparent border-b-gray-800 dark:border-b-gray-900"
            ></div>
          </div>
        </div>
      </div>

      <div class="flex items-center gap-2">
        <span class="w-[64px] shrink-0 text-[11px] text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.statusLayers.scheduling') }}
        </span>
        <span :class="['badge text-xs', schedulingStatusClass]">
          {{ schedulingStatusText }}
        </span>
      </div>

      <div class="flex items-start gap-2">
        <span class="w-[64px] shrink-0 pt-0.5 text-[11px] text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.statusLayers.runtime') }}
        </span>
        <div class="min-w-0">
          <button
            v-if="runtimeState.clickable"
            type="button"
            :class="['badge text-xs cursor-pointer', runtimeStatusClass]"
            :title="t('admin.accounts.status.viewTempUnschedDetails')"
            @click="handleTempUnschedClick"
          >
            {{ runtimeStatusText }}
          </button>
          <span v-else :class="['badge text-xs', runtimeStatusClass]">
            {{ runtimeStatusText }}
          </span>

          <div
            v-if="runtimeDetailText"
            class="mt-1 text-[11px] leading-4 text-gray-500 dark:text-gray-400"
          >
            {{ runtimeDetailText }}
          </div>
        </div>
      </div>
    </div>

    <div
      v-if="activeModelStatuses.length > 0"
      :class="[
        activeModelStatuses.length <= 4
          ? 'flex flex-col gap-1'
          : activeModelStatuses.length <= 8
            ? 'columns-2 gap-x-2'
            : 'columns-3 gap-x-2'
      ]"
    >
      <div
        v-for="item in activeModelStatuses"
        :key="`${item.kind}-${item.model}`"
        class="group relative mb-1 break-inside-avoid"
      >
        <span
          v-if="item.kind === 'credits_exhausted'"
          class="inline-flex items-center gap-1 rounded bg-red-100 px-1.5 py-0.5 text-xs font-medium text-red-700 dark:bg-red-900/30 dark:text-red-400"
        >
          <Icon name="exclamationTriangle" size="xs" :stroke-width="2" />
          {{ t('admin.accounts.status.creditsExhausted') }}
          <span class="text-[10px] opacity-70">{{ formatModelResetTime(item.reset_at) }}</span>
        </span>
        <span
          v-else-if="item.kind === 'credits_active'"
          class="inline-flex items-center gap-1 rounded bg-amber-100 px-1.5 py-0.5 text-xs font-medium text-amber-700 dark:bg-amber-900/30 dark:text-amber-400"
        >
          <span>⚡</span>
          {{ formatScopeName(item.model) }}
          <span class="text-[10px] opacity-70">{{ formatModelResetTime(item.reset_at) }}</span>
        </span>
        <span
          v-else
          class="inline-flex items-center gap-1 rounded bg-purple-100 px-1.5 py-0.5 text-xs font-medium text-purple-700 dark:bg-purple-900/30 dark:text-purple-400"
        >
          <Icon name="exclamationTriangle" size="xs" :stroke-width="2" />
          {{ formatScopeName(item.model) }}
          <span class="text-[10px] opacity-70">{{ formatModelResetTime(item.reset_at) }}</span>
        </span>
        <div
          class="pointer-events-none absolute bottom-full left-1/2 z-50 mb-2 w-56 -translate-x-1/2 whitespace-normal rounded bg-gray-900 px-3 py-2 text-center text-xs leading-relaxed text-white opacity-0 transition-opacity group-hover:opacity-100 dark:bg-gray-700"
        >
          {{
            item.kind === 'credits_exhausted'
              ? t('admin.accounts.status.creditsExhaustedUntil', { time: formatTime(item.reset_at) })
              : item.kind === 'credits_active'
                ? t('admin.accounts.status.modelCreditOveragesUntil', {
                    model: formatScopeName(item.model),
                    time: formatTime(item.reset_at),
                  })
                : t('admin.accounts.status.modelRateLimitedUntil', {
                    model: formatScopeName(item.model),
                    time: formatTime(item.reset_at),
                  })
          }}
          <div
            class="absolute left-1/2 top-full -translate-x-1/2 border-4 border-transparent border-t-gray-900 dark:border-t-gray-700"
          ></div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { Account } from '@/types'
import { formatCountdown, formatCountdownWithSuffix, formatTime } from '@/utils/format'
import {
  getAccountMainStatusState,
  getAccountRuntimeStatusState,
  getAccountSchedulingStatusState,
  getBadgeClassForTone,
} from '@/utils/accountStatus'

const { t } = useI18n()

const props = defineProps<{
  account: Account
}>()

const emit = defineEmits<{
  (e: 'show-temp-unsched', account: Account): void
}>()

type AccountModelStatusItem = {
  kind: 'rate_limit' | 'credits_exhausted' | 'credits_active'
  model: string
  reset_at: string
}

const activeModelStatuses = computed<AccountModelStatusItem[]>(() => {
  const extra = props.account.extra as Record<string, unknown> | undefined
  const modelLimits = extra?.model_rate_limits as
    | Record<string, { rate_limited_at: string; rate_limit_reset_at: string }>
    | undefined
  const now = new Date()
  const items: AccountModelStatusItem[] = []

  if (!modelLimits) return items

  const aiCreditsEntry = modelLimits['AICredits']
  const hasActiveAICredits = aiCreditsEntry && new Date(aiCreditsEntry.rate_limit_reset_at) > now
  const allowOverages = !!extra?.allow_overages

  for (const [model, info] of Object.entries(modelLimits)) {
    if (new Date(info.rate_limit_reset_at) <= now) continue

    if (model === 'AICredits') {
      items.push({ kind: 'credits_exhausted', model, reset_at: info.rate_limit_reset_at })
    } else if (allowOverages && !hasActiveAICredits) {
      items.push({ kind: 'credits_active', model, reset_at: info.rate_limit_reset_at })
    } else {
      items.push({ kind: 'rate_limit', model, reset_at: info.rate_limit_reset_at })
    }
  }

  return items
})

const formatScopeName = (scope: string): string => {
  const aliases: Record<string, string> = {
    'claude-opus-4-6': 'COpus46',
    'claude-opus-4-6-thinking': 'COpus46T',
    'claude-sonnet-4-6': 'CSon46',
    'claude-sonnet-4-5': 'CSon45',
    'claude-sonnet-4-5-thinking': 'CSon45T',
    'gemini-2.5-flash': 'G25F',
    'gemini-2.5-flash-lite': 'G25FL',
    'gemini-2.5-flash-thinking': 'G25FT',
    'gemini-2.5-pro': 'G25P',
    'gemini-2.5-flash-image': 'G25I',
    'gemini-3-flash': 'G3F',
    'gemini-3.1-pro-high': 'G3PH',
    'gemini-3.1-pro-low': 'G3PL',
    'gemini-3-pro-image': 'G3PI',
    'gemini-3.1-flash-image': 'G31FI',
    'gpt-oss-120b-medium': 'GPT120',
    tab_flash_lite_preview: 'TabFL',
    claude: 'Claude',
    claude_sonnet: 'CSon',
    claude_opus: 'COpus',
    claude_haiku: 'CHaiku',
    gemini_text: 'Gemini',
    gemini_image: 'GImg',
    gemini_flash: 'GFlash',
    gemini_pro: 'GPro',
  }
  return aliases[scope] || scope
}

const formatModelResetTime = (resetAt: string): string => {
  const date = new Date(resetAt)
  const now = new Date()
  const diffMs = date.getTime() - now.getTime()
  if (diffMs <= 0) return ''
  const totalSecs = Math.floor(diffMs / 1000)
  const h = Math.floor(totalSecs / 3600)
  const m = Math.floor((totalSecs % 3600) / 60)
  const s = totalSecs % 60
  if (h > 0) return `${h}h${m}m`
  if (m > 0) return `${m}m${s}s`
  return `${s}s`
}

const hasError = computed(() => props.account.status === 'error')

const mainStatusState = computed(() => getAccountMainStatusState(props.account))

const schedulingStatusState = computed(() => getAccountSchedulingStatusState(props.account))

const runtimeState = computed(() => getAccountRuntimeStatusState(props.account))

const rateLimitCountdown = computed(() => formatCountdown(props.account.rate_limit_reset_at))

const rateLimitResumeText = computed(() => {
  if (!rateLimitCountdown.value) return ''
  return t('admin.accounts.status.rateLimitedAutoResume', { time: rateLimitCountdown.value })
})

const overloadCountdown = computed(() => formatCountdownWithSuffix(props.account.overload_until))

const statusTextKeyMap = {
  main_active: 'admin.accounts.status.mainActive',
  main_inactive: 'admin.accounts.status.mainInactive',
  main_error: 'admin.accounts.status.mainError',
  schedule_enabled: 'admin.accounts.status.scheduleEnabled',
  schedule_manual_paused: 'admin.accounts.status.scheduleManualPaused',
  schedule_expired_paused: 'admin.accounts.status.scheduleExpiredPaused',
  runtime_normal: 'admin.accounts.status.runtimeNormal',
  runtime_rate_limited: 'admin.accounts.status.runtimeRateLimited',
  runtime_overloaded: 'admin.accounts.status.runtimeOverloaded',
  runtime_oauth401_cooldown: 'admin.accounts.status.runtimeOauth401Cooldown',
  runtime_forbidden_cooldown: 'admin.accounts.status.runtimeForbiddenCooldown',
  runtime_http_cooldown: 'admin.accounts.status.runtimeHttpCooldown',
  runtime_stream_timeout_cooldown: 'admin.accounts.status.runtimeStreamTimeoutCooldown',
  runtime_token_refresh_cooldown: 'admin.accounts.status.runtimeTokenRefreshCooldown',
  runtime_quota_exceeded: 'admin.accounts.status.runtimeQuotaExceeded',
  runtime_temp_unschedulable: 'admin.accounts.status.runtimeTempUnschedulable',
} as const

const mainStatusClass = computed(() => getBadgeClassForTone(mainStatusState.value.tone))

const mainStatusText = computed(() => t(statusTextKeyMap[mainStatusState.value.code]))

const schedulingStatusClass = computed(() => getBadgeClassForTone(schedulingStatusState.value.tone))

const schedulingStatusText = computed(() => t(statusTextKeyMap[schedulingStatusState.value.code]))

const runtimeStatusClass = computed(() => getBadgeClassForTone(runtimeState.value.tone))

const runtimeStatusText = computed(() => {
  if (runtimeState.value.code === 'runtime_http_cooldown') {
    return t(statusTextKeyMap[runtimeState.value.code], {
      code: runtimeState.value.statusCode,
    })
  }
  return t(statusTextKeyMap[runtimeState.value.code])
})

const runtimeDetailText = computed(() => {
  if (runtimeState.value.code === 'runtime_rate_limited') {
    return rateLimitResumeText.value
  }
  if (runtimeState.value.code === 'runtime_overloaded') {
    return overloadCountdown.value
      ? t('admin.accounts.status.overloadedAutoResume', { time: overloadCountdown.value })
      : t('admin.accounts.status.runtimeOverloaded')
  }
  if (runtimeState.value.clickable && runtimeState.value.until) {
    const countdown = formatCountdown(runtimeState.value.until)
    if (!countdown) return runtimeStatusText.value
    return t('admin.accounts.status.tempUnschedAutoResume', {
      reason: runtimeStatusText.value,
      time: countdown,
    })
  }
  if (runtimeState.value.code === 'runtime_quota_exceeded') {
    return t('admin.accounts.status.quotaExceededHint')
  }
  return ''
})

const handleTempUnschedClick = () => {
  if (!runtimeState.value.clickable) return
  emit('show-temp-unsched', props.account)
}
</script>
