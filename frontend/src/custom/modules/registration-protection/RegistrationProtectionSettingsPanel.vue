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
