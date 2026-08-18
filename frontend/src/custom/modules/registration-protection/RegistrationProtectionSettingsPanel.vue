<template>
  <div class="card" data-testid="registration-protection-settings">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
        {{ localText('注册与奖励防护', 'Registration and reward protection') }}
      </h2>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
        {{ localText('减少邮箱别名批量注册，并避免重复注册特征重复领取注册赠送和签到奖励。', 'Reduce alias-based registrations and prevent repeated registration fingerprints from claiming signup and check-in rewards.') }}
      </p>
    </div>

    <div class="space-y-5 p-6">
      <div class="flex items-start justify-between gap-4">
        <div>
          <p class="font-medium text-gray-900 dark:text-white">
            {{ localText('拒绝邮箱别名注册', 'Reject email aliases during registration') }}
          </p>
          <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">
            {{ localText('拒绝 + 别名，以及 Gmail/Googlemail 本地部分点号变体；关闭后仍保留已有账号的别名查重。', 'Reject + aliases and Gmail/Googlemail dot variants. Existing-account alias deduplication remains when this is disabled.') }}
          </p>
        </div>
        <Toggle
          v-model="form.registration_alias_protection_enabled"
          data-testid="registration-alias-protection-toggle"
        />
      </div>

      <div class="flex items-start justify-between gap-4 border-t border-gray-100 pt-5 dark:border-dark-700">
        <div>
          <p class="font-medium text-gray-900 dark:text-white">
            {{ localText('重复注册特征奖励防护', 'Protect rewards for repeated registration fingerprints') }}
          </p>
          <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">
            {{ localText('相同公网 IP 与规范化浏览器 User-Agent 的后续注册不发放注册赠送，也不能签到；不会影响登录、充值或 API 使用。', 'Later registrations with the same public IP and normalized browser User-Agent receive no signup bonus and cannot check in. Login, payments, and API usage are unaffected.') }}
          </p>
        </div>
        <Toggle
          v-model="form.registration_reward_fingerprint_protection_enabled"
          data-testid="registration-reward-fingerprint-protection-toggle"
        />
      </div>

      <div class="border-t border-gray-100 pt-5 dark:border-dark-700">
        <div class="flex items-start justify-between gap-4">
          <div>
            <p class="font-medium text-gray-900 dark:text-white">{{ localText('网段聚集风控', 'Network cluster risk scoring') }}</p>
            <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ localText('同一 /24（IPv4）或 /64（IPv6）与规范化 UA 聚集时，停止奖励并暂时冻结新 Key。', 'Score registrations sharing an IPv4 /24 or IPv6 /64 with the same normalized UA.') }}</p>
          </div>
          <Toggle v-model="form.registration_cluster_protection_enabled" />
        </div>
        <div class="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-4">
          <label v-for="field in numericFields" :key="field.key" class="text-xs text-gray-500 dark:text-gray-400">
            {{ localText(field.zh, field.en) }}
            <input v-model.number="form[field.key]" type="number" min="1" class="mt-1 w-full rounded border border-gray-200 bg-white px-2 py-1 text-sm text-gray-900 dark:border-dark-600 dark:bg-dark-800 dark:text-white" />
          </label>
        </div>
      </div>

      <div class="grid gap-4 border-t border-gray-100 pt-5 sm:grid-cols-2 dark:border-dark-700">
        <div class="flex items-start justify-between gap-4">
          <div>
            <p class="font-medium text-gray-900 dark:text-white">{{ localText('高风险自动锁定', 'Automatic high-risk lock') }}</p>
            <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ localText('高置信度新账号临时拒绝 API 请求。', 'Temporarily deny API requests for high-confidence new accounts.') }}</p>
          </div>
          <Toggle v-model="form.registration_auto_high_risk_lock_enabled" />
        </div>
        <div class="flex items-start justify-between gap-4">
          <div>
            <p class="font-medium text-gray-900 dark:text-white">{{ localText('支付自动解锁', 'Payment auto-unlock') }}</p>
            <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ localText('累计完成支付达到阈值后恢复权限。', 'Restore access after the completed-payment threshold is reached.') }}</p>
          </div>
          <Toggle v-model="form.registration_payment_unlock_enabled" />
        </div>
      </div>

      <div class="border-t border-gray-100 pt-5 dark:border-dark-700">
        <div class="flex items-start justify-between gap-4">
          <div>
            <p class="font-medium text-gray-900 dark:text-white">{{ localText('新用户试用额度', 'New-user trial limits') }}</p>
            <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ localText('注册后默认 24 小时、2 美元、10 RPM、并发 2；累计完成 10 USDT 支付后自动解锁。', 'Default 24 hours, $2, 10 RPM, concurrency 2; unlock after $10 USDT completed payments.') }}</p>
          </div>
          <Toggle v-model="form.registration_trial_enabled" />
        </div>
        <div class="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-4">
          <label v-for="field in trialFields" :key="field.key" class="text-xs text-gray-500 dark:text-gray-400">
            {{ localText(field.zh, field.en) }}
            <input v-model.number="form[field.key]" type="number" min="0.1" step="0.1" class="mt-1 w-full rounded border border-gray-200 bg-white px-2 py-1 text-sm text-gray-900 dark:border-dark-600 dark:bg-dark-800 dark:text-white" />
          </label>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Toggle from '@/components/common/Toggle.vue'
import type { RegistrationProtectionSettings } from './settings'

const props = defineProps<{ modelValue: RegistrationProtectionSettings }>()
const emit = defineEmits<{ 'update:modelValue': [value: RegistrationProtectionSettings] }>()
const { locale } = useI18n()

const form = reactive<RegistrationProtectionSettings>({ ...props.modelValue })

const numericFields = [
  { key: 'registration_cluster_window_6h', zh: '6 小时账号数', en: '6h accounts' },
  { key: 'registration_cluster_window_24h', zh: '24 小时账号数', en: '24h accounts' },
  { key: 'registration_cluster_medium_threshold', zh: '中风险分数', en: 'Medium score' },
  { key: 'registration_cluster_high_threshold', zh: '高风险分数', en: 'High score' },
] as const
const trialFields = [
  { key: 'registration_trial_hours', zh: '试用小时', en: 'Trial hours' },
  { key: 'registration_trial_cost_limit', zh: '费用上限', en: 'Cost limit' },
  { key: 'registration_trial_rpm_limit', zh: 'RPM 上限', en: 'RPM limit' },
  { key: 'registration_trial_concurrency', zh: '并发上限', en: 'Concurrency' },
  { key: 'registration_payment_unlock_amount', zh: '解锁支付额', en: 'Unlock amount' },
] as const

function localText(zh: string, en: string): string {
  return locale.value.startsWith('zh') ? zh : en
}

watch(() => props.modelValue, (value) => {
  Object.assign(form, value)
}, { deep: true })

watch(form, (value) => {
  emit('update:modelValue', { ...value })
}, { deep: true })
</script>
