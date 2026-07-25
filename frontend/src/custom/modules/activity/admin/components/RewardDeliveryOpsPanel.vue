<template>
  <section class="border-t border-gray-100 pt-5 dark:border-dark-700">
    <div class="mb-4 flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
      <div>
        <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
          {{ t('admin.blindbox.deliveryTitle') }}
        </h3>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.blindbox.deliveryDescription') }}
        </p>
      </div>
      <div class="flex items-center gap-2">
        <label class="sr-only" for="reward-delivery-status">{{ t('admin.blindbox.deliveryStatusFilter') }}</label>
        <select id="reward-delivery-status" v-model="statusFilter" class="input min-w-36 py-2 text-sm" @change="resetAndLoad">
          <option value="">{{ t('admin.blindbox.deliveryAllStatuses') }}</option>
          <option value="failed">{{ t('admin.blindbox.deliveryStatus.failed') }}</option>
          <option value="pending">{{ t('admin.blindbox.deliveryStatus.pending') }}</option>
          <option value="delivering">{{ t('admin.blindbox.deliveryStatus.delivering') }}</option>
          <option value="delivered">{{ t('admin.blindbox.deliveryStatus.delivered') }}</option>
          <option value="compensated">{{ t('admin.blindbox.deliveryStatus.compensated') }}</option>
        </select>
        <button
          type="button"
          class="rounded-md p-2 text-gray-500 hover:bg-gray-100 hover:text-gray-800 disabled:opacity-50 dark:text-gray-400 dark:hover:bg-dark-700 dark:hover:text-white"
          :title="t('common.refresh')"
          :disabled="loading"
          @click="loadDeliveries"
        >
          <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
        </button>
      </div>
    </div>

    <div v-if="operationError" role="alert" class="mb-3 rounded-md bg-red-50 px-3 py-2 text-xs text-red-700 dark:bg-red-900/20 dark:text-red-300">
      {{ operationError }}
    </div>

    <div class="overflow-x-auto rounded-md border border-gray-200 dark:border-dark-700">
      <div v-if="loading && deliveries.length === 0" class="flex items-center justify-center py-8">
        <Icon name="refresh" size="md" class="animate-spin text-gray-400" />
      </div>
      <div v-else-if="deliveries.length === 0" class="py-8 text-center text-sm text-gray-400 dark:text-gray-500">
        {{ t('admin.blindbox.deliveryEmpty') }}
      </div>
      <table v-else class="w-full min-w-[860px] text-sm">
        <thead class="border-b border-gray-100 bg-gray-50 dark:border-dark-700 dark:bg-dark-800">
          <tr>
            <th class="px-3 py-2 text-left font-medium text-gray-500 dark:text-gray-400">{{ t('admin.blindbox.deliveryColSource') }}</th>
            <th class="px-3 py-2 text-left font-medium text-gray-500 dark:text-gray-400">{{ t('admin.blindbox.deliveryColUser') }}</th>
            <th class="px-3 py-2 text-left font-medium text-gray-500 dark:text-gray-400">{{ t('admin.blindbox.deliveryColReward') }}</th>
            <th class="px-3 py-2 text-left font-medium text-gray-500 dark:text-gray-400">{{ t('admin.blindbox.deliveryColStatus') }}</th>
            <th class="px-3 py-2 text-left font-medium text-gray-500 dark:text-gray-400">{{ t('admin.blindbox.deliveryColAttempts') }}</th>
            <th class="px-3 py-2 text-left font-medium text-gray-500 dark:text-gray-400">{{ t('admin.blindbox.deliveryColTime') }}</th>
            <th class="px-3 py-2 text-right font-medium text-gray-500 dark:text-gray-400">{{ t('admin.blindbox.colActions') }}</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-50 dark:divide-dark-800/50">
          <tr v-for="delivery in deliveries" :key="delivery.id" class="align-top hover:bg-gray-50 dark:hover:bg-dark-800/50">
            <td class="px-3 py-2">
              <div class="font-medium text-gray-900 dark:text-white">#{{ delivery.source_id }}</div>
              <div class="text-xs text-gray-500">{{ delivery.source_type }}</div>
            </td>
            <td class="px-3 py-2 text-gray-700 dark:text-gray-300">#{{ delivery.user_id }}</td>
            <td class="px-3 py-2">
              <div class="text-gray-700 dark:text-gray-300">{{ rewardTypeLabel(delivery.reward_type) }} · {{ formatRewardValue(delivery) }}</div>
              <div v-if="delivery.reward_detail" class="mt-0.5 max-w-56 truncate text-xs text-gray-500" :title="delivery.reward_detail">{{ delivery.reward_detail }}</div>
            </td>
            <td class="px-3 py-2">
              <span class="inline-flex rounded-full px-2 py-0.5 text-xs font-medium" :class="statusClass(delivery.status)">
                {{ t(`admin.blindbox.deliveryStatus.${delivery.status}`) }}
              </span>
              <div v-if="delivery.last_error" class="mt-1 max-w-64 whitespace-normal text-xs text-red-600 dark:text-red-400" :title="delivery.last_error">
                {{ delivery.last_error }}
              </div>
            </td>
            <td class="px-3 py-2 text-gray-600 dark:text-gray-300">{{ delivery.attempts }}</td>
            <td class="px-3 py-2 text-xs text-gray-500">{{ formatDate(delivery.updated_at) }}</td>
            <td class="px-3 py-2 text-right">
              <div v-if="delivery.status === 'failed'" class="flex justify-end gap-3">
                <button type="button" class="text-xs font-medium text-primary-600 hover:text-primary-700 disabled:opacity-50 dark:text-primary-400" :disabled="actingId === delivery.id" @click="pendingRetry = delivery">
                  {{ t('admin.blindbox.deliveryRetry') }}
                </button>
                <button type="button" class="text-xs font-medium text-amber-700 hover:text-amber-800 disabled:opacity-50 dark:text-amber-400" :disabled="actingId === delivery.id" @click="openCompensation(delivery)">
                  {{ t('admin.blindbox.deliveryCompensate') }}
                </button>
              </div>
              <span v-else class="text-xs text-gray-400">-</span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="total > pageSize" class="mt-3 flex items-center justify-between text-xs text-gray-500">
      <span>{{ t('admin.blindbox.deliveryTotal', { total }) }}</span>
      <div class="flex items-center gap-2">
        <button type="button" class="btn btn-secondary px-3 py-1.5 text-xs" :disabled="page <= 1 || loading" @click="changePage(page - 1)">
          {{ t('common.back') }}
        </button>
        <span>{{ page }} / {{ totalPages }}</span>
        <button type="button" class="btn btn-secondary px-3 py-1.5 text-xs" :disabled="page >= totalPages || loading" @click="changePage(page + 1)">
          {{ t('common.next') }}
        </button>
      </div>
    </div>

    <ConfirmDialog
      :show="!!pendingRetry"
      :title="t('admin.blindbox.deliveryRetryTitle')"
      :message="t('admin.blindbox.deliveryRetryMessage')"
      :confirm-text="t('admin.blindbox.deliveryRetry')"
      @confirm="confirmRetry"
      @cancel="pendingRetry = null"
    />

    <div v-if="pendingCompensation" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50" @click.self="closeCompensation">
      <div class="card mx-4 w-full max-w-md space-y-4 p-6">
        <div>
          <h3 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.blindbox.deliveryCompensateTitle') }}</h3>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.blindbox.deliveryCompensateMessage') }}</p>
        </div>
        <div>
          <label for="reward-compensation-reason" class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.blindbox.deliveryCompensateReason') }}</label>
          <textarea id="reward-compensation-reason" v-model="compensationReason" rows="3" maxlength="500" class="input resize-none" />
        </div>
        <div class="flex justify-end gap-3">
          <button type="button" class="btn btn-secondary" @click="closeCompensation">{{ t('common.cancel') }}</button>
          <button type="button" class="btn btn-primary" :disabled="!compensationReason.trim() || actingId === pendingCompensation.id" @click="confirmCompensation">
            {{ t('admin.blindbox.deliveryConfirmCompensate') }}
          </button>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import {
  rewardDeliveriesAPI,
  type RewardDelivery,
  type RewardDeliveryStatus,
} from '@/custom/modules/activity/api/admin/rewardDeliveries'

const { t, locale } = useI18n()
const deliveries = ref<RewardDelivery[]>([])
const statusFilter = ref<RewardDeliveryStatus | ''>('failed')
const loading = ref(false)
const operationError = ref('')
const actingId = ref<number | null>(null)
const pendingRetry = ref<RewardDelivery | null>(null)
const pendingCompensation = ref<RewardDelivery | null>(null)
const compensationReason = ref('')
const total = ref(0)
const page = ref(1)
const pageSize = 10
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

function statusClass(status: RewardDeliveryStatus): string {
  const classes: Record<RewardDeliveryStatus, string> = {
    pending: 'bg-blue-50 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300',
    delivering: 'bg-amber-50 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300',
    delivered: 'bg-green-50 text-green-700 dark:bg-green-900/30 dark:text-green-300',
    failed: 'bg-red-50 text-red-700 dark:bg-red-900/30 dark:text-red-300',
    compensated: 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300',
  }
  return classes[status]
}

function rewardTypeLabel(type: string): string {
  const key: Record<string, string> = {
    balance: 'rewardBalance',
    concurrency: 'rewardConcurrency',
    subscription: 'rewardSubscription',
    invitation_code: 'rewardInvitation',
  }
  return key[type] ? t(`admin.blindbox.${key[type]}`) : type
}

function formatRewardValue(delivery: RewardDelivery): string {
  if (delivery.reward_type === 'balance') return delivery.reward_value.toFixed(2)
  if (delivery.reward_type === 'subscription') return `${Math.round(delivery.reward_value)} ${t('admin.blindbox.days')}`
  return String(delivery.reward_value)
}

function formatDate(value: string): string {
  return new Intl.DateTimeFormat(locale.value, { dateStyle: 'short', timeStyle: 'short' }).format(new Date(value))
}

async function loadDeliveries() {
  loading.value = true
  operationError.value = ''
  try {
    const result = await rewardDeliveriesAPI.list({
      status: statusFilter.value,
      source_type: 'checkin_blindbox',
      page: page.value,
      page_size: pageSize,
    })
    deliveries.value = result.items || []
    total.value = result.total || 0
  } catch (error: any) {
    operationError.value = error?.message || error?.response?.data?.message || t('admin.blindbox.deliveryLoadFailed')
  } finally {
    loading.value = false
  }
}

function resetAndLoad() {
  page.value = 1
  loadDeliveries()
}

function changePage(nextPage: number) {
  page.value = nextPage
  loadDeliveries()
}

async function confirmRetry() {
  const delivery = pendingRetry.value
  if (!delivery) return
  pendingRetry.value = null
  actingId.value = delivery.id
  operationError.value = ''
  try {
    await rewardDeliveriesAPI.retry(delivery.id)
    await loadDeliveries()
  } catch (error: any) {
    operationError.value = error?.message || error?.response?.data?.message || t('admin.blindbox.deliveryRetryFailed')
  } finally {
    actingId.value = null
  }
}

function openCompensation(delivery: RewardDelivery) {
  pendingCompensation.value = delivery
  compensationReason.value = ''
}

function closeCompensation() {
  pendingCompensation.value = null
  compensationReason.value = ''
}

async function confirmCompensation() {
  const delivery = pendingCompensation.value
  const reason = compensationReason.value.trim()
  if (!delivery || !reason) return
  actingId.value = delivery.id
  operationError.value = ''
  try {
    await rewardDeliveriesAPI.compensate(delivery.id, reason)
    closeCompensation()
    await loadDeliveries()
  } catch (error: any) {
    operationError.value = error?.message || error?.response?.data?.message || t('admin.blindbox.deliveryCompensateFailed')
  } finally {
    actingId.value = null
  }
}

onMounted(loadDeliveries)
</script>
