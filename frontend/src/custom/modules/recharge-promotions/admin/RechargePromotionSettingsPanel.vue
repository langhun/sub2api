<template>
  <div class="card">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.rechargePromotion.title') }}</h2>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.rechargePromotion.description') }}</p>
    </div>
    <div class="space-y-5 p-6">
      <div class="flex items-center justify-between gap-4">
        <span class="font-medium text-gray-900 dark:text-white">{{ t('admin.rechargePromotion.enabled') }}</span>
        <Toggle v-model="form.recharge_promotion_enabled" />
      </div>
      <div class="grid gap-4 border-t border-gray-100 pt-4 dark:border-dark-700 md:grid-cols-2">
        <label class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.rechargePromotion.mode') }}
          <select v-model="form.recharge_promotion_mode" class="input mt-1">
            <option value="percentage">{{ t('admin.rechargePromotion.percentage') }}</option>
            <option value="tiered">{{ t('admin.rechargePromotion.tiered') }}</option>
          </select>
        </label>
      </div>

      <div v-if="form.recharge_promotion_mode === 'percentage'" class="grid gap-4 md:grid-cols-3">
        <label class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.rechargePromotion.percentageRate') }}<input v-model.number="form.recharge_promotion_percentage_rate" type="number" min="0" max="100" step="0.01" class="input mt-1" /></label>
        <label class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.rechargePromotion.minimumAmount') }}<input v-model.number="form.recharge_promotion_minimum_amount" type="number" min="0" step="0.01" class="input mt-1" /></label>
        <label class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.rechargePromotion.maximumBonus') }}<input v-model.number="form.recharge_promotion_maximum_bonus" type="number" min="0" step="0.01" class="input mt-1" /><span class="mt-1 block text-xs text-gray-400">{{ t('admin.rechargePromotion.noCap') }}</span></label>
      </div>

      <div v-else class="space-y-3 border-t border-gray-100 pt-4 dark:border-dark-700">
        <div class="flex items-center justify-between gap-4"><span class="font-medium text-gray-900 dark:text-white">{{ t('admin.rechargePromotion.tiers') }}</span><button type="button" class="btn btn-secondary btn-sm" @click="addTier">{{ t('admin.rechargePromotion.addTier') }}</button></div>
        <div v-for="(tier, index) in form.recharge_promotion_tier_rules" :key="index" class="grid gap-3 sm:grid-cols-[1fr_1fr_auto]">
          <label class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.rechargePromotion.tierMinimum') }}<input v-model.number="tier.minimum_amount" type="number" min="0.01" step="0.01" class="input mt-1" /></label>
          <label class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.rechargePromotion.tierBonus') }}<input v-model.number="tier.bonus_credit" type="number" min="0.01" step="0.01" class="input mt-1" /></label>
          <button type="button" class="btn btn-secondary self-end text-red-600 dark:text-red-300" @click="removeTier(index)">{{ t('admin.rechargePromotion.removeTier') }}</button>
        </div>
      </div>

      <div class="grid gap-4 border-t border-gray-100 pt-4 dark:border-dark-700 md:grid-cols-2">
        <label class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.rechargePromotion.startAt') }}<input v-model="form.recharge_promotion_start_at" type="datetime-local" class="input mt-1" /></label>
        <label class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.rechargePromotion.endAt') }}<input v-model="form.recharge_promotion_end_at" type="datetime-local" class="input mt-1" /></label>
      </div>
      <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.rechargePromotion.optionalTime') }}</p>
    </div>

    <RechargePromotionAdminPanel />
  </div>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Toggle from '@/components/common/Toggle.vue'
import RechargePromotionAdminPanel from './RechargePromotionAdminPanel.vue'
import type { RechargePromotionMode, RechargePromotionTierRule } from '../api'

interface Settings {
  recharge_promotion_enabled: boolean
  recharge_promotion_mode: RechargePromotionMode
  recharge_promotion_percentage_rate: number
  recharge_promotion_minimum_amount: number
  recharge_promotion_maximum_bonus: number
  recharge_promotion_tier_rules: RechargePromotionTierRule[]
  recharge_promotion_start_at: string
  recharge_promotion_end_at: string
}

const props = defineProps<{ modelValue: Settings }>()
const emit = defineEmits<{ 'update:modelValue': [value: Settings] }>()
const { t } = useI18n()

function normalizeDate(value: string): string {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hours = String(date.getHours()).padStart(2, '0')
  const minutes = String(date.getMinutes()).padStart(2, '0')
  return `${year}-${month}-${day}T${hours}:${minutes}`
}

function clone(value: Settings): Settings {
  return {
    ...value,
    recharge_promotion_start_at: normalizeDate(value.recharge_promotion_start_at),
    recharge_promotion_end_at: normalizeDate(value.recharge_promotion_end_at),
    recharge_promotion_tier_rules: value.recharge_promotion_tier_rules.map((tier) => ({ ...tier })),
  }
}

function payload(value: Settings): Settings {
  const toIso = (text: string) => text ? new Date(text).toISOString().replace('.000Z', 'Z') : ''
  return { ...value, recharge_promotion_start_at: toIso(value.recharge_promotion_start_at), recharge_promotion_end_at: toIso(value.recharge_promotion_end_at), recharge_promotion_tier_rules: value.recharge_promotion_tier_rules.map((tier) => ({ ...tier })) }
}

function sameSettings(left: Settings, right: Settings): boolean {
  return left.recharge_promotion_enabled === right.recharge_promotion_enabled &&
    left.recharge_promotion_mode === right.recharge_promotion_mode &&
    left.recharge_promotion_percentage_rate === right.recharge_promotion_percentage_rate &&
    left.recharge_promotion_minimum_amount === right.recharge_promotion_minimum_amount &&
    left.recharge_promotion_maximum_bonus === right.recharge_promotion_maximum_bonus &&
    left.recharge_promotion_start_at === right.recharge_promotion_start_at &&
    left.recharge_promotion_end_at === right.recharge_promotion_end_at &&
    left.recharge_promotion_tier_rules.length === right.recharge_promotion_tier_rules.length &&
    left.recharge_promotion_tier_rules.every((tier, index) =>
      tier.minimum_amount === right.recharge_promotion_tier_rules[index]?.minimum_amount &&
      tier.bonus_credit === right.recharge_promotion_tier_rules[index]?.bonus_credit,
    )
}

const form = reactive<Settings>(clone(props.modelValue))
watch(() => props.modelValue, (value) => {
  const next = clone(value)
  if (!sameSettings(form, next)) Object.assign(form, next)
}, { deep: true })
watch(form, (value) => {
  const next = payload(clone(value))
  const current = payload(clone(props.modelValue))
  if (!sameSettings(next, current)) emit('update:modelValue', next)
}, { deep: true })
function addTier() { form.recharge_promotion_tier_rules.push({ minimum_amount: 10, bonus_credit: 1 }) }
function removeTier(index: number) { form.recharge_promotion_tier_rules.splice(index, 1) }
</script>
