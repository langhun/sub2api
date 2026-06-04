<template>
  <div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
    <!-- Today Revenue -->
    <div class="stat-card">
      <div class="flex items-center gap-3">
        <div class="stat-icon stat-icon-success">
          <Icon name="dollar" size="md" class="text-current" :stroke-width="2" />
        </div>
        <div>
          <p class="stat-label text-xs font-medium">{{ t('payment.admin.todayRevenue') }}</p>
          <p class="stat-value">${{ formatMoney(stats.today_amount) }}</p>
          <p class="stat-label text-xs">
            {{ stats.today_count }} {{ t('payment.admin.orders') }}
          </p>
        </div>
      </div>
    </div>

    <!-- Total Revenue -->
    <div class="stat-card">
      <div class="flex items-center gap-3">
        <div class="stat-icon stat-icon-primary">
          <Icon name="creditCard" size="md" class="text-current" :stroke-width="2" />
        </div>
        <div>
          <p class="stat-label text-xs font-medium">{{ t('payment.admin.totalRevenue') }}</p>
          <p class="stat-value">${{ formatMoney(stats.total_amount) }}</p>
          <p class="stat-label text-xs">
            {{ stats.total_count }} {{ t('payment.admin.orders') }}
          </p>
        </div>
      </div>
    </div>

    <!-- Today Orders -->
    <div class="stat-card">
      <div class="flex items-center gap-3">
        <div class="stat-icon stat-icon-primary">
          <Icon name="chart" size="md" class="text-current" :stroke-width="2" />
        </div>
        <div>
          <p class="stat-label text-xs font-medium">{{ t('payment.admin.todayOrders') }}</p>
          <p class="stat-value">{{ stats.today_count }}</p>
        </div>
      </div>
    </div>

    <!-- Average Amount -->
    <div class="stat-card">
      <div class="flex items-center gap-3">
        <div class="stat-icon stat-icon-warning">
          <Icon name="chart" size="md" class="text-current" :stroke-width="2" />
        </div>
        <div>
          <p class="stat-label text-xs font-medium">{{ t('payment.admin.avgAmount') }}</p>
          <p class="stat-value">${{ formatMoney(stats.avg_amount) }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { DashboardStats } from '@/types/payment'

const { t } = useI18n()

defineProps<{
  stats: DashboardStats
}>()

function formatMoney(value: number): string {
  return value.toFixed(2)
}
</script>
