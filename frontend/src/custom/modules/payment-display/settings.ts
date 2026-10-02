import type { CustomSettingsPanel, CustomSettingsValues } from '../../registry'
import PaymentDisplaySettingsPanel from './PaymentDisplaySettingsPanel.vue'

interface PaymentDisplaySettings {
  payment_quick_amount_max: number
}

const defaultPaymentDisplaySettings = (): PaymentDisplaySettings => ({
  payment_quick_amount_max: 500,
})

function readSettings(settings: CustomSettingsValues): PaymentDisplaySettings {
  const values = settings as Record<string, unknown>
  const max = Number(values.payment_quick_amount_max)
  return {
    payment_quick_amount_max: Number.isFinite(max) && max >= 0 ? max : 500,
  }
}

export const paymentDisplaySettingsPanels: readonly CustomSettingsPanel[] = [
  {
    id: 'payment-display',
    placement: 'site',
    order: 25,
    component: PaymentDisplaySettingsPanel,
    settingKeys: ['payment_quick_amount_max'],
    createForm: defaultPaymentDisplaySettings,
    fromSettings: readSettings,
    toPayload: (form) => readSettings(form),
    validate: (form) => {
      const value = Number((form as Partial<PaymentDisplaySettings>).payment_quick_amount_max)
      return Number.isFinite(value) && value >= 0
        ? ''
        : '快捷充值金额上限必须是 0 或更大的有效数字'
    },
  },
]
