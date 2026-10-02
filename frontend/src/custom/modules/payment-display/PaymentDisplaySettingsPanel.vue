<template>
  <div class="space-y-4">
    <div>
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white">
        {{ localText('充值页面', 'Payment page') }}
      </h3>
      <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
        {{ localText('控制充值页面快捷金额按钮显示的最高金额。自定义输入和支付渠道限额不受影响。', 'Control the highest preset amount shown on the payment page. Custom input and provider limits are unchanged.') }}
      </p>
    </div>

    <label class="block max-w-sm text-sm font-medium text-gray-700 dark:text-gray-300">
      {{ localText('快捷金额最高显示', 'Maximum preset amount') }}
      <input
        v-model.number="form.payment_quick_amount_max"
        data-testid="payment-quick-amount-max"
        class="input mt-1"
        type="number"
        min="0"
        step="0.01"
        inputmode="decimal"
      />
      <span class="mt-1 block text-xs font-normal text-gray-500 dark:text-dark-400">
        {{ localText('填 0 表示不限制；例如填 500 时不显示 1000、2000、5000。', 'Use 0 for no limit. For example, 500 hides 1000, 2000, and 5000.') }}
      </span>
    </label>
  </div>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'

interface PaymentDisplaySettings {
  payment_quick_amount_max: number
}

const props = defineProps<{ modelValue: PaymentDisplaySettings }>()
const emit = defineEmits<{ 'update:modelValue': [value: PaymentDisplaySettings] }>()
const { locale } = useI18n()
const form = reactive<PaymentDisplaySettings>({ ...props.modelValue })

function localText(zh: string, en: string): string {
  return locale.value.startsWith('zh') ? zh : en
}

watch(() => props.modelValue, (value) => Object.assign(form, value), { deep: true })
watch(form, (value) => emit('update:modelValue', { ...value }), { deep: true })
</script>
