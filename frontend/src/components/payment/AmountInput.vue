<template>
  <div class="space-y-5">
    <!-- Custom Amount Input -->
    <div>
      <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
        {{ t('payment.amountLabel') }}
      </label>
      <div class="relative">
        <span class="pointer-events-none absolute left-4 top-1/2 -translate-y-1/2 text-lg font-medium text-gray-500 dark:text-gray-400">
          {{ currencyMark }}
        </span>
        <input
          data-testid="amount-input"
          type="text"
          inputmode="decimal"
          :value="customText"
          :placeholder="placeholderText"
          class="input h-14 w-full rounded-md border-gray-300 pl-10 pr-16 text-lg tabular-nums dark:border-dark-600"
          :aria-label="t('payment.amountLabel')"
          :aria-invalid="!!error"
          @input="handleInput"
        />
        <span class="pointer-events-none absolute right-4 top-1/2 -translate-y-1/2 text-xs font-medium text-gray-500 dark:text-gray-400">
          {{ normalizedCurrency }}
        </span>
      </div>
      <p class="min-h-10 pt-2 text-xs leading-4 text-amber-700 dark:text-amber-300" role="status">{{ error }}</p>
    </div>

    <!-- Quick Amount Buttons -->
    <div>
      <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
        {{ t('payment.quickAmounts') }}
      </label>
      <div class="grid grid-cols-3 gap-2 sm:grid-cols-6">
        <button
          v-for="amt in filteredAmounts"
          :key="amt"
          type="button"
          :data-testid="`quick-amount-${amt}`"
          :aria-pressed="modelValue === amt"
          :class="[
            'amount-preset h-10 rounded-md border px-2 text-center text-sm font-semibold tabular-nums transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary-500',
            modelValue === amt
              ? 'border-primary-500 bg-primary-50 text-primary-700 shadow-sm dark:border-primary-400 dark:bg-primary-900/30 dark:text-primary-300'
              : 'border-gray-300 bg-white text-gray-700 hover:border-primary-300 hover:text-primary-700 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-200 dark:hover:border-primary-700 dark:hover:text-primary-300',
          ]"
          @click="selectAmount(amt)"
        >
          {{ currencyMark }}{{ amt }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { currencySymbol, normalizePaymentCurrency } from './currency'

const props = withDefaults(defineProps<{
  amounts?: number[]
  modelValue: number | null
  min?: number
  max?: number
  currency?: string
  error?: string
}>(), {
  amounts: () => [10, 20, 50, 100, 200, 500, 1000, 2000, 5000],
  min: 0,
  max: 0,
  currency: 'USD',
  error: '',
})

const emit = defineEmits<{
  'update:modelValue': [value: number | null]
}>()

const { t } = useI18n()

const customText = ref('')
const normalizedCurrency = computed(() => normalizePaymentCurrency(props.currency))
const currencyMark = computed(() => currencySymbol(normalizedCurrency.value))

// 0 = no limit
const filteredAmounts = computed(() =>
  props.amounts.filter((a) => (props.min <= 0 || a >= props.min) && (props.max <= 0 || a <= props.max))
)

const placeholderText = computed(() => {
  if (props.min > 0 && props.max > 0) return `${props.min} - ${props.max}`
  if (props.min > 0) return `≥ ${props.min}`
  if (props.max > 0) return `≤ ${props.max}`
  return t('payment.enterAmount')
})

const AMOUNT_PATTERN = /^\d*(\.\d{0,2})?$/

function selectAmount(amt: number) {
  customText.value = String(amt)
  emit('update:modelValue', amt)
}

function handleInput(e: Event) {
  const input = e.target as HTMLInputElement
  const val = input.value
  if (!AMOUNT_PATTERN.test(val) || (val !== '' && val !== '.' && !Number.isFinite(Number(val)))) {
    input.value = customText.value
    return
  }
  customText.value = val
  if (val === '') {
    emit('update:modelValue', null)
    return
  }
  const num = parseFloat(val)
  if (!isNaN(num) && num > 0) {
    emit('update:modelValue', num)
  } else {
    emit('update:modelValue', null)
  }
}

watch(() => props.modelValue, (v) => {
  const nextText = v === null ? '' : String(v)
  if (nextText !== customText.value) {
    customText.value = nextText
  }
}, { immediate: true })
</script>
