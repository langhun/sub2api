import paymentIcon from '@/assets/icons/payment.svg'
import { registerCustomPaymentDefinition } from '../../paymentExtensions'

export const gmpayEdgeProviderKey = 'gmpay_edge'

registerCustomPaymentDefinition({
  providerKey: gmpayEdgeProviderKey,
  providerLabelKey: 'admin.settings.payment.providerGmpayEdge',
  supportedTypes: [gmpayEdgeProviderKey],
  paymentTypes: [{ value: gmpayEdgeProviderKey, label: 'payment.methods.gmpayEdge' }],
  displayNameConfigKey: 'displayName',
  defaultDisplayName: 'USDT / USDC',
  methodOrder: 70,
  configFields: [
    { key: 'displayName', label: '', sensitive: false, optional: true, clearable: true, defaultValue: 'USDT / USDC' },
    { key: 'pid', label: 'PID', sensitive: false },
    { key: 'apiSecret', label: '', sensitive: true },
    { key: 'apiBase', label: '', sensitive: false, defaultValue: 'https://pay.example.com/payments/epay/v1/order/create-transaction', hintKey: 'admin.settings.payment.field_gmpayEdgeApiBaseHint' },
  ],
  callbackPaths: {
    notifyUrl: '/api/v1/payment/webhook/gmpay_edge',
    returnUrl: '/payment/result',
  },
  webhookHintKey: 'admin.settings.payment.gmpayEdgeWebhookHint',
  methodIcon: paymentIcon,
  selectedClass: 'border-[#16A085] bg-emerald-50 text-gray-900 shadow-sm dark:bg-emerald-950 dark:text-gray-100',
  buttonClass: 'btn-primary',
})
