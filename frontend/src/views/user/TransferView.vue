<template>
  <AppLayout>
    <div class="mx-auto max-w-4xl space-y-6">
      <section class="rounded-2xl bg-gradient-to-br from-teal-500 to-teal-600 px-6 py-8 text-center text-white shadow-lg shadow-teal-500/15 sm:px-8">
        <div class="mx-auto flex h-16 w-16 items-center justify-center rounded-2xl bg-white/15 ring-1 ring-inset ring-white/15">
          <Icon name="dollar" size="xl" />
        </div>
        <p class="mt-4 text-sm text-white/85">{{ t('transfer.currentBalance') }}</p>
        <h1 class="mt-1 truncate text-3xl font-bold tabular-nums sm:text-4xl" :title="money(currentBalance)">{{ money(currentBalance) }}</h1>
      </section>

      <section class="grid gap-4 sm:grid-cols-3">
        <article v-for="item in statCards" :key="item.label" class="card flex flex-col items-center p-4 text-center">
          <span class="flex h-9 w-9 items-center justify-center rounded-lg" :class="item.iconClass"><Icon :name="item.icon" size="sm" /></span>
          <strong class="mt-3 max-w-full truncate text-xl font-bold tabular-nums text-gray-900 dark:text-white" :title="money(item.value)">{{ money(item.value) }}</strong>
          <span class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ item.label }}</span>
        </article>
      </section>

      <form class="card space-y-5 p-5 sm:p-6" @submit.prevent="prepareTransfer">
        <label class="block">
          <span class="input-label">{{ t('transfer.receiverId') }}</span>
          <span class="relative block">
            <Icon name="search" size="sm" class="pointer-events-none absolute left-4 top-1/2 -translate-y-1/2 text-gray-400" />
            <input
              v-model.trim="receiverQuery"
              class="input pl-11 pr-24"
              type="text"
              autocomplete="off"
              :placeholder="t('transfer.receiverPlaceholder')"
              required
              @input="resetReceiver"
              @blur="resolveReceiver"
            />
            <button type="button" class="absolute right-2 top-1/2 -translate-y-1/2 rounded-lg px-2.5 py-1.5 text-xs font-medium text-primary-600 hover:bg-primary-50 dark:text-primary-400 dark:hover:bg-primary-900/20" :disabled="receiverLoading || !receiverQuery" @mousedown.prevent @click="resolveReceiver">
              {{ receiverLoading ? t('common.loading') : t('common.search') }}
            </button>
          </span>
          <span v-if="receiver" data-testid="resolved-receiver" class="mt-2 flex items-center gap-1.5 text-xs text-emerald-600 dark:text-emerald-400"><Icon name="checkCircle" size="xs" />{{ t('transfer.receiverResolved', { recipient: receiver.receiver_display }) }}</span>
        </label>

        <label class="block">
          <span class="input-label">{{ t('transfer.amount') }}</span>
          <span class="relative block">
            <Icon name="dollar" size="sm" class="pointer-events-none absolute left-4 top-1/2 -translate-y-1/2 text-gray-400" />
            <input v-model.number="form.amount" class="input pl-11" type="number" min="0.01" step="0.01" required @input="resetPreview" />
          </span>
          <span class="mt-2 flex items-center justify-between gap-3 text-xs">
            <button type="button" class="font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400" :disabled="submitting || form.amount <= 0 || !receiverQuery" @click="calculatePreview">{{ t('transfer.calculateFee') }}</button>
            <span class="truncate text-gray-400 dark:text-dark-500">{{ t('transfer.available', { amount: money(currentBalance) }) }}</span>
          </span>
        </label>

        <div v-if="preview" class="rounded-xl border border-primary-100 bg-primary-50/60 p-4 text-sm dark:border-primary-900/50 dark:bg-primary-900/10">
          <div class="flex justify-between gap-4"><span class="text-gray-500 dark:text-dark-400">{{ t('transfer.feePreview') }}</span><strong>{{ money(preview.fee) }}</strong></div>
          <div class="mt-2 flex justify-between gap-4"><span class="text-gray-500 dark:text-dark-400">{{ t('transfer.total') }}</span><strong>{{ money(preview.gross_amount ?? form.amount + preview.fee) }}</strong></div>
          <p v-if="preview.daily_remaining_amount != null || preview.daily_remaining_count != null" class="mt-2 text-xs text-gray-500 dark:text-dark-400">{{ t('transfer.dailyRemaining', { amount: money(preview.daily_remaining_amount || 0), count: preview.daily_remaining_count || 0 }) }}</p>
        </div>

        <label class="block">
          <span class="input-label">{{ t('transfer.memo') }}</span>
          <span class="relative block">
            <Icon name="chat" size="sm" class="pointer-events-none absolute left-4 top-1/2 -translate-y-1/2 text-gray-400" />
            <input v-model.trim="form.memo" class="input pl-11" type="text" maxlength="200" :placeholder="t('transfer.memoPlaceholder')" />
          </span>
        </label>

        <p v-if="formError" class="text-sm text-red-600 dark:text-red-400">{{ formError }}</p>
        <button class="btn btn-primary w-full" type="submit" :disabled="submitting || !receiverIsCurrent || form.amount <= 0">
          <Icon name="checkCircle" size="sm" />
          {{ submitting ? t('common.processing') : t('transfer.submit') }}
        </button>
      </form>

      <section class="card overflow-hidden p-0">
        <div class="flex flex-col gap-3 border-b border-gray-100 px-5 py-4 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
          <div><h2 class="font-semibold text-gray-900 dark:text-white">{{ t('transfer.history') }}</h2><p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ t('transfer.historyHint') }}</p></div>
          <div class="flex items-center gap-2">
            <div class="flex rounded-lg bg-gray-100 p-0.5 dark:bg-dark-800">
              <button v-for="role in roles" :key="role.value" type="button" class="rounded-md px-3 py-1.5 text-xs font-medium" :class="historyRole === role.value ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white' : 'text-gray-500 dark:text-dark-400'" @click="changeRole(role.value)">{{ role.label }}</button>
            </div>
            <button class="rounded-lg p-2 text-gray-400 hover:bg-gray-100 hover:text-gray-700 dark:hover:bg-dark-800 dark:hover:text-white" type="button" :title="t('common.refresh')" @click="loadHistory"><Icon name="refresh" size="sm" /></button>
          </div>
        </div>
        <div v-if="historyLoading" class="flex min-h-48 items-center justify-center"><LoadingSpinner /></div>
        <div v-else-if="historyError" class="py-12 text-center"><p class="text-sm text-red-600 dark:text-red-400">{{ historyError }}</p><button class="btn btn-secondary btn-sm mt-3" @click="loadHistory">{{ t('common.retry') }}</button></div>
        <div v-else-if="history.length === 0" class="py-12 text-center text-sm text-gray-500 dark:text-dark-400">{{ t('transfer.emptyHistory') }}</div>
        <div v-else class="divide-y divide-gray-100 dark:divide-dark-700">
          <div v-for="record in history" :key="record.id" class="flex items-center justify-between gap-4 px-5 py-4">
            <div class="min-w-0">
              <p class="truncate text-sm font-medium text-gray-900 dark:text-white">{{ historyRole === 'sender' ? t('transfer.toUser', { id: record.receiver_id }) : t('transfer.fromUser', { id: record.sender_id }) }}</p>
              <p class="mt-1 truncate text-xs text-gray-500 dark:text-dark-400">{{ transferTypeLabel(record.transfer_type) }} · #{{ record.id }} · {{ dateTime(record.created_at) }}<span v-if="record.memo"> · {{ record.memo }}</span></p>
            </div>
            <div class="shrink-0 text-right"><p class="font-semibold" :class="historyRole === 'sender' ? 'text-red-600 dark:text-red-400' : 'text-emerald-600 dark:text-emerald-400'">{{ historyRole === 'sender' ? '-' : '+' }}{{ money(record.amount) }}</p><p class="mt-1 text-xs text-gray-500">{{ t(`transfer.status.${record.status}`, record.status) }}</p></div>
          </div>
        </div>
        <Pagination v-if="historyTotal > pageSize" :page="page" :page-size="pageSize" :total="historyTotal" :show-page-size-selector="false" @update:page="changePage" />
      </section>

      <ConfirmDialog :show="confirming" :title="t('transfer.confirmTitle')" :message="t('transfer.confirmMessage')" :confirm-text="t('transfer.submit')" @confirm="submitTransfer" @cancel="confirming = false">
        <div v-if="preview && receiver" class="rounded-xl bg-gray-50 p-3 text-sm dark:bg-dark-800"><p>{{ t('transfer.confirmRecipient', { recipient: receiver.receiver_display }) }}</p><p class="mt-2">{{ t('transfer.confirmAmount', { amount: money(form.amount), fee: money(preview.fee), total: money(preview.gross_amount ?? form.amount + preview.fee) }) }}</p></div>
      </ConfirmDialog>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Pagination from '@/components/common/Pagination.vue'
import Icon from '@/components/icons/Icon.vue'
import {
  createActivityIdempotencyKey,
  getTransferHistory,
  getTransferStats,
  resolveTransferReceiver,
  transferBalance,
  validateTransfer,
  type TransferReceiver,
  type TransferRecord,
  type TransferStats,
  type TransferValidation,
} from '@/api/transfer'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { activityErrorMessage } from '@/utils/activityError'

const { t, locale } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const stats = ref<TransferStats>({ total_sent: 0, total_received: 0, total_fee_paid: 0 })
const receiverQuery = ref('')
const receiver = ref<TransferReceiver | null>(null)
const receiverResolvedQuery = ref('')
const receiverLoading = ref(false)
const form = reactive({ amount: 0, memo: '' })
const preview = ref<TransferValidation | null>(null)
const formError = ref('')
const submitting = ref(false)
const confirming = ref(false)
const history = ref<TransferRecord[]>([])
const historyRole = ref<'sender' | 'receiver'>('sender')
const historyLoading = ref(false)
const historyError = ref('')
const historyTotal = ref(0)
const page = ref(1)
const pageSize = 10
let receiverRequestToken = 0
const transferAttempt = ref<{ signature: string; key: string } | null>(null)

const currentBalance = computed(() => Number(authStore.user?.balance || 0))
const receiverIsCurrent = computed(() => !!receiver.value && receiverResolvedQuery.value === receiverQuery.value.trim())
const statCards = computed(() => [
  { label: t('transfer.totalSent'), value: stats.value.total_sent, icon: 'arrowUp' as const, iconClass: 'bg-blue-50 text-blue-500 dark:bg-blue-900/20 dark:text-blue-400' },
  { label: t('transfer.totalReceived'), value: stats.value.total_received, icon: 'arrowDown' as const, iconClass: 'bg-emerald-50 text-emerald-500 dark:bg-emerald-900/20 dark:text-emerald-400' },
  { label: t('transfer.totalFee'), value: stats.value.total_fee_paid, icon: 'creditCard' as const, iconClass: 'bg-orange-50 text-orange-500 dark:bg-orange-900/20 dark:text-orange-400' },
])
const roles = computed(() => [
  { value: 'sender' as const, label: t('transfer.sent') },
  { value: 'receiver' as const, label: t('transfer.received') },
])

function money(value: number) {
  return `$${Number(value || 0).toFixed(2)}`
}

function dateTime(value: string) {
  return new Intl.DateTimeFormat(locale.value, { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value))
}

function errorMessage(error: unknown, fallback: string) {
  return activityErrorMessage(error, t, fallback)
}

function transferTypeLabel(type: string) {
  return t(`transfer.transferTypes.${type}`, type)
}

function resetPreview() {
  preview.value = null
  formError.value = ''
}

function resetReceiver() {
  receiverRequestToken += 1
  receiverLoading.value = false
  receiver.value = null
  receiverResolvedQuery.value = ''
  resetPreview()
}

async function resolveReceiver() {
  const query = receiverQuery.value.trim()
  if (!query) return false
  const requestToken = ++receiverRequestToken
  receiverLoading.value = true
  formError.value = ''
  try {
    const result = await resolveTransferReceiver(query)
    if (requestToken !== receiverRequestToken || receiverQuery.value.trim() !== query) return false
    receiver.value = result
    receiverResolvedQuery.value = query
    return true
  } catch (error) {
    if (requestToken !== receiverRequestToken || receiverQuery.value.trim() !== query) return false
    receiver.value = null
    receiverResolvedQuery.value = ''
    formError.value = errorMessage(error, t('transfer.receiverSearchFailed'))
    return false
  } finally {
    if (requestToken === receiverRequestToken) receiverLoading.value = false
  }
}

async function calculatePreview() {
  if (!receiverIsCurrent.value && !(await resolveReceiver())) return false
  if (!receiver.value || !receiverIsCurrent.value || form.amount <= 0) return false
  formError.value = ''
  submitting.value = true
  try {
    preview.value = await validateTransfer(receiver.value.receiver_id, form.amount)
    return true
  } catch (error) {
    preview.value = null
    formError.value = errorMessage(error, t('transfer.validationFailed'))
    return false
  } finally {
    submitting.value = false
  }
}

async function loadStats() {
  try {
    stats.value = await getTransferStats()
  } catch {
    // The form and history remain usable when summary loading fails.
  }
}

async function loadHistory() {
  historyLoading.value = true
  historyError.value = ''
  try {
    const result = await getTransferHistory({ role: historyRole.value, page: page.value, page_size: pageSize })
    history.value = result.items || []
    historyTotal.value = result.total || 0
  } catch (error) {
    historyError.value = errorMessage(error, t('transfer.historyFailed'))
  } finally {
    historyLoading.value = false
  }
}

function changeRole(role: 'sender' | 'receiver') {
  historyRole.value = role
  page.value = 1
  void loadHistory()
}

function changePage(value: number) {
  page.value = value
  void loadHistory()
}

async function prepareTransfer() {
  if (await calculatePreview()) confirming.value = true
}

async function submitTransfer() {
  if (!receiver.value || !receiverIsCurrent.value) {
    formError.value = t('transfer.receiverSearchFailed')
    confirming.value = false
    return
  }
  confirming.value = false
  submitting.value = true
  const signature = JSON.stringify([receiver.value.receiver_id, form.amount, form.memo.trim()])
  if (transferAttempt.value?.signature !== signature) {
    transferAttempt.value = { signature, key: createActivityIdempotencyKey('balance-transfer') }
  }
  try {
    const result = await transferBalance(receiver.value.receiver_id, form.amount, form.memo || undefined, transferAttempt.value.key)
    appStore.showSuccess(t('transfer.successWithId', { id: result.id }))
    transferAttempt.value = null
    receiverQuery.value = ''
    receiver.value = null
    receiverResolvedQuery.value = ''
    form.amount = 0
    form.memo = ''
    preview.value = null
    await Promise.all([loadStats(), loadHistory(), authStore.refreshUser()])
  } catch (error) {
    formError.value = errorMessage(error, t('transfer.failed'))
    appStore.showError(formError.value)
  } finally {
    submitting.value = false
  }
}

onMounted(() => {
  void loadStats()
  void loadHistory()
  void authStore.refreshUser().catch(() => undefined)
})
</script>
