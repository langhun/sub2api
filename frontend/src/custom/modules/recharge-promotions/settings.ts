import type { CustomSettingsPanel, CustomSettingsValues } from '../../registry'
import RechargePromotionSettingsPanel from './admin/RechargePromotionSettingsPanel.vue'

export interface RechargePromotionSettings {
  recharge_promotion_enabled: boolean
  recharge_promotion_bonus_rate: number
  recharge_promotion_min_amount: number
  recharge_promotion_max_bonus: number
  recharge_promotion_start_at: string
  recharge_promotion_end_at: string
}

const defaults = (): RechargePromotionSettings => ({
  recharge_promotion_enabled: false,
  recharge_promotion_bonus_rate: 0,
  recharge_promotion_min_amount: 0,
  recharge_promotion_max_bonus: 0,
  recharge_promotion_start_at: '',
  recharge_promotion_end_at: '',
})

function readSettings(settings: CustomSettingsValues): RechargePromotionSettings {
  const result = defaults()
  const values = settings as Record<string, unknown>
  for (const key of Object.keys(result) as Array<keyof RechargePromotionSettings>) {
    if (values[key] !== null && values[key] !== undefined) {
      result[key] = values[key] as never
    }
  }
  return result
}

function validNumber(value: unknown): value is number {
  return typeof value === 'number' && Number.isFinite(value) && value >= 0
}

export const rechargePromotionSettingsPanels: readonly CustomSettingsPanel[] = [{
  id: 'recharge-promotion',
  placement: 'features',
  order: 30,
  component: RechargePromotionSettingsPanel,
  settingKeys: Object.keys(defaults()),
  createForm: defaults,
  fromSettings: readSettings,
  toPayload: (form) => {
    const values = form as unknown as RechargePromotionSettings
    return {
      recharge_promotion_enabled: values.recharge_promotion_enabled,
      recharge_promotion_bonus_rate: Number(values.recharge_promotion_bonus_rate) || 0,
      recharge_promotion_min_amount: Number(values.recharge_promotion_min_amount) || 0,
      recharge_promotion_max_bonus: Number(values.recharge_promotion_max_bonus) || 0,
      recharge_promotion_start_at: values.recharge_promotion_start_at || '',
      recharge_promotion_end_at: values.recharge_promotion_end_at || '',
    }
  },
  validate: (form) => {
    const values = form as unknown as RechargePromotionSettings
    if (!validNumber(Number(values.recharge_promotion_bonus_rate)) || Number(values.recharge_promotion_bonus_rate) > 100) return '赠送比例必须在 0 到 100 之间'
    if (values.recharge_promotion_enabled && Number(values.recharge_promotion_bonus_rate) <= 0) return '开启活动时赠送比例必须大于 0'
    if (!validNumber(Number(values.recharge_promotion_min_amount)) || !validNumber(Number(values.recharge_promotion_max_bonus))) return '金额必须是非负有效数字'
    const start = values.recharge_promotion_start_at ? Date.parse(values.recharge_promotion_start_at) : NaN
    const end = values.recharge_promotion_end_at ? Date.parse(values.recharge_promotion_end_at) : NaN
    if (values.recharge_promotion_start_at && Number.isNaN(start)) return '开始时间必须是有效时间'
    if (values.recharge_promotion_end_at && Number.isNaN(end)) return '结束时间必须是有效时间'
    if (!Number.isNaN(start) && !Number.isNaN(end) && start >= end) return '开始时间必须早于结束时间'
    return ''
  },
}]
