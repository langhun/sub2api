export const gmpayEdgeLocaleMessages = {
  en: {
    payment: {
      methods: { gmpayEdge: 'USDT / USDC' },
    },
  },
  zh: {
    payment: {
      methods: { gmpayEdge: 'USDT / USDC' },
    },
  },
} as const

export const gmpayEdgeAdminLocaleMessages = {
  en: {
    settings: {
      payment: {
        providerGmpayEdge: 'GMPay Edge',
        field_apiSecret: 'API Secret',
        field_gmpayEdgeApiBaseHint: 'Enter the GMPay Edge EPay base URL, ending with /payments/epay/v1/order/create-transaction. The /mapi.php and /api.php paths are added automatically.',
        gmpayEdgeWebhookHint: 'Configure this URL as the EPay notify URL in GMPay Edge. GMPay Edge signs the callback with the API Secret and expects a plain-text ok acknowledgement.',
      },
    },
  },
  zh: {
    settings: {
      payment: {
        providerGmpayEdge: 'GMPay Edge',
        field_apiSecret: 'API Secret',
        field_gmpayEdgeApiBaseHint: '填写 GMPay Edge 的 EPay 基础地址，末尾为 /payments/epay/v1/order/create-transaction；系统会自动追加 /mapi.php 和 /api.php。',
        gmpayEdgeWebhookHint: '请将此地址填入 GMPay Edge 的 EPay 异步通知地址。GMPay Edge 使用 API Secret 签名，并要求返回纯文本 ok。',
      },
    },
  },
} as const
