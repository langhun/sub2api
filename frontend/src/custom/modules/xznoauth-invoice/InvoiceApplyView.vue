<template>
  <AppLayout>
    <div class="mx-auto w-full max-w-5xl space-y-5" data-glass-content>
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">{{ t('invoices.applyTitle') }}</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('invoices.paymentHint') }}</p>
        </div>
        <button class="btn btn-secondary" type="button" @click="router.push('/invoices')">
          <Icon name="document" size="sm" /> {{ t('invoices.openRecords') }}
        </button>
      </div>

      <section class="card space-y-5 p-5">
        <div>
          <label class="input-label">{{ t('invoices.orderNumbers') }}</label>
          <textarea v-model="orderText" class="input mt-1 min-h-24 w-full" :placeholder="t('invoices.orderPlaceholder')" @input="invalidateValidation" />
        </div>
        <fieldset>
          <legend class="input-label">{{ t('invoices.taxMode') }}</legend>
          <div class="mt-2 grid gap-3 sm:grid-cols-2">
            <label v-for="option in taxOptions" :key="String(option.value)" class="flex cursor-pointer gap-3 rounded-lg border p-3 transition-colors" :class="needPayTax === option.value ? 'border-primary-500 bg-primary-50/60 dark:bg-primary-900/20' : 'border-gray-200 dark:border-dark-700'">
              <input v-model="needPayTax" class="mt-1" type="radio" :value="option.value" @change="invalidateValidation" />
              <span class="text-sm font-medium text-gray-800 dark:text-gray-100">{{ option.label }}</span>
            </label>
          </div>
        </fieldset>
        <div class="flex justify-end">
          <button class="btn btn-primary" type="button" :disabled="loading" @click="validate">
            <Icon v-if="loading" name="refresh" size="sm" class="animate-spin" /> {{ loading ? t('invoices.validating') : t('invoices.validate') }}
          </button>
        </div>
      </section>

      <section v-if="validation" class="card space-y-5 p-5">
        <div class="flex items-center justify-between">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('invoices.validatedOrders') }}</h2>
          <span class="text-sm text-gray-500">{{ validation.currency }}</span>
        </div>
        <div class="overflow-x-auto">
          <table class="min-w-full text-left text-sm">
            <thead class="border-b border-gray-200 text-xs uppercase text-gray-500 dark:border-dark-700">
              <tr><th class="px-3 py-2">{{ t('invoices.orderNumbers') }}</th><th class="px-3 py-2">{{ t('invoices.total') }}</th></tr>
            </thead>
            <tbody>
              <tr v-for="order in validation.orders" :key="order.orderNo" class="border-b border-gray-100 dark:border-dark-800">
                <td class="px-3 py-2 font-mono text-xs">{{ order.orderNo }}</td><td class="px-3 py-2">{{ order.amount }} {{ order.currency }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div class="grid gap-3 text-sm sm:grid-cols-4">
          <div><p class="text-gray-500">{{ t('invoices.total') }}</p><p class="font-semibold">{{ validation.totalAmount }} {{ validation.currency }}</p></div>
          <div><p class="text-gray-500">{{ t('invoices.tax') }}</p><p class="font-semibold">{{ validation.taxAmount || '0.00' }}</p></div>
          <div><p class="text-gray-500">{{ t('invoices.taxPaid') }}</p><p class="font-semibold">{{ validation.taxPaidAmount || '0.00' }}</p></div>
          <div><p class="text-gray-500">{{ t('invoices.taxDue') }}</p><p class="font-semibold">{{ validation.taxDueAmount || '0.00' }}</p></div>
        </div>
      </section>

      <section v-if="validation" class="card space-y-5 p-5">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('invoices.buyer') }}</h2>
        <div class="grid gap-4 sm:grid-cols-2">
          <div>
            <label class="input-label">{{ t('invoices.buyer') }}</label>
            <div class="mt-2 flex gap-4 text-sm"><label><input v-model="form.buyerType" type="radio" value="individual" /> {{ t('invoices.individual') }}</label><label><input v-model="form.buyerType" type="radio" value="company" /> {{ t('invoices.company') }}</label></div>
          </div>
          <label class="sm:col-span-2"><span class="input-label">{{ t('invoices.title') }}</span><input v-model="form.title" class="input mt-1 w-full" /></label>
          <label><span class="input-label">{{ t('invoices.taxpayerId') }}</span><input v-model="form.taxpayerId" class="input mt-1 w-full" /></label>
          <label><span class="input-label">{{ t('invoices.email') }}</span><input v-model="form.recipientEmail" class="input mt-1 w-full" type="email" /></label>
          <label><span class="input-label">{{ t('invoices.address') }}</span><input v-model="form.buyerAddress" class="input mt-1 w-full" /></label>
          <label><span class="input-label">{{ t('invoices.phone') }}</span><input v-model="form.buyerPhone" class="input mt-1 w-full" /></label>
          <label><span class="input-label">{{ t('invoices.bank') }}</span><input v-model="form.buyerBank" class="input mt-1 w-full" /></label>
          <label><span class="input-label">{{ t('invoices.bankAccount') }}</span><input v-model="form.buyerBankAccount" class="input mt-1 w-full" /></label>
        </div>
        <div class="flex justify-end">
          <button class="btn btn-primary" type="button" :disabled="submitting" @click="submit">
            <Icon v-if="submitting" name="refresh" size="sm" class="animate-spin" /> {{ submitting ? t('invoices.submitting') : t('invoices.submit') }}
          </button>
        </div>
      </section>
    </div>

    <BaseDialog :show="!!checkout" :title="t('invoices.paymentTitle')" width="normal" @close="checkout = null">
      <div v-if="checkout" class="space-y-4">
        <p class="text-sm text-gray-600 dark:text-gray-300">{{ t('invoices.paymentHint') }}</p>
        <div class="rounded-lg bg-gray-50 p-4 text-sm dark:bg-dark-800">
          <div class="flex justify-between"><span>{{ t('invoices.taxDue') }}</span><strong>{{ checkout.taxDueAmount }}</strong></div>
        </div>
        <div v-if="Object.keys(checkout.taxPayments || {}).length" class="grid gap-3">
          <button v-for="(channel, key) in checkout.taxPayments" :key="key" class="btn btn-primary justify-between" type="button" @click="openPayment(channel)">
            <span class="capitalize">{{ key }}</span><span>{{ t('invoices.pay') }}</span>
          </button>
        </div>
        <p v-else class="text-sm text-amber-600">{{ t('invoices.noChannels') }}</p>
      </div>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { reactive, ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores'
import { extractI18nErrorMessage } from '@/utils/apiError'
import { invoiceAPI, type InvoiceValidation, type TaxCheckout, type TaxPayment } from './api'

const { t } = useI18n()
const router = useRouter()
const appStore = useAppStore()
const orderText = ref('')
const needPayTax = ref(false)
const loading = ref(false)
const submitting = ref(false)
const validation = ref<InvoiceValidation | null>(null)
const validatedSnapshot = ref<{ orderNos: string[]; needPayTax: boolean } | null>(null)
const checkout = ref<TaxCheckout | null>(null)
const form = reactive({ buyerType: 'individual' as 'individual' | 'company', title: '', taxpayerId: '', buyerAddress: '', buyerPhone: '', buyerBank: '', buyerBankAccount: '', recipientEmail: '' })
const taxOptions = computed(() => [{ value: false, label: t('invoices.noTaxPayment') }, { value: true, label: t('invoices.taxPaymentRequired') }])

function parseOrders() { return orderText.value.split(/[\s,;]+/).map(value => value.trim()).filter(Boolean) }
function invalidateValidation() { validation.value = null; validatedSnapshot.value = null }
async function validate() {
  const orderNos = parseOrders()
  if (!orderNos.length) return appStore.showError(t('invoices.errors.invalid'))
  if (new Set(orderNos).size !== orderNos.length) return appStore.showError(t('invoices.errors.duplicateOrders'))
  loading.value = true
  try {
    validation.value = (await invoiceAPI.validateOrders({ orderNos, needPayTax: needPayTax.value })).data
    validatedSnapshot.value = { orderNos: [...orderNos], needPayTax: needPayTax.value }
  } catch (error) {
    validation.value = null
    validatedSnapshot.value = null
    appStore.showError(resolveInvoiceError(error))
  } finally { loading.value = false }
}
async function submit() {
  if (!validation.value || !validatedSnapshot.value) return
  submitting.value = true
  try {
    const saved = (await invoiceAPI.createInvoice({ ...form, orderNos: [...validatedSnapshot.value.orderNos], needPayTax: validatedSnapshot.value.needPayTax })).data
    appStore.showSuccess(t('invoices.saved'))
    router.push(saved.status === 'awaiting_tax_payment' ? `/invoices?resume=${saved.id}` : '/invoices')
  } catch (error) { appStore.showError(resolveInvoiceError(error)) } finally { submitting.value = false }
}
function resolveInvoiceError(error: unknown) {
  const apiError = error as { code?: string; reason?: string }
  if (apiError.code === 'IDENTITY_VERIFICATION_REQUIRED' || apiError.reason === 'IDENTITY_VERIFICATION_REQUIRED') {
    return t('invoices.errors.ownerVerification')
  }
  return extractI18nErrorMessage(error, t, 'invoices.errors', t('invoices.errors.invalid'))
}
function openPayment(channel: TaxPayment) {
  window.open(channel.payUrl, '_blank', 'noopener,noreferrer')
  if (checkout.value) void pollPayment(checkout.value.application.id, channel.taxOrderNo)
}
async function pollPayment(applicationId: number, taxOrderNo: string) {
  for (let attempt = 0; attempt < 8; attempt += 1) {
    await new Promise(resolve => window.setTimeout(resolve, Math.min(15000, 2000 + attempt * 1500)))
    try {
      if ((await invoiceAPI.checkTaxPayment({ applicationId, taxOrderNo })).data.paid) {
        appStore.showSuccess(t('invoices.paymentComplete'))
        router.push('/invoices')
        return
      }
    } catch { return }
  }
}
</script>
