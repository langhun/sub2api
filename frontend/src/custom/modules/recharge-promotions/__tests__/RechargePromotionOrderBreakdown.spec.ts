import { createI18n } from 'vue-i18n'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const api = vi.hoisted(() => ({
  getUserOrder: vi.fn(),
  getAdminOrder: vi.fn(),
}))

vi.mock('../api', () => ({ rechargePromotionsAPI: api }))

import RechargePromotionOrderBreakdown from '../RechargePromotionOrderBreakdown.vue'

function mountBreakdown(admin = false) {
  return mount(RechargePromotionOrderBreakdown, {
    props: { orderId: 42, admin },
    global: {
      plugins: [createI18n({
        legacy: false,
        locale: 'zh',
        messages: { zh: { rechargePromotion: {
          detail: '到账明细', baseCredit: '基础到账', bonusCredit: '活动赠送', finalCredit: '最终到账',
          refundedBase: '已退基础到账', refundedBonus: '已退活动赠送', refundedTotal: '已退到账额度',
        } } },
      })],
    },
  })
}

const record = {
  payment_order_id: 42,
  user_id: 9,
  mode: 'percentage',
  payment_amount: 100,
  pay_amount: 100,
  base_credit: 120,
  bonus_credit: 12,
  final_credit: 132,
  refunded_credit: 66,
  refunded_base_credit: 60,
  refunded_bonus_credit: 6,
  created_at: '2026-08-12T00:00:00Z',
  completed_at: '2026-08-12T00:01:00Z',
}

describe('RechargePromotionOrderBreakdown', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    api.getUserOrder.mockResolvedValue({ data: record })
    api.getAdminOrder.mockResolvedValue({ data: record })
  })

  it('shows immutable base, bonus, final and refund split for a user order', async () => {
    const wrapper = mountBreakdown()
    await flushPromises()

    expect(api.getUserOrder).toHaveBeenCalledWith(42)
    expect(wrapper.text()).toContain('$120.00')
    expect(wrapper.text()).toContain('+$12.00')
    expect(wrapper.text()).toContain('$132.00')
    expect(wrapper.text()).toContain('-$60.00')
    expect(wrapper.text()).toContain('-$6.00')
  })

  it('uses the administrator-only order route inside the admin mount point', async () => {
    mountBreakdown(true)
    await flushPromises()

    expect(api.getAdminOrder).toHaveBeenCalledWith(42)
    expect(api.getUserOrder).not.toHaveBeenCalled()
  })
})
