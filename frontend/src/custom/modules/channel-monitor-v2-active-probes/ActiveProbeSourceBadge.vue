<template>
  <span v-if="source?.active_probe" class="inline-flex flex-wrap items-center gap-1.5">
    <span class="badge" :class="sourceHealthy ? 'badge-success' : 'badge-danger'">{{ t('channelMonitorV2ActiveProbes.source.active') }}</span>
    <span v-if="source.passive_traffic" class="badge badge-gray">{{ t('channelMonitorV2ActiveProbes.source.passive') }}</span>
    <span v-if="source.last_active_probe_at" class="text-xs text-gray-400">
      {{ t('channelMonitorV2ActiveProbes.source.checkedAt', { time: formatTime(source.last_active_probe_at) }) }}
    </span>
  </span>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { computed } from 'vue'
import type { MonitorExtensions } from '@/api/channelMonitorV2'

interface ActiveProbeSource {
  passive_traffic?: boolean
  active_probe?: boolean
  last_active_probe_at?: string
  active_probe_latency_ms?: number
  active_probe_status?: string
}

const props = defineProps<{ extensions?: MonitorExtensions }>()
const source = computed<ActiveProbeSource | undefined>(() => {
  const value = props.extensions?.channel_monitor_v2_active_probes
  return value && typeof value === 'object' ? value as ActiveProbeSource : undefined
})
const sourceHealthy = computed(() => ['operational', 'degraded'].includes(source.value?.active_probe_status || ''))

const { t, locale } = useI18n()
function formatTime(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : new Intl.DateTimeFormat(locale.value, { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }).format(date)
}
</script>
