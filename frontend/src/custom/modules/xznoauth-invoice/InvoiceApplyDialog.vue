<template>
  <BaseDialog :show="show" :title="t('invoices.applyTitle')" width="wide" @close="close">
    <div class="space-y-5">
      <div class="flex items-center gap-2 border-b border-gray-200 pb-4 text-xs dark:border-dark-700">
        <span class="flex items-center gap-1.5 font-medium text-primary-600 dark:text-primary-400"><span class="flex h-5 w-5 items-center justify-center rounded-full bg-primary-100 text-[11px] dark:bg-primary-900/40">✓</span>{{ t('invoices.selectOrders') }}</span>
        <span class="h-px flex-1 bg-gray-200 dark:bg-dark-700" />
        <span :class="step >= 2 ? 'font-medium text-primary-600 dark:text-primary-400' : 'text-gray-400'">{{ t('invoices.taxReview') }}</span>
        <span class="h-px flex-1 bg-gray-200 dark:bg-dark-700" />
        <span :class="step >= 3 ? 'font-medium text-primary-600 dark:text-primary-400' : 'text-gray-400'">{{ t('invoices.buyerDetails') }}</span>
      </div>

      <div v-if="step === 2" class="space-y-5">
        <div class="grid grid-cols-2 gap-3 border-y border-gray-200 py-4 text-sm dark:border-dark-700 sm:grid-cols-4">
          <div><p class="text-xs text-gray-500 dark:text-gray-400">{{ t('invoices.orderCount') }}</p><p class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">{{ orders.length }}</p></div>
          <div><p class="text-xs text-gray-500 dark:text-gray-400">{{ t('invoices.orderTotal') }}</p><p class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">{{ money(validation?.totalAmount, validation?.currency) }}</p></div>
          <div><p class="text-xs text-gray-500 dark:text-gray-400">{{ t('invoices.tax') }}</p><p class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">{{ money(validation?.taxAmount, validation?.currency) }}</p></div>
          <div><p class="text-xs text-gray-500 dark:text-gray-400">{{ t('invoices.taxDue') }}</p><p class="mt-1 text-lg font-semibold text-primary-700 dark:text-primary-300">{{ money(validation?.taxDueAmount, validation?.currency) }}</p></div>
        </div>

        <fieldset>
          <legend class="input-label">{{ t('invoices.taxMode') }}</legend>
          <div class="mt-2 grid gap-2 sm:grid-cols-2">
            <label v-for="option in taxOptions" :key="String(option.value)" class="flex cursor-pointer items-center gap-3 rounded-lg border px-3 py-2.5 text-sm transition-colors" :class="needPayTax === option.value ? 'border-primary-500 bg-primary-50/60 dark:bg-primary-900/20' : 'border-gray-200 dark:border-dark-700'">
              <input v-model="needPayTax" type="radio" :value="option.value" />
              <span>{{ option.label }}</span>
            </label>
          </div>
        </fieldset>

        <p v-if="loading" class="text-sm text-gray-500">{{ t('invoices.validating') }}</p>
        <p v-else-if="validationError" class="text-sm text-red-600 dark:text-red-400">{{ validationError }}</p>
        <div class="flex justify-end gap-2 border-t border-gray-200 pt-4 dark:border-dark-700">
          <button class="btn btn-secondary" type="button" @click="close">{{ t('common.cancel') }}</button>
        <button class="btn btn-primary" type="button" :disabled="loading || !validationReady" @click="step = 3">{{ t('common.next') }}</button>
        </div>
      </div>

      <div v-else-if="step === 3" class="space-y-5">
        <div class="rounded-lg bg-gray-50 px-4 py-3 text-sm dark:bg-dark-800">
          <div class="flex flex-wrap items-center justify-between gap-2"><span class="text-gray-500">{{ t('invoices.selectedOrders') }}</span><strong>{{ orders.length }} {{ t('invoices.ordersUnit') }}</strong></div>
          <div class="mt-1 flex flex-wrap items-center justify-between gap-2"><span class="text-gray-500">{{ t('invoices.taxDue') }}</span><strong>{{ money(validation?.taxDueAmount, validation?.currency) }}</strong></div>
        </div>

        <div class="grid gap-x-4 gap-y-4 sm:grid-cols-2">
          <div class="sm:col-span-2">
            <span class="input-label">{{ t('invoices.buyer') }}</span>
            <div class="mt-2 grid grid-cols-2 rounded-lg bg-gray-100 p-1 text-sm dark:bg-dark-800">
              <label class="cursor-pointer rounded-md px-3 py-2 text-center" :class="form.buyerType === 'company' ? 'bg-white shadow-sm dark:bg-dark-700' : 'text-gray-500'"><input v-model="form.buyerType" class="sr-only" type="radio" value="company" />{{ t('invoices.company') }}</label>
              <label class="cursor-pointer rounded-md px-3 py-2 text-center" :class="form.buyerType === 'individual' ? 'bg-white shadow-sm dark:bg-dark-700' : 'text-gray-500'"><input v-model="form.buyerType" class="sr-only" type="radio" value="individual" />{{ t('invoices.individual') }}</label>
            </div>
          </div>
          <label><span class="input-label">{{ t('invoices.title') }}</span><input v-model.trim="form.title" class="input mt-1 w-full" :placeholder="t('invoices.titlePlaceholder')" /></label>
          <label><span class="input-label">{{ t('invoices.taxpayerId') }}</span><input v-model.trim="form.taxpayerId" class="input mt-1 w-full" /></label>
          <label class="sm:col-span-2"><span class="input-label">{{ t('invoices.email') }}</span><input v-model.trim="form.recipientEmail" class="input mt-1 w-full" type="email" :placeholder="t('invoices.emailPlaceholder')" /><span class="mt-1 block text-xs text-gray-500">{{ t('invoices.emailHint') }}</span></label>
          <label><span class="input-label">{{ t('invoices.address') }} <em class="font-normal not-italic text-gray-400">{{ t('invoices.optional') }}</em></span><input v-model.trim="form.buyerAddress" class="input mt-1 w-full" /></label>
          <label><span class="input-label">{{ t('invoices.phone') }} <em class="font-normal not-italic text-gray-400">{{ t('invoices.optional') }}</em></span><input v-model.trim="form.buyerPhone" class="input mt-1 w-full" /></label>
          <label><span class="input-label">{{ t('invoices.bank') }} <em class="font-normal not-italic text-gray-400">{{ t('invoices.optional') }}</em></span><input v-model.trim="form.buyerBank" class="input mt-1 w-full" /></label>
          <label><span class="input-label">{{ t('invoices.bankAccount') }} <em class="font-normal not-italic text-gray-400">{{ t('invoices.optional') }}</em></span><input v-model.trim="form.buyerBankAccount" class="input mt-1 w-full" /></label>
        </div>

        <p v-if="formError" class="text-sm text-red-600 dark:text-red-400">{{ formError }}</p>
        <div class="flex justify-end gap-2 border-t border-gray-200 pt-4 dark:border-dark-700">
          <button class="btn btn-secondary" type="button" @click="step = 2">{{ t('common.back') }}</button>
          <button v-if="submissionUncertain" class="btn btn-secondary" type="button" @click="emit('saved')">{{ t('invoices.openRecords') }}</button>
          <button class="btn btn-primary" type="button" :disabled="submitting || submissionUncertain" @click="submit">{{ submitting ? t('invoices.submitting') : t('invoices.submit') }}</button>
        </div>
      </div>

      <div v-else class="space-y-4">
        <div class="rounded-lg bg-primary-50 px-4 py-3 text-sm text-primary-800 dark:bg-primary-900/20 dark:text-primary-200">{{ paymentPending ? t('invoices.paymentPending') : t('invoices.paymentHint') }}</div>
        <div class="rounded-lg border border-gray-200 p-4 text-sm dark:border-dark-700"><div class="flex justify-between"><span>{{ t('invoices.taxPaid') }}</span><strong>{{ money(checkout?.taxPaidAmount, checkout?.application.currency) }}</strong></div><div class="mt-2 flex justify-between"><span>{{ t('invoices.taxDue') }}</span><strong>{{ money(checkout?.taxDueAmount, checkout?.application.currency) }}</strong></div></div>
        <div v-if="!paymentPending && checkout && Object.keys(checkout.taxPayments || {}).length" class="grid gap-2 sm:grid-cols-2">
          <button v-for="(channel, key) in checkout.taxPayments" :key="key" class="btn btn-primary justify-between" type="button" @click="openPayment(channel)"><span class="capitalize">{{ key }}</span><span>{{ t('invoices.pay') }}</span></button>
        </div>
        <p v-if="!paymentPending && (!checkout || !Object.keys(checkout.taxPayments || {}).length)" class="text-sm text-primary-700 dark:text-primary-300">{{ t('invoices.noChannels') }}</p>
        <div class="flex justify-end border-t border-gray-200 pt-4 dark:border-dark-700"><button class="btn btn-secondary" type="button" @click="close">{{ t('common.close') }}</button></div>
      </div>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { PaymentOrder } from '@/types/payment'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { useAppStore } from '@/stores'
import { extractI18nErrorMessage } from '@/utils/apiError'
import { invoiceAPI, type InvoiceValidation, type TaxCheckout, type TaxPayment } from './api'

const props = defineProps<{ show: boolean; orders: PaymentOrder[] }>()
const emit = defineEmits<{ close: []; saved: [] }>()
const { t } = useI18n()
const appStore = useAppStore()
const step = ref(2)
const needPayTax = ref(false)
const loading = ref(false)
const submitting = ref(false)
const validation = ref<InvoiceValidation | null>(null)
const validationError = ref('')
const formError = ref('')
const checkout = ref<TaxCheckout | null>(null)
const paymentPending = ref(false)
const submissionUncertain = ref(false)
const savedApplicationId = ref<number | null>(null)
const validatedOrderNos = ref<string[]>([])
const validatedNeedPayTax = ref<boolean | null>(null)
const validationRun = ref(0)
const pollRun = ref(0)
let pollTimer: number | undefined
let pollResume: (() => void) | undefined
const form = reactive({ buyerType: 'company' as 'individual' | 'company', title: '', taxpayerId: '', buyerAddress: '', buyerPhone: '', buyerBank: '', buyerBankAccount: '', recipientEmail: '' })
const taxOptions = computed(() => [{ value: false, label: t('invoices.noTaxPayment') }, { value: true, label: t('invoices.taxPaymentRequired') }])
const currentOrderNos = computed(() => props.orders.map(order => order.out_trade_no.trim()).filter(Boolean))
const validationReady = computed(() => {
  return Boolean(validation.value) && validatedNeedPayTax.value === needPayTax.value &&
    validatedOrderNos.value.length === currentOrderNos.value.length &&
    validatedOrderNos.value.every((orderNo, index) => orderNo === currentOrderNos.value[index])
})
function money(value?: string | number, currency?: string) {
  if (value === undefined || value === null || value === '') return '-'
  return `${currency || 'CNY'} ${Number(value).toFixed(2)}`
}

function close() {
  stopPolling()
  emit('close')
}

function stopPolling() {
  pollRun.value += 1
  if (pollTimer) {
    window.clearTimeout(pollTimer)
    pollTimer = undefined
  }
  pollResume?.()
  pollResume = undefined
}

function resolveError(error: unknown) {
  return extractI18nErrorMessage(error, t, 'invoices.errors', t('invoices.errors.invalid'))
}

async function validate() {
  const orderNos = currentOrderNos.value
  const run = ++validationRun.value
  validation.value = null
  validatedOrderNos.value = []
  validatedNeedPayTax.value = null
  if (!orderNos.length) {
    loading.value = false
    return
  }
  loading.value = true
  validationError.value = ''
  try {
    const requestedNeedPayTax = needPayTax.value
    const result = (await invoiceAPI.validateOrders({ orderNos, needPayTax: requestedNeedPayTax })).data
    if (run !== validationRun.value) return
    const verifiedOrders = result.orders || []
    const identifiers = new Set(verifiedOrders.flatMap(order => [order.orderNo, order.externalNo, order.transactionId].filter(Boolean) as string[]))
    if (verifiedOrders.length !== orderNos.length || orderNos.some(orderNo => !identifiers.has(orderNo))) {
      validationError.value = t('invoices.errors.incompleteValidation')
      return
    }
    validation.value = result
    validatedOrderNos.value = [...orderNos]
    validatedNeedPayTax.value = requestedNeedPayTax
  } catch (error) {
    if (run !== validationRun.value) return
    validation.value = null
    validationError.value = resolveError(error)
  } finally {
    if (run === validationRun.value) loading.value = false
  }
}

async function submit() {
  formError.value = ''
  if (!validationReady.value) {
    formError.value = t('invoices.errors.revalidate')
    return
  }
  const email = form.recipientEmail.trim()
  const phone = form.buyerPhone.trim()
  const bankAccount = form.buyerBankAccount.replace(/[\s-]/g, '')
  const phoneDigits = phone.replace(/\D/g, '').length
  if (!form.title.trim() || !email || !form.taxpayerId.trim()) {
    formError.value = t('invoices.errors.requiredFields')
    return
  }
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
    formError.value = t('invoices.errors.invalidEmail')
    return
  }
  if (phone && (!/^[0-9+()\- ]{6,32}$/.test(phone) || phoneDigits < 6)) {
    formError.value = t('invoices.errors.invalidPhone')
    return
  }
  if (bankAccount && !/^\d{8,32}$/.test(bankAccount)) {
    formError.value = t('invoices.errors.invalidBankAccount')
    return
  }
  form.buyerBankAccount = bankAccount
  submitting.value = true
  try {
    const saved = (await invoiceAPI.createInvoice({ ...form, orderNos: validatedOrderNos.value, needPayTax: validatedNeedPayTax.value ?? false, recipientEmail: email, buyerPhone: phone })).data
    savedApplicationId.value = saved.id
    if (saved.status === 'awaiting_tax_payment') {
      try {
        checkout.value = (await invoiceAPI.getTaxPayments(saved.id)).data
        if (checkout.value.application.status !== 'awaiting_tax_payment') {
          appStore.showSuccess(t('invoices.saved'))
          emit('saved')
          return
        }
        paymentPending.value = Number(checkout.value.taxDueAmount || 0) <= 0 && checkout.value.application.status === 'awaiting_tax_payment'
        step.value = 4
      } catch (checkoutError) {
        appStore.showError(resolveError(checkoutError))
        emit('saved')
      }
    } else {
      appStore.showSuccess(t('invoices.saved'))
      emit('saved')
    }
  } catch (error) {
    const status = error && typeof error === 'object' && 'status' in error ? Number((error as { status?: unknown }).status) : 0
    if (status === 0 || status >= 500) {
      submissionUncertain.value = true
      formError.value = t('invoices.errors.submitUncertain')
    } else {
      formError.value = resolveError(error)
    }
  } finally { submitting.value = false }
}

function openPayment(channel: TaxPayment) {
  window.open(channel.payUrl, '_blank', 'noopener,noreferrer')
  if (savedApplicationId.value) void pollPayment(savedApplicationId.value, channel.taxOrderNo)
}

async function pollPayment(applicationId: number, taxOrderNo: string) {
  stopPolling()
  const run = ++pollRun.value
  for (let attempt = 0; attempt < 8; attempt += 1) {
    await new Promise<void>(resolve => {
      const resume = () => {
        pollTimer = undefined
        pollResume = undefined
        resolve()
      }
      pollResume = resume
      pollTimer = window.setTimeout(resume, Math.min(15000, 2000 + attempt * 1500))
    })
    if (run !== pollRun.value) return
    try {
      if ((await invoiceAPI.checkTaxPayment({ taxOrderNo })).data.paid) {
        if (run !== pollRun.value) return
        const refreshed = (await invoiceAPI.getTaxPayments(applicationId)).data
        if (run !== pollRun.value) return
        if (refreshed.application.status === 'pending' || refreshed.application.status === 'approved' || refreshed.application.status === 'completed') {
          appStore.showSuccess(t('invoices.paymentComplete'))
          emit('saved')
        } else if (refreshed.application.status !== 'awaiting_tax_payment') {
          emit('saved')
        } else {
          checkout.value = refreshed
          paymentPending.value = Number(refreshed.taxDueAmount || 0) <= 0
          appStore.showSuccess(paymentPending.value ? t('invoices.paymentPending') : t('invoices.paymentPartial'))
        }
        return
      }
    } catch (error) {
      appStore.showError(resolveError(error))
      return
    }
  }
}

watch(() => props.show, (show) => {
  if (!show) {
    stopPolling()
    return
  }
  step.value = 2
  checkout.value = null
  paymentPending.value = false
  submissionUncertain.value = false
  savedApplicationId.value = null
  void validate()
}, { immediate: true })
watch(needPayTax, () => { if (props.show) { step.value = 2; void validate() } })
watch(() => currentOrderNos.value.join('\u0000'), () => { if (props.show) { step.value = 2; void validate() } })
onBeforeUnmount(stopPolling)
</script>
