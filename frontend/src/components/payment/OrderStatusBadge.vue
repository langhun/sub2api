<template>
  <span
    class="inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium"
    :class="statusClass"
  >
    {{ statusLabel }}
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { OrderStatus } from '@/types/payment'

const props = defineProps<{
  status: OrderStatus
}>()

const { t } = useI18n()

const statusMap: Record<OrderStatus, { key: string; class: string }> = {
  PENDING: { key: 'payment.status.pending', class: 'bg-amber-50 text-amber-700 ring-1 ring-amber-200 dark:bg-amber-950/30 dark:text-amber-300 dark:ring-amber-800/50' },
  PAID: { key: 'payment.status.paid', class: 'bg-sky-50 text-sky-700 ring-1 ring-sky-200 dark:bg-sky-950/30 dark:text-sky-300 dark:ring-sky-800/50' },
  RECHARGING: { key: 'payment.status.recharging', class: 'bg-sky-50 text-sky-700 ring-1 ring-sky-200 dark:bg-sky-950/30 dark:text-sky-300 dark:ring-sky-800/50' },
  COMPLETED: { key: 'payment.status.completed', class: 'bg-emerald-50 text-emerald-700 ring-1 ring-emerald-200 dark:bg-emerald-950/30 dark:text-emerald-300 dark:ring-emerald-800/50' },
  EXPIRED: { key: 'payment.status.expired', class: 'bg-amber-50 text-amber-700 ring-1 ring-amber-200 dark:bg-amber-950/30 dark:text-amber-300 dark:ring-amber-800/50' },
  CANCELLED: { key: 'payment.status.cancelled', class: 'bg-slate-100 text-slate-700 ring-1 ring-slate-200 dark:bg-slate-900/50 dark:text-slate-300 dark:ring-slate-700/60' },
  FAILED: { key: 'payment.status.failed', class: 'bg-red-50 text-red-700 ring-1 ring-red-200 dark:bg-red-950/30 dark:text-red-300 dark:ring-red-800/50' },
  REFUND_REQUESTED: { key: 'payment.status.refund_requested', class: 'bg-orange-50 text-orange-700 ring-1 ring-orange-200 dark:bg-orange-950/30 dark:text-orange-300 dark:ring-orange-800/50' },
  REFUNDING: { key: 'payment.status.refunding', class: 'bg-orange-50 text-orange-700 ring-1 ring-orange-200 dark:bg-orange-950/30 dark:text-orange-300 dark:ring-orange-800/50' },
  REFUNDED: { key: 'payment.status.refunded', class: 'bg-violet-50 text-violet-700 ring-1 ring-violet-200 dark:bg-violet-950/30 dark:text-violet-300 dark:ring-violet-800/50' },
  PARTIALLY_REFUNDED: { key: 'payment.status.partially_refunded', class: 'bg-violet-50 text-violet-700 ring-1 ring-violet-200 dark:bg-violet-950/30 dark:text-violet-300 dark:ring-violet-800/50' },
  REFUND_FAILED: { key: 'payment.status.refund_failed', class: 'bg-red-50 text-red-700 ring-1 ring-red-200 dark:bg-red-950/30 dark:text-red-300 dark:ring-red-800/50' },
}

const statusLabel = computed(() => {
  const entry = statusMap[props.status]
  return entry ? t(entry.key) : props.status
})

const statusClass = computed(() => {
  const entry = statusMap[props.status]
  return entry?.class ?? 'bg-muted text-muted-foreground ring-1 ring-border'
})
</script>
