import type { CustomSettingsPanel, CustomSettingsValues } from '../../registry'
import RechargePromotionSettingsPanel from './admin/RechargePromotionSettingsPanel.vue'
import type { RechargePromotionMode, RechargePromotionTieredPercentageRule, RechargePromotionTierRule } from './api'

interface RechargePromotionSettings {
  recharge_promotion_enabled: boolean
  recharge_promotion_mode: RechargePromotionMode
  recharge_promotion_percentage_rate: number
  recharge_promotion_minimum_amount: number
  recharge_promotion_maximum_bonus: number
  recharge_promotion_tier_rules: RechargePromotionTierRule[]
  recharge_promotion_tiered_percentage_rules: RechargePromotionTieredPercentageRule[]
  recharge_promotion_start_at: string
  recharge_promotion_end_at: string
}

const defaultRechargePromotionSettings = (): RechargePromotionSettings => ({
  recharge_promotion_enabled: false,
  recharge_promotion_mode: 'percentage',
  recharge_promotion_percentage_rate: 0,
  recharge_promotion_minimum_amount: 0,
  recharge_promotion_maximum_bonus: 0,
  recharge_promotion_tier_rules: [],
  recharge_promotion_tiered_percentage_rules: [],
  recharge_promotion_start_at: '',
  recharge_promotion_end_at: '',
})

function finiteMoney(value: unknown): number {
  const parsed = Number(value)
  return Number.isFinite(parsed) ? Math.round(parsed * 100) / 100 : 0
}

function tiers(value: unknown): RechargePromotionTierRule[] {
  if (!Array.isArray(value)) return []
  return value.map((item) => {
    const rule = item as Partial<RechargePromotionTierRule>
    return { minimum_amount: finiteMoney(rule.minimum_amount), bonus_credit: finiteMoney(rule.bonus_credit) }
  }).sort((left, right) => left.minimum_amount - right.minimum_amount)
}

function tieredPercentages(value: unknown): RechargePromotionTieredPercentageRule[] {
  if (!Array.isArray(value)) return []
  return value.map((item) => {
    const rule = item as Partial<RechargePromotionTieredPercentageRule>
    return {
      minimum_amount: finiteMoney(rule.minimum_amount),
      maximum_amount: finiteMoney(rule.maximum_amount),
      percentage_rate: finiteMoney(rule.percentage_rate),
    }
  }).sort((left, right) => left.minimum_amount - right.minimum_amount)
}

function mode(value: unknown): RechargePromotionMode {
  if (value === 'tiered' || value === 'tiered_percentage') return value
  return 'percentage'
}

function readSettings(value: CustomSettingsValues): RechargePromotionSettings {
  const values = value as Record<string, unknown>
  const defaults = defaultRechargePromotionSettings()
  return {
    recharge_promotion_enabled: values.recharge_promotion_enabled === true,
    recharge_promotion_mode: mode(values.recharge_promotion_mode),
    recharge_promotion_percentage_rate: finiteMoney(values.recharge_promotion_percentage_rate),
    recharge_promotion_minimum_amount: finiteMoney(values.recharge_promotion_minimum_amount),
    recharge_promotion_maximum_bonus: finiteMoney(values.recharge_promotion_maximum_bonus),
    recharge_promotion_tier_rules: tiers(values.recharge_promotion_tier_rules),
    recharge_promotion_tiered_percentage_rules: tieredPercentages(values.recharge_promotion_tiered_percentage_rules),
    recharge_promotion_start_at: typeof values.recharge_promotion_start_at === 'string' ? values.recharge_promotion_start_at : defaults.recharge_promotion_start_at,
    recharge_promotion_end_at: typeof values.recharge_promotion_end_at === 'string' ? values.recharge_promotion_end_at : defaults.recharge_promotion_end_at,
  }
}

function hasAtMostTwoDecimals(value: unknown): boolean {
  const number = Number(value)
  return Number.isFinite(number) && Math.abs(number * 100 - Math.round(number * 100)) < 1e-9
}

function validateTieredPercentagePrecision(value: CustomSettingsValues): string {
  const values = value as Record<string, unknown>
  if (values.recharge_promotion_mode !== 'tiered_percentage') return ''
  if (!Array.isArray(values.recharge_promotion_tiered_percentage_rules)) return ''
  for (const item of values.recharge_promotion_tiered_percentage_rules) {
    const rule = item as Partial<RechargePromotionTieredPercentageRule>
    if (!hasAtMostTwoDecimals(rule.minimum_amount) || !hasAtMostTwoDecimals(rule.maximum_amount) || !hasAtMostTwoDecimals(rule.percentage_rate)) {
      return '阶梯比例金额和赠送比例最多保留两位小数'
    }
  }
  return ''
}

function validate(settings: RechargePromotionSettings): string {
  const active = settings.recharge_promotion_enabled
  if (settings.recharge_promotion_percentage_rate < 0 || settings.recharge_promotion_percentage_rate > 100) return '充值赠送比例必须在 0 到 100 之间'
  if (settings.recharge_promotion_minimum_amount < 0 || settings.recharge_promotion_maximum_bonus < 0) return '充值赠送金额不能小于 0'
  if (active && settings.recharge_promotion_mode === 'percentage' && settings.recharge_promotion_percentage_rate <= 0) return '启用固定比例活动时，赠送比例必须大于 0'
  if (settings.recharge_promotion_mode === 'tiered') {
    if (active && settings.recharge_promotion_tier_rules.length === 0) return '启用阶梯活动时至少添加一个档位'
    let previous = 0
    for (const rule of settings.recharge_promotion_tier_rules) {
      if (rule.minimum_amount <= 0 || rule.bonus_credit <= 0) return '阶梯金额和赠送额度必须大于 0'
      if (previous > 0 && rule.minimum_amount <= previous) return '阶梯满额金额必须严格递增'
      previous = rule.minimum_amount
    }
  }
  if (settings.recharge_promotion_mode === 'tiered_percentage') {
    const rules = settings.recharge_promotion_tiered_percentage_rules
    if (active && rules.length === 0) return '启用阶梯比例活动时至少添加一个档位'
    for (let index = 0; index < rules.length; index++) {
      const rule = rules[index]
      if (rule.minimum_amount <= 0) return '阶梯比例最低金额必须大于 0'
      if (rule.maximum_amount < 0) return '阶梯比例最高金额不能小于 0'
      if (rule.percentage_rate <= 0 || rule.percentage_rate > 100) return '阶梯比例赠送比例必须大于 0 且不超过 100'
      const last = index === rules.length - 1
      if (last && rule.maximum_amount !== 0) return '阶梯比例最后一档最高金额必须为 0（无上限）'
      if (!last) {
        const next = rules[index + 1]
        if (rule.maximum_amount <= rule.minimum_amount) return '阶梯比例最高金额必须大于最低金额'
        if (rule.maximum_amount !== next.minimum_amount) return '阶梯比例区间必须连续且不能重叠'
        if (rule.percentage_rate > next.percentage_rate) return '阶梯比例赠送比例必须非递减'
      }
    }
  }
  const start = settings.recharge_promotion_start_at ? Date.parse(settings.recharge_promotion_start_at) : NaN
  const end = settings.recharge_promotion_end_at ? Date.parse(settings.recharge_promotion_end_at) : NaN
  if (Number.isFinite(start) && Number.isFinite(end) && start >= end) return '开始时间必须早于结束时间'
  return ''
}

export const rechargePromotionSettingsPanels: readonly CustomSettingsPanel[] = [
  {
    id: 'recharge-promotions',
    placement: 'features',
    order: 30,
    component: RechargePromotionSettingsPanel,
    settingKeys: Object.keys(defaultRechargePromotionSettings()),
    createForm: defaultRechargePromotionSettings,
    fromSettings: readSettings,
    toPayload: (form) => {
      const settings = readSettings(form)
      return {
        ...settings,
        recharge_promotion_percentage_rate: finiteMoney(settings.recharge_promotion_percentage_rate),
        recharge_promotion_minimum_amount: finiteMoney(settings.recharge_promotion_minimum_amount),
        recharge_promotion_maximum_bonus: finiteMoney(settings.recharge_promotion_maximum_bonus),
        recharge_promotion_tier_rules: tiers(settings.recharge_promotion_tier_rules),
        recharge_promotion_tiered_percentage_rules: tieredPercentages(settings.recharge_promotion_tiered_percentage_rules),
      }
    },
    validate: (form) => validateTieredPercentagePrecision(form) || validate(readSettings(form)),
  },
]
