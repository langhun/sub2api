<template>
  <AppLayout>
    <div class="w-full space-y-5" data-glass-content>
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div><h1 class="text-2xl font-semibold text-gray-900 dark:text-white">{{ t('invoices.recordsTitle') }}</h1></div>
        <div class="flex gap-2"><button class="btn btn-secondary" type="button" :disabled="loading" @click="load"><Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" /> {{ t('invoices.refresh') }}</button><button class="btn btn-primary" type="button" @click="router.push('/invoices/apply')">{{ t('invoices.apply') }}</button></div>
      </div>
      <section class="card overflow-hidden">
        <div v-if="error && !loading" class="space-y-3 p-10 text-center text-sm text-gray-500">
          <p>{{ t('invoices.loadFailed') }}</p>
          <button class="btn btn-secondary" type="button" @click="load">{{ t('common.retry') }}</button>
        </div>
        <div v-else-if="!loading && !items.length" class="p-10 text-center text-sm text-gray-500">{{ t('invoices.empty') }}</div>
        <div v-else class="overflow-x-auto">
          <table class="min-w-[900px] w-full text-left text-sm">
            <thead class="border-b border-gray-200 bg-gray-50 text-xs uppercase text-gray-500 dark:border-dark-700 dark:bg-dark-800"><tr><th class="px-4 py-3">ID</th><th class="px-4 py-3">{{ t('invoices.orderSet') }}</th><th class="px-4 py-3">{{ t('invoices.total') }}</th><th class="px-4 py-3">{{ t('invoices.status') }}</th><th class="px-4 py-3">{{ t('invoices.createdAt') }}</th><th class="px-4 py-3 text-right">{{ t('invoices.actions') }}</th></tr></thead>
            <tbody><tr v-for="item in items" :key="item.id" class="border-b border-gray-100 dark:border-dark-800"><td class="px-4 py-3 font-mono text-xs">{{ item.id }}</td><td class="max-w-xs truncate px-4 py-3 font-mono text-xs">{{ item.orderNos?.join(', ') }}</td><td class="px-4 py-3">{{ item.totalAmount || '-' }} {{ item.currency || '' }}</td><td class="px-4 py-3"><span class="rounded-full bg-gray-100 px-2 py-1 text-xs dark:bg-dark-700">{{ t(`invoices.statuses.${item.status}`) }}</span><p v-if="item.reviewNote" class="mt-1 text-xs text-gray-500">{{ item.reviewNote }}</p></td><td class="px-4 py-3 text-xs text-gray-500">{{ formatDate(item.createdAt) }}</td><td class="px-4 py-3"><div class="flex justify-end gap-2"><button v-if="item.status === 'awaiting_tax_payment'" class="btn btn-secondary btn-sm" type="button" :disabled="checkoutId === item.id" @click="continuePayment(item)">{{ t('invoices.continuePayment') }}</button><button v-if="['awaiting_tax_payment','pending','approved'].includes(item.status)" class="btn btn-ghost btn-sm text-red-600" type="button" :disabled="cancelId === item.id" @click="cancel(item)">{{ t('invoices.cancel') }}</button><button v-if="item.status === 'completed' && item.hasPdf !== false" class="btn btn-ghost btn-sm" type="button" :disabled="previewId === item.id" @click="preview(item)">{{ t('invoices.preview') }}</button><button v-if="item.status === 'completed' && item.hasPdf !== false" class="btn btn-secondary btn-sm" type="button" :disabled="downloadId === item.id" @click="download(item)">{{ t('invoices.download') }}</button></div></td></tr></tbody>
          </table>
        </div>
      </section>
      <Pagination v-if="page.total > 0" :page="page.number" :total="page.total" :page-size="page.size" @update:page="changePage" @update:page-size="changeSize" />
    </div>
    <BaseDialog :show="!!checkout" :title="t('invoices.paymentTitle')" @close="closeCheckout">
      <div v-if="checkout" class="space-y-4"><p class="text-sm text-gray-600 dark:text-gray-300">{{ t('invoices.paymentHint') }}</p><div v-for="(channel, key) in checkout.taxPayments" :key="key" class="flex items-center justify-between gap-3"><span class="capitalize">{{ key }}</span><button class="btn btn-primary" type="button" @click="pay(channel)">{{ t('invoices.pay') }}</button></div></div>
    </BaseDialog>
    <BaseDialog :show="!!previewUrl" :title="t('invoices.preview')" width="wide" @close="closePreview">
      <iframe v-if="previewUrl" :src="previewUrl" class="h-[70vh] w-full border-0" :title="t('invoices.preview')" />
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores'
import { extractI18nErrorMessage } from '@/utils/apiError'
import { invoiceAPI, type InvoiceApplication, type TaxCheckout, type TaxPayment } from './api'

const { t } = useI18n()
const router = useRouter()
const route = useRoute()
const appStore = useAppStore()
const loading = ref(false)
const checkoutId = ref<number | null>(null)
const cancelId = ref<number | null>(null)
const downloadId = ref<number | null>(null)
const previewId = ref<number | null>(null)
const error = ref(false)
const items = ref<InvoiceApplication[]>([])
const checkout = ref<TaxCheckout | null>(null)
const previewUrl = ref<string | null>(null)
const page = reactive({ number: 1, size: 20, total: 0 })
let pollTimer: number | undefined
async function load() { loading.value = true; error.value = false; try { const result = (await invoiceAPI.listInvoices({ page: page.number, pageSize: page.size })).data; items.value = result.items || []; page.total = result.total || 0 } catch (loadError) { error.value = true; appStore.showError(extractI18nErrorMessage(loadError, t, 'invoices', t('invoices.loadFailed'))) } finally { loading.value = false } }
function changePage(value: number) { page.number = value; void load() }
function changeSize(value: number) { page.number = 1; page.size = value; void load() }
async function continuePayment(item: InvoiceApplication) { checkoutId.value = item.id; try { checkout.value = (await invoiceAPI.getTaxPayments(item.id)).data } catch (actionError) { appStore.showError(extractI18nErrorMessage(actionError, t, 'invoices', t('invoices.loadFailed'))) } finally { checkoutId.value = null } }
async function cancel(item: InvoiceApplication) { if (!window.confirm(t('invoices.cancelConfirm'))) return; cancelId.value = item.id; try { await invoiceAPI.cancelInvoice(item.id); await load() } catch (actionError) { appStore.showError(extractI18nErrorMessage(actionError, t, 'invoices', t('invoices.loadFailed'))); await load() } finally { cancelId.value = null } }
function pay(channel: TaxPayment) { window.open(channel.payUrl, '_blank', 'noopener,noreferrer'); if (checkout.value) void pollPayment(checkout.value.application.id, channel.taxOrderNo) }
async function pollPayment(applicationId: number, taxOrderNo: string) { if (pollTimer) window.clearTimeout(pollTimer); for (let attempt = 0; attempt < 8; attempt += 1) { await new Promise<void>(resolve => { pollTimer = window.setTimeout(resolve, Math.min(15000, 2000 + attempt * 1500)) }); try { if ((await invoiceAPI.checkTaxPayment({ applicationId, taxOrderNo })).data.paid) { appStore.showSuccess(t('invoices.paymentComplete')); checkout.value = null; await load(); return } } catch { return } } }
function closeCheckout() { checkout.value = null; if (pollTimer) window.clearTimeout(pollTimer) }
function closePreview() { if (previewUrl.value) URL.revokeObjectURL(previewUrl.value); previewUrl.value = null }
async function preview(item: InvoiceApplication) { previewId.value = item.id; try { closePreview(); previewUrl.value = URL.createObjectURL((await invoiceAPI.downloadInvoicePDF(item.id)).data) } catch (previewError) { appStore.showError(extractI18nErrorMessage(previewError, t, 'invoices', t('invoices.loadFailed'))) } finally { previewId.value = null } }
async function download(item: InvoiceApplication) { downloadId.value = item.id; let url: string | null = null; try { url = URL.createObjectURL((await invoiceAPI.downloadInvoicePDF(item.id)).data); const anchor = document.createElement('a'); anchor.href = url; anchor.download = `invoice-${item.id}.pdf`; anchor.click() } catch (downloadError) { appStore.showError(extractI18nErrorMessage(downloadError, t, 'invoices', t('invoices.loadFailed'))) } finally { if (url) window.setTimeout(() => URL.revokeObjectURL(url as string), 1000); downloadId.value = null } }
function formatDate(value?: string) { return value ? new Date(value).toLocaleString() : '-' }
watch(checkout, value => { if (value) void load() })
onMounted(async () => {
  await load()
  const resumeID = Number(route.query.resume)
  if (resumeID > 0) {
    const item = items.value.find(candidate => candidate.id === resumeID)
    if (item?.status === 'awaiting_tax_payment') {
      await continuePayment(item)
    }
    await router.replace({ path: '/invoices', query: {} })
  }
})
onBeforeUnmount(() => { closeCheckout(); closePreview() })
</script>
