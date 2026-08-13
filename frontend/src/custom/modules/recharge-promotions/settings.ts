import type { CustomSettingsPanel, CustomSettingsValues } from '../../registry'
import RechargePromotionSettingsPanel from './admin/RechargePromotionSettingsPanel.vue'
import type { RechargePromotionMode, RechargePromotionTierRule } from './api'

interface RechargePromotionSettings {
  recharge_promotion_enabled: boolean
  recharge_promotion_mode: RechargePromotionMode
  recharge_promotion_percentage_rate: number
  recharge_promotion_minimum_amount: number
  recharge_promotion_maximum_bonus: number
  recharge_promotion_tier_rules: RechargePromotionTierRule[]
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

function readSettings(value: CustomSettingsValues): RechargePromotionSettings {
  const values = value as Record<string, unknown>
  const defaults = defaultRechargePromotionSettings()
  return {
    recharge_promotion_enabled: values.recharge_promotion_enabled === true,
    recharge_promotion_mode: values.recharge_promotion_mode === 'tiered' ? 'tiered' : 'percentage',
    recharge_promotion_percentage_rate: finiteMoney(values.recharge_promotion_percentage_rate),
    recharge_promotion_minimum_amount: finiteMoney(values.recharge_promotion_minimum_amount),
    recharge_promotion_maximum_bonus: finiteMoney(values.recharge_promotion_maximum_bonus),
    recharge_promotion_tier_rules: tiers(values.recharge_promotion_tier_rules),
    recharge_promotion_start_at: typeof values.recharge_promotion_start_at === 'string' ? values.recharge_promotion_start_at : defaults.recharge_promotion_start_at,
    recharge_promotion_end_at: typeof values.recharge_promotion_end_at === 'string' ? values.recharge_promotion_end_at : defaults.recharge_promotion_end_at,
  }
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
      }
    },
    validate: (form) => validate(readSettings(form)),
  },
]
