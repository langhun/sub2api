<template>
  <AppLayout>
    <div class="mx-auto max-w-6xl space-y-6">
      <header><h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('transfer.title') }}</h1><p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('transfer.description') }}</p></header>

      <section class="grid gap-4 sm:grid-cols-3">
        <div v-for="item in statCards" :key="item.label" class="card p-5"><p class="text-sm text-gray-500 dark:text-dark-400">{{ item.label }}</p><p class="mt-2 text-xl font-semibold text-gray-900 dark:text-white">{{ money(item.value) }}</p></div>
      </section>

      <section class="grid gap-6 lg:grid-cols-[minmax(320px,0.75fr)_minmax(0,1.25fr)]">
        <form class="card space-y-4 p-5" @submit.prevent="prepareTransfer">
          <div><h2 class="font-semibold text-gray-900 dark:text-white">{{ t('transfer.newTransfer') }}</h2><p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ t('transfer.irreversible') }}</p></div>
          <label class="block"><span class="input-label">{{ t('transfer.receiverId') }}</span><input v-model.number="form.receiverId" class="input" type="number" min="1" required @input="resetPreview" /></label>
          <label class="block"><span class="input-label">{{ t('transfer.amount') }}</span><input v-model.number="form.amount" class="input" type="number" min="0.01" step="0.01" required @input="resetPreview" /></label>
          <label class="block"><span class="input-label">{{ t('transfer.memo') }}</span><input v-model.trim="form.memo" class="input" type="text" maxlength="200" /></label>
          <div v-if="preview" class="rounded-lg border border-gray-200 p-4 text-sm dark:border-dark-700">
            <div v-if="preview.receiver_display" class="mb-3 flex justify-between"><span class="text-gray-500">{{ t('transfer.recipient') }}</span><strong>{{ preview.receiver_display }}</strong></div>
            <div class="flex justify-between text-gray-500 dark:text-dark-400"><span>{{ t('transfer.feePreview') }}</span><span>{{ money(preview.fee) }}</span></div>
            <div class="mt-2 flex justify-between font-medium"><span>{{ t('transfer.total') }}</span><span>{{ money(preview.gross_amount ?? form.amount + preview.fee) }}</span></div>
          </div>
          <p v-if="formError" class="text-sm text-red-600 dark:text-red-400">{{ formError }}</p>
          <button class="btn btn-primary w-full" type="submit" :disabled="submitting || form.receiverId < 1 || form.amount <= 0"><Icon name="arrowRight" size="sm" />{{ submitting ? t('common.processing') : t('transfer.continue') }}</button>
        </form>

        <div class="card overflow-hidden p-0">
          <div class="flex items-center justify-between border-b border-gray-100 px-5 py-4 dark:border-dark-700"><div><h2 class="font-semibold text-gray-900 dark:text-white">{{ t('transfer.history') }}</h2><p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ t('transfer.historyHint') }}</p></div><button class="btn btn-ghost" type="button" @click="loadHistory"><Icon name="refresh" size="sm" /></button></div>
          <div class="grid grid-cols-2 border-b border-gray-100 p-1 dark:border-dark-700"><button v-for="role in roles" :key="role.value" class="px-3 py-2 text-sm font-medium" :class="historyRole === role.value ? 'text-primary-600 dark:text-primary-400' : 'text-gray-500'" @click="changeRole(role.value)">{{ role.label }}</button></div>
          <div v-if="historyLoading" class="flex min-h-48 items-center justify-center"><LoadingSpinner /></div>
          <div v-else-if="historyError" class="py-12 text-center"><p class="text-sm text-red-600 dark:text-red-400">{{ historyError }}</p><button class="btn btn-secondary mt-3" @click="loadHistory">{{ t('common.retry') }}</button></div>
          <div v-else-if="history.length === 0" class="py-12 text-center text-sm text-gray-500 dark:text-dark-400">{{ t('transfer.emptyHistory') }}</div>
          <div v-else class="divide-y divide-gray-100 dark:divide-dark-700">
            <div v-for="record in history" :key="record.id" class="flex items-center justify-between gap-4 px-5 py-4">
              <div class="min-w-0"><p class="truncate text-sm font-medium text-gray-900 dark:text-white">{{ historyRole === 'sent' ? t('transfer.toUser', { id: record.receiver_id }) : t('transfer.fromUser', { id: record.sender_id }) }}</p><p class="mt-1 text-xs text-gray-500 dark:text-dark-400">#{{ record.id }} · {{ dateTime(record.created_at) }}<span v-if="record.memo"> · {{ record.memo }}</span></p></div>
              <div class="shrink-0 text-right"><p class="font-semibold" :class="historyRole === 'sent' ? 'text-red-600 dark:text-red-400' : 'text-green-600 dark:text-green-400'">{{ historyRole === 'sent' ? '-' : '+' }}{{ money(record.amount) }}</p><p class="mt-1 text-xs text-gray-500">{{ t(`transfer.status.${record.status}`, record.status) }}</p></div>
            </div>
          </div>
          <Pagination v-if="historyTotal > pageSize" :page="page" :page-size="pageSize" :total="historyTotal" :show-page-size-selector="false" @update:page="changePage" />
        </div>
      </section>

      <ConfirmDialog :show="confirming" :title="t('transfer.confirmTitle')" :message="t('transfer.confirmMessage')" :confirm-text="t('transfer.submit')" @confirm="submitTransfer" @cancel="confirming = false">
        <div v-if="preview" class="rounded-lg bg-gray-50 p-3 text-sm dark:bg-dark-800"><p>{{ t('transfer.confirmRecipient', { recipient: preview.receiver_display || `#${form.receiverId}` }) }}</p><p class="mt-2">{{ t('transfer.confirmAmount', { amount: money(form.amount), fee: money(preview.fee), total: money(preview.gross_amount ?? form.amount + preview.fee) }) }}</p></div>
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
import { getTransferHistory, getTransferStats, transferBalance, validateTransfer, type TransferRecord, type TransferStats, type TransferValidation } from '@/api/transfer'
import { useAppStore } from '@/stores/app'
import { activityErrorMessage } from '@/utils/activityError'

const { t, locale } = useI18n(); const appStore = useAppStore()
const stats = ref<TransferStats>({ total_sent: 0, total_received: 0, total_fee_paid: 0 })
const form = reactive({ receiverId: 0, amount: 0, memo: '' }); const preview = ref<TransferValidation | null>(null)
const formError = ref(''); const submitting = ref(false); const confirming = ref(false)
const history = ref<TransferRecord[]>([]); const historyRole = ref<'sent' | 'received'>('sent'); const historyLoading = ref(false); const historyError = ref(''); const historyTotal = ref(0); const page = ref(1); const pageSize = 10
const statCards = computed(() => [{ label: t('transfer.totalSent'), value: stats.value.total_sent }, { label: t('transfer.totalReceived'), value: stats.value.total_received }, { label: t('transfer.totalFee'), value: stats.value.total_fee_paid }])
const roles = computed(() => [{ value: 'sent' as const, label: t('transfer.sent') }, { value: 'received' as const, label: t('transfer.received') }])
function money(value: number) { return Number(value || 0).toFixed(2) } function dateTime(value: string) { return new Intl.DateTimeFormat(locale.value, { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value)) }
function resetPreview() { preview.value = null; formError.value = '' }
function errorMessage(error: unknown, fallback: string) { return activityErrorMessage(error, t, fallback) }
async function loadStats() { try { stats.value = await getTransferStats() } catch { /* history remains usable */ } }
async function loadHistory() { historyLoading.value = true; historyError.value = ''; try { const result = await getTransferHistory({ role: historyRole.value, page: page.value, page_size: pageSize }); history.value = result.items || []; historyTotal.value = result.total || 0 } catch (error) { historyError.value = errorMessage(error, t('transfer.historyFailed')) } finally { historyLoading.value = false } }
function changeRole(role: 'sent' | 'received') { historyRole.value = role; page.value = 1; void loadHistory() } function changePage(value: number) { page.value = value; void loadHistory() }
async function prepareTransfer() { formError.value = ''; submitting.value = true; try { preview.value = await validateTransfer(form.receiverId, form.amount); confirming.value = true } catch (error) { formError.value = errorMessage(error, t('transfer.validationFailed')) } finally { submitting.value = false } }
async function submitTransfer() { confirming.value = false; submitting.value = true; try { const result = await transferBalance(form.receiverId, form.amount, form.memo || undefined); appStore.showSuccess(t('transfer.successWithId', { id: result.id })); form.receiverId = 0; form.amount = 0; form.memo = ''; preview.value = null; await Promise.all([loadStats(), loadHistory()]) } catch (error) { formError.value = errorMessage(error, t('transfer.failed')); appStore.showError(formError.value) } finally { submitting.value = false } }
onMounted(() => { void loadStats(); void loadHistory() })
</script>
