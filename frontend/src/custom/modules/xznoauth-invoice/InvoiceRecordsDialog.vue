<template>
  <BaseDialog :show="show" :title="t('invoices.recordsTitle')" width="extra-wide" @close="emit('close')">
    <div class="space-y-4">
      <div class="flex items-center justify-between gap-3">
        <span class="text-sm text-gray-500">{{ t('invoices.recordCount', { count: page.total }) }}</span>
        <button class="btn btn-secondary btn-sm" type="button" :disabled="loading" @click="load"><Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" /> {{ t('invoices.refresh') }}</button>
      </div>

      <div v-if="error && !loading" class="rounded-lg border border-dashed border-gray-300 p-10 text-center text-sm text-gray-500 dark:border-dark-600">
        <p>{{ t('invoices.loadFailed') }}</p>
        <button class="btn btn-secondary btn-sm mt-3" type="button" @click="load">{{ t('common.retry') }}</button>
      </div>
      <div v-else-if="!loading && !items.length" class="rounded-lg border border-dashed border-gray-300 p-10 text-center text-sm text-gray-500 dark:border-dark-600">{{ t('invoices.empty') }}</div>
      <div v-else class="overflow-x-auto rounded-lg border border-gray-200 dark:border-dark-700">
        <table class="min-w-[760px] w-full text-left text-sm">
          <thead class="border-b border-gray-200 bg-gray-50 text-xs text-gray-500 dark:border-dark-700 dark:bg-dark-800"><tr><th class="px-4 py-3">{{ t('invoices.orderSet') }}</th><th class="px-4 py-3">{{ t('invoices.total') }}</th><th class="px-4 py-3">{{ t('invoices.status') }}</th><th class="px-4 py-3">{{ t('invoices.createdAt') }}</th><th class="px-4 py-3 text-right">{{ t('invoices.actions') }}</th></tr></thead>
          <tbody><tr v-for="item in items" :key="item.id" class="border-b border-gray-100 last:border-0 dark:border-dark-800"><td class="max-w-[240px] truncate px-4 py-3 font-mono text-xs">{{ item.orderNos?.join(', ') || '-' }}</td><td class="px-4 py-3">{{ item.totalAmount || '-' }} {{ item.currency || '' }}</td><td class="px-4 py-3"><span class="rounded-full bg-gray-100 px-2 py-1 text-xs dark:bg-dark-700">{{ t(`invoices.statuses.${item.status}`) }}</span><p v-if="item.reviewNote" class="mt-1 text-xs text-gray-500">{{ item.reviewNote }}</p></td><td class="px-4 py-3 text-xs text-gray-500">{{ formatDate(item.createdAt) }}</td><td class="px-4 py-3"><div class="flex justify-end gap-2"><button v-if="item.status === 'awaiting_tax_payment'" class="btn btn-secondary btn-sm" type="button" :disabled="checkoutId === item.id" @click="continuePayment(item)">{{ t('invoices.continuePayment') }}</button><button v-if="['awaiting_tax_payment','pending','approved'].includes(item.status)" class="btn btn-ghost btn-sm text-red-600" type="button" :disabled="cancelId === item.id" @click="cancel(item)">{{ t('invoices.cancel') }}</button><button v-if="item.status === 'completed' && item.hasPdf !== false" class="btn btn-ghost btn-sm" type="button" :disabled="previewId === item.id" @click="preview(item)">{{ t('invoices.preview') }}</button><button v-if="item.status === 'completed' && item.hasPdf !== false" class="btn btn-secondary btn-sm" type="button" :disabled="downloadId === item.id" @click="download(item)">{{ t('invoices.download') }}</button></div></td></tr></tbody>
        </table>
      </div>
      <Pagination v-if="page.total > 0" :page="page.number" :total="page.total" :page-size="page.size" @update:page="changePage" @update:page-size="changeSize" />
    </div>

    <BaseDialog :show="!!checkout" :title="t('invoices.paymentTitle')" width="normal" :z-index="60" @close="closeCheckout">
      <div v-if="checkout" class="space-y-4"><p class="text-sm text-gray-600 dark:text-gray-300">{{ t('invoices.paymentHint') }}</p><div v-for="(channel, key) in checkout.taxPayments" :key="key" class="flex items-center justify-between gap-3"><span class="capitalize">{{ key }}</span><button class="btn btn-primary" type="button" @click="pay(channel)">{{ t('invoices.pay') }}</button></div></div>
    </BaseDialog>
    <BaseDialog :show="!!previewUrl" :title="t('invoices.preview')" width="wide" :z-index="60" @close="closePreview"><iframe v-if="previewUrl" :src="previewUrl" class="h-[70vh] w-full border-0" :title="t('invoices.preview')" /></BaseDialog>
  </BaseDialog>
</template>

<script setup lang="ts">
import { onBeforeUnmount, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Pagination from '@/components/common/Pagination.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores'
import { extractI18nErrorMessage } from '@/utils/apiError'
import { invoiceAPI, type InvoiceApplication, type TaxCheckout, type TaxPayment } from './api'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{ close: [] }>()
const { t } = useI18n()
const appStore = useAppStore()
const loading = ref(false)
const error = ref(false)
const items = ref<InvoiceApplication[]>([])
const checkoutId = ref<number | null>(null)
const cancelId = ref<number | null>(null)
const downloadId = ref<number | null>(null)
const previewId = ref<number | null>(null)
const checkout = ref<TaxCheckout | null>(null)
const previewUrl = ref<string | null>(null)
const page = reactive({ number: 1, size: 20, total: 0 })
let pollTimer: number | undefined

async function load() {
  loading.value = true
  error.value = false
  try {
    const result = (await invoiceAPI.listInvoices({ page: page.number, pageSize: page.size })).data
    items.value = result.items || []
    page.total = result.total || 0
  } catch (loadError) {
    error.value = true
    appStore.showError(extractI18nErrorMessage(loadError, t, 'invoices', t('invoices.loadFailed')))
  } finally { loading.value = false }
}
function changePage(value: number) { page.number = value; void load() }
function changeSize(value: number) { page.number = 1; page.size = value; void load() }
function formatDate(value?: string) { return value ? new Date(value).toLocaleString() : '-' }
async function continuePayment(item: InvoiceApplication) { checkoutId.value = item.id; try { checkout.value = (await invoiceAPI.getTaxPayments(item.id)).data } catch (actionError) { appStore.showError(extractI18nErrorMessage(actionError, t, 'invoices', t('invoices.loadFailed'))) } finally { checkoutId.value = null } }
async function cancel(item: InvoiceApplication) { if (!window.confirm(t('invoices.cancelConfirm'))) return; cancelId.value = item.id; try { await invoiceAPI.cancelInvoice(item.id); await load() } catch (actionError) { appStore.showError(extractI18nErrorMessage(actionError, t, 'invoices', t('invoices.loadFailed'))) } finally { cancelId.value = null } }
function pay(channel: TaxPayment) { window.open(channel.payUrl, '_blank', 'noopener,noreferrer'); if (checkout.value) void pollPayment(checkout.value.application.id, channel.taxOrderNo) }
async function pollPayment(applicationId: number, taxOrderNo: string) { for (let attempt = 0; attempt < 8; attempt += 1) { await new Promise<void>(resolve => { pollTimer = window.setTimeout(resolve, Math.min(15000, 2000 + attempt * 1500)) }); try { if ((await invoiceAPI.checkTaxPayment({ applicationId, taxOrderNo })).data.paid) { appStore.showSuccess(t('invoices.paymentComplete')); checkout.value = null; await load(); return } } catch { return } } }
function closeCheckout() { checkout.value = null; if (pollTimer) window.clearTimeout(pollTimer) }
function closePreview() { if (previewUrl.value) URL.revokeObjectURL(previewUrl.value); previewUrl.value = null }
async function preview(item: InvoiceApplication) { previewId.value = item.id; try { closePreview(); previewUrl.value = URL.createObjectURL((await invoiceAPI.downloadInvoicePDF(item.id)).data) } catch (previewError) { appStore.showError(extractI18nErrorMessage(previewError, t, 'invoices', t('invoices.loadFailed'))) } finally { previewId.value = null } }
async function download(item: InvoiceApplication) { downloadId.value = item.id; let url: string | null = null; try { url = URL.createObjectURL((await invoiceAPI.downloadInvoicePDF(item.id)).data); const anchor = document.createElement('a'); anchor.href = url; anchor.download = `invoice-${item.id}.pdf`; anchor.click() } catch (downloadError) { appStore.showError(extractI18nErrorMessage(downloadError, t, 'invoices', t('invoices.loadFailed'))) } finally { if (url) window.setTimeout(() => URL.revokeObjectURL(url as string), 1000); downloadId.value = null } }
watch(() => props.show, (show) => { if (show) { page.number = 1; void load() } })
onBeforeUnmount(() => { closeCheckout(); closePreview() })
</script>
