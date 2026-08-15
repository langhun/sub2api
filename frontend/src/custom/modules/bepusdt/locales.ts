export const bepusdtLocaleMessages = {
  en: {
    payment: {
      methods: { bepusdt: 'USDT' },
    },
  },
  zh: {
    payment: {
      methods: { bepusdt: 'USDT' },
    },
  },
} as const

export const bepusdtAdminLocaleMessages = {
  en: {
    settings: {
      payment: {
        providerBepusdt: 'BEpusdt',
        field_displayName: 'User-facing payment name',
        field_bepusdtApiBaseHint: 'Enter the BEpusdt service URL, for example https://pay.example.com. The API path /api/v1/order/create-order is added automatically.',
        field_timeout: 'BEpusdt order timeout (seconds)',
        bepusdtWebhookHint: 'Configure the following URL as the asynchronous notify URL in BEpusdt. Successful payments send a JSON callback with status=2.',
      },
    },
  },
  zh: {
    settings: {
      payment: {
        providerBepusdt: 'BEpusdt',
        field_displayName: '用户侧支付方式名称',
        field_bepusdtApiBaseHint: '填写 BEpusdt 服务地址，例如 https://pay.example.com；系统会自动调用 /api/v1/order/create-order。',
        field_timeout: 'BEpusdt 订单超时（秒）',
        bepusdtWebhookHint: '请在 BEpusdt 后台将以下地址配置为异步通知地址。支付成功后 BEpusdt 会发送 status=2 的 JSON 回调。',
      },
    },
  },
} as const
