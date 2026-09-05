<template>
  <div>
    <label v-if="showLabel" class="mb-3 block text-sm font-medium text-gray-700 dark:text-gray-300">
      {{ t('payment.paymentMethod') }}
    </label>
    <div
      data-testid="payment-method-grid"
      class="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3"
    >
      <button
        v-for="method in sortedMethods"
        :key="method.type"
        type="button"
        :title="methodLabel(method)"
        :disabled="!method.available"
        :class="[
          'relative flex min-h-20 min-w-0 items-start justify-start rounded-md border px-3.5 py-2.5 text-left transition-colors',
          !method.available
            ? 'cursor-not-allowed border-gray-200 bg-gray-50 opacity-50 dark:border-dark-700 dark:bg-dark-800/50'
            : selected === method.type
              ? methodSelectedClass(method.type)
              : 'border-gray-300 bg-white text-gray-700 hover:border-gray-400 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-200 dark:hover:border-dark-500',
        ]"
        @click="method.available && emit('select', method.type)"
      >
        <span class="flex w-full min-w-0 items-start gap-2.5 pr-4">
          <img :src="methodIcon(method.type)" :alt="methodLabel(method)" class="h-7 w-7 shrink-0 object-contain" />
          <span class="flex min-w-0 flex-col items-start leading-none">
            <span data-testid="payment-method-label" class="block w-full whitespace-normal break-words text-base font-semibold leading-tight">
              {{ methodLabel(method) }}
            </span>
            <span
              v-if="method.fee_rate > 0"
              class="text-[10px] tracking-wide text-gray-500 dark:text-dark-400"
            >
              {{ t('payment.fee') }} {{ method.fee_rate }}%
            </span>
            <span
              v-if="!method.available && method.unavailable_reason === 'minimum' && method.minimum_amount != null"
              data-testid="payment-method-limit-hint"
              class="mt-1 block w-full text-xs leading-4 text-amber-700 dark:text-amber-300"
            >
              {{ t('payment.minimumToUse', { amount: formatPaymentAmount(method.minimum_amount, method.currency) }) }}
            </span>
            <span
              v-else-if="!method.available && method.unavailable_reason === 'maximum' && method.maximum_amount != null"
              data-testid="payment-method-limit-hint"
              class="mt-1 block w-full text-xs leading-4 text-amber-700 dark:text-amber-300"
            >
              {{ t('payment.maximumToUse', { amount: formatPaymentAmount(method.maximum_amount, method.currency) }) }}
            </span>
          </span>
        </span>
        <span
          v-if="selected === method.type && method.available"
          class="absolute right-2 top-2 flex h-4 w-4 items-center justify-center rounded-full bg-primary-500 text-white"
          aria-hidden="true"
        >
          <Icon name="check" size="xs" />
        </span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { paymentMethodOrder, isBuiltInAlipayMethod, isBuiltInWxpayMethod } from './providerConfig'
import { customPaymentTypeDefinition } from '@/custom/payments'
import alipayIcon from '@/assets/icons/alipay.svg'
import wxpayIcon from '@/assets/icons/wxpay.svg'
import stripeIcon from '@/assets/icons/stripe.svg'
import airwallexIcon from '@/assets/icons/airwallex.svg'
import paymentIcon from '@/assets/icons/payment.svg'
import { formatPaymentAmount as formatCurrencyAmount, normalizePaymentCurrency } from './currency'

export interface PaymentMethodOption {
  type: string
  display_name?: string
  fee_rate: number
  available: boolean
  unavailable_reason?: 'minimum' | 'maximum' | 'provider'
  minimum_amount?: number
  maximum_amount?: number
  currency?: string
}

const props = withDefaults(defineProps<{
  methods: PaymentMethodOption[]
  selected: string
  showLabel?: boolean
}>(), {
  showLabel: true,
})

const emit = defineEmits<{
  select: [type: string]
}>()

const { t } = useI18n()

const METHOD_ICONS: Record<string, string> = {
  alipay: alipayIcon,
  wxpay: wxpayIcon,
  stripe: stripeIcon,
  airwallex: airwallexIcon,
  credit_card: paymentIcon,
}

const sortedMethods = computed(() => {
  return [...props.methods].sort((a, b) => paymentMethodOrder(a.type) - paymentMethodOrder(b.type))
})

function methodIcon(type: string): string {
  if (isBuiltInAlipayMethod(type)) return METHOD_ICONS.alipay
  if (isBuiltInWxpayMethod(type)) return METHOD_ICONS.wxpay
  if (type === 'airwallex') return METHOD_ICONS.airwallex
  const customIcon = customPaymentTypeDefinition(type)?.methodIcon
  if (customIcon) return customIcon
  return METHOD_ICONS[type] || paymentIcon
}

function methodLabel(method: PaymentMethodOption): string {
  return method.display_name || t(`payment.methods.${method.type}`, method.type)
}

function formatPaymentAmount(value: number, currency?: string): string {
  return formatCurrencyAmount(value, normalizePaymentCurrency(currency))
}

function methodSelectedClass(type: string): string {
  if (isBuiltInAlipayMethod(type)) return 'border-[#02A9F1] bg-blue-50 text-gray-900 shadow-sm dark:bg-blue-950 dark:text-gray-100'
  if (isBuiltInWxpayMethod(type)) return 'border-[#09BB07] bg-green-50 text-gray-900 shadow-sm dark:bg-green-950 dark:text-gray-100'
  if (type === 'stripe') return 'border-[#676BE5] bg-indigo-50 text-gray-900 shadow-sm dark:bg-indigo-950 dark:text-gray-100'
  if (type === 'airwallex') return 'border-[#FF6B3D] bg-orange-50 text-gray-900 shadow-sm dark:border-[#FF8E3C] dark:bg-orange-950 dark:text-gray-100'
  const customClass = customPaymentTypeDefinition(type)?.selectedClass
  if (customClass) return customClass
  return 'border-primary-500 bg-primary-50 text-gray-900 shadow-sm dark:bg-primary-950 dark:text-gray-100'
}
</script>
