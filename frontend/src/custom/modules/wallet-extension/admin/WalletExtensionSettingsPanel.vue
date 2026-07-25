<template>
  <div class="card">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.settings.balanceFeatures.transferTitle') }}</h2>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.settings.balanceFeatures.transferDescription') }}</p>
    </div>
    <div class="space-y-5 p-6">
      <div class="flex items-center justify-between"><span class="font-medium text-gray-900 dark:text-white">{{ t('admin.settings.balanceFeatures.transferEnabled') }}</span><Toggle v-model="form.transfer_enabled" /></div>
      <div v-if="form.transfer_enabled" class="grid gap-4 md:grid-cols-3">
        <label class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.settings.balanceFeatures.feeRate') }}<input v-model.number="form.transfer_fee_rate" type="number" min="0" step="0.001" class="input mt-1" /></label>
        <label class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.settings.balanceFeatures.minAmount') }}<input v-model.number="form.transfer_min_amount" type="number" min="0" step="0.01" class="input mt-1" /></label>
        <label class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.settings.balanceFeatures.maxAmount') }}<input v-model.number="form.transfer_max_amount" type="number" min="0" step="0.01" class="input mt-1" /></label>
        <label class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.settings.balanceFeatures.dailyLimit') }}<input v-model.number="form.transfer_daily_limit" type="number" min="0" step="0.01" class="input mt-1" /></label>
        <label class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.settings.balanceFeatures.dailyCount') }}<input v-model.number="form.transfer_daily_count_limit" type="number" min="0" class="input mt-1" /></label>
        <label class="flex items-center gap-3 pt-6 text-sm text-gray-600 dark:text-gray-300"><Toggle v-model="form.transfer_vip_fee_exempt" />{{ t('admin.settings.balanceFeatures.vipExempt') }}</label>
      </div>
      <div class="flex items-center justify-between border-t border-gray-100 pt-4 dark:border-dark-700"><span class="font-medium text-gray-900 dark:text-white">{{ t('admin.settings.balanceFeatures.redpacketEnabled') }}</span><Toggle v-model="form.redpacket_enabled" /></div>
      <div v-if="form.redpacket_enabled" class="grid gap-4 md:grid-cols-2">
        <label class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.settings.balanceFeatures.redpacketMaxCount') }}<input v-model.number="form.redpacket_max_count" type="number" min="1" class="input mt-1" /></label>
        <label class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.settings.balanceFeatures.redpacketExpireHours') }}<input v-model.number="form.redpacket_expire_hours" type="number" min="1" class="input mt-1" /></label>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Toggle from '@/components/common/Toggle.vue'

interface WalletExtensionSettings {
  transfer_enabled: boolean
  transfer_fee_rate: number
  transfer_min_amount: number
  transfer_max_amount: number
  transfer_daily_limit: number
  transfer_daily_count_limit: number
  transfer_vip_fee_exempt: boolean
  redpacket_enabled: boolean
  redpacket_max_count: number
  redpacket_expire_hours: number
}

const props = defineProps<{ modelValue: WalletExtensionSettings }>()
const emit = defineEmits<{ 'update:modelValue': [value: WalletExtensionSettings] }>()
const { t } = useI18n()
const form = reactive<WalletExtensionSettings>({ ...props.modelValue })

watch(() => props.modelValue, (value) => Object.assign(form, value), { deep: true })
watch(form, (value) => emit('update:modelValue', { ...value }), { deep: true })
</script>
