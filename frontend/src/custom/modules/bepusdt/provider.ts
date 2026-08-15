import paymentIcon from '@/assets/icons/payment.svg'
import { registerCustomPaymentDefinition } from '../../paymentExtensions'

export const bepusdtProviderKey = 'bepusdt'

registerCustomPaymentDefinition({
  providerKey: bepusdtProviderKey,
  providerLabelKey: 'admin.settings.payment.providerBepusdt',
  supportedTypes: [bepusdtProviderKey],
  paymentTypes: [{ value: bepusdtProviderKey, label: 'payment.methods.bepusdt' }],
  displayNameConfigKey: 'displayName',
  defaultDisplayName: 'USDT',
  methodOrder: 70,
  configFields: [
    { key: 'displayName', label: '', sensitive: false, optional: true, clearable: true, defaultValue: 'USDT' },
    { key: 'apiBase', label: '', sensitive: false, defaultValue: 'https://example.com', hintKey: 'admin.settings.payment.field_bepusdtApiBaseHint' },
    { key: 'apiToken', label: '', sensitive: true },
    { key: 'fiat', label: '', sensitive: false, defaultValue: 'CNY' },
    { key: 'currencies', label: '', sensitive: false, defaultValue: 'USDT' },
    { key: 'timeout', label: '', sensitive: false, defaultValue: '600' },
    { key: 'reselect', label: '', sensitive: false, defaultValue: 'true' },
  ],
  callbackPaths: {
    notifyUrl: '/api/v1/payment/webhook/bepusdt',
    returnUrl: '/payment/result',
  },
  webhookHintKey: 'admin.settings.payment.bepusdtWebhookHint',
  methodIcon: paymentIcon,
  selectedClass: 'border-[#26A17B] bg-green-50 text-gray-900 shadow-sm dark:bg-green-950 dark:text-gray-100',
  buttonClass: 'btn-primary',
})
