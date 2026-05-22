import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import type { PaymentConfig, PaymentOrder, SubscriptionPlan } from '@/types/payment'

const { paymentAPIMock } = vi.hoisted(() => ({
  paymentAPIMock: {
    getConfig: vi.fn(),
    getPlans: vi.fn(),
    getOrder: vi.fn(),
  },
}))

vi.mock('@/api/payment', () => ({
  paymentAPI: paymentAPIMock,
}))

import { usePaymentStore } from '@/stores/payment'

function createConfig(overrides: Partial<PaymentConfig> = {}): PaymentConfig {
  return {
    payment_enabled: true,
    min_amount: 1,
    max_amount: 1000,
    daily_limit: 5000,
    max_pending_orders: 3,
    order_timeout_minutes: 30,
    balance_disabled: false,
    balance_recharge_multiplier: 1,
    enabled_payment_types: ['alipay'],
    help_image_url: '',
    help_text: 'pay help',
    stripe_publishable_key: 'pk_test_123',
    ...overrides,
  }
}

function createOrder(overrides: Partial<PaymentOrder> = {}): PaymentOrder {
  return {
    id: 1,
    user_id: 10,
    amount: 20,
    pay_amount: 20,
    currency: 'CNY',
    fee_rate: 0,
    payment_type: 'alipay',
    out_trade_no: 'order-1',
    status: 'PENDING',
    order_type: 'balance',
    created_at: '2026-05-22T00:00:00Z',
    expires_at: '2026-05-22T00:30:00Z',
    refund_amount: 0,
    ...overrides,
  }
}

function createDeferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((res, rej) => {
    resolve = res
    reject = rej
  })

  return { promise, resolve, reject }
}

describe('usePaymentStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  describe('fetchConfig', () => {
    it('缓存已加载配置并复用缓存结果', async () => {
      const config = createConfig()
      paymentAPIMock.getConfig.mockResolvedValueOnce({ data: config })
      const store = usePaymentStore()

      const first = await store.fetchConfig()
      const second = await store.fetchConfig()

      expect(first).toEqual(config)
      expect(second).toEqual(config)
      expect(store.config).toEqual(config)
      expect(store.configLoaded).toBe(true)
      expect(store.configLoading).toBe(false)
      expect(paymentAPIMock.getConfig).toHaveBeenCalledTimes(1)
    })

    it('force=true 时绕过缓存重新请求配置', async () => {
      const firstConfig = createConfig({ max_amount: 1000 })
      const secondConfig = createConfig({ max_amount: 2000, payment_enabled: false })
      paymentAPIMock.getConfig
        .mockResolvedValueOnce({ data: firstConfig })
        .mockResolvedValueOnce({ data: secondConfig })
      const store = usePaymentStore()

      await store.fetchConfig()
      const forced = await store.fetchConfig(true)

      expect(forced).toEqual(secondConfig)
      expect(store.config).toEqual(secondConfig)
      expect(store.configLoaded).toBe(true)
      expect(paymentAPIMock.getConfig).toHaveBeenCalledTimes(2)
    })

    it('请求进行中时返回当前缓存值且不重复发起请求', async () => {
      const deferred = createDeferred<{ data: PaymentConfig }>()
      const config = createConfig({ daily_limit: 8888 })
      paymentAPIMock.getConfig.mockReturnValueOnce(deferred.promise)
      const store = usePaymentStore()

      const firstRequest = store.fetchConfig()
      expect(store.configLoading).toBe(true)

      const secondRequest = store.fetchConfig()

      await expect(secondRequest).resolves.toBeNull()
      expect(paymentAPIMock.getConfig).toHaveBeenCalledTimes(1)

      deferred.resolve({ data: config })
      await expect(firstRequest).resolves.toEqual(config)
      expect(store.config).toEqual(config)
      expect(store.configLoaded).toBe(true)
      expect(store.configLoading).toBe(false)
    })
  })

  describe('fetchPlans', () => {
    it('将换行字符串 features 拆分为数组并过滤空白项', async () => {
      const rawPlans: Array<Omit<SubscriptionPlan, 'features'> & { features: string | string[] }> = [
        {
          id: 1,
          group_id: 1,
          name: 'Pro Plan',
          description: 'for power users',
          price: 99,
          validity_days: 30,
          validity_unit: 'day',
          features: ' 专属模型 \n\n 高峰优先 \n 多设备使用 ',
          for_sale: true,
          sort_order: 1,
        },
        {
          id: 2,
          group_id: 2,
          name: 'Team Plan',
          description: 'for teams',
          price: 199,
          validity_days: 30,
          validity_unit: 'day',
          features: ['共享额度'],
          for_sale: true,
          sort_order: 2,
        },
      ]
      paymentAPIMock.getPlans.mockResolvedValueOnce({ data: rawPlans })
      const store = usePaymentStore()

      const plans = await store.fetchPlans()

      expect(plans).toHaveLength(2)
      expect(plans[0].features).toEqual(['专属模型', '高峰优先', '多设备使用'])
      expect(plans[1].features).toEqual(['共享额度'])
      expect(store.plans).toEqual(plans)
      expect(paymentAPIMock.getPlans).toHaveBeenCalledTimes(1)
    })
  })

  describe('pollOrderStatus', () => {
    it('仅在 orderId 匹配 currentOrder 时更新当前订单', async () => {
      const store = usePaymentStore()
      const current = createOrder({ id: 42, status: 'PENDING', out_trade_no: 'order-42' })
      const updated = createOrder({ id: 42, status: 'PAID', out_trade_no: 'order-42' })
      store.currentOrder = current
      paymentAPIMock.getOrder.mockResolvedValueOnce({ data: updated })

      const result = await store.pollOrderStatus(42)

      expect(result).toEqual(updated)
      expect(store.currentOrder).toEqual(updated)
      expect(paymentAPIMock.getOrder).toHaveBeenCalledWith(42)
    })

    it('orderId 不匹配时保持现有 currentOrder 不变', async () => {
      const store = usePaymentStore()
      const current = createOrder({ id: 42, status: 'PENDING', out_trade_no: 'order-42' })
      const fetched = createOrder({ id: 99, status: 'COMPLETED', out_trade_no: 'order-99' })
      store.currentOrder = current
      paymentAPIMock.getOrder.mockResolvedValueOnce({ data: fetched })

      const result = await store.pollOrderStatus(99)

      expect(result).toEqual(fetched)
      expect(store.currentOrder).toEqual(current)
      expect(paymentAPIMock.getOrder).toHaveBeenCalledWith(99)
    })
  })

  describe('clearCurrentOrder', () => {
    it('清空当前订单状态', () => {
      const store = usePaymentStore()
      store.currentOrder = createOrder({ id: 123, out_trade_no: 'order-123' })

      store.clearCurrentOrder()

      expect(store.currentOrder).toBeNull()
    })
  })
})
