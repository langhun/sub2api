import { createI18n } from 'vue-i18n'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const api = vi.hoisted(() => ({
  getSummary: vi.fn(),
  getAdminOrders: vi.fn(),
}))

vi.mock('../../api', () => ({ rechargePromotionsAPI: api }))

import RechargePromotionAdminPanel from '../RechargePromotionAdminPanel.vue'

function mountPanel() {
  return mount(RechargePromotionAdminPanel, {
    global: {
      plugins: [createI18n({
        legacy: false,
        locale: 'en',
        messages: { en: { admin: { rechargePromotion: {
          summary: 'Promotion summary', enabledStatus: 'Enabled', disabledStatus: 'Disabled', refresh: 'Refresh',
          promotedOrders: 'Promoted orders', totalBonus: 'Total bonus', totalFinalCredit: 'Final credit', refundedBonus: 'Refunded bonus',
          details: 'Order details', allModes: 'All modes', percentageMode: 'Percentage', tieredMode: 'Tiered', tieredPercentageMode: 'Tiered percentage', order: 'Order', user: 'User',
          baseCredit: 'Base credit', bonusCredit: 'Bonus credit', finalCredit: 'Final credit', refundedCredit: 'Refunded credit', completedAt: 'Credited at',
          loadFailed: 'Unable to load recharge promotion data.',
        } } } },
      })],
    },
  })
}

describe('RechargePromotionAdminPanel', () => {
  beforeEach(() => vi.clearAllMocks())

  it('keeps the settings page usable when promotion APIs are unavailable', async () => {
    api.getSummary.mockRejectedValue(new Error('unavailable'))
    api.getAdminOrders.mockRejectedValue(new Error('unavailable'))

    const wrapper = mountPanel()
    await flushPromises()

    expect(wrapper.get('[data-testid="recharge-promotion-load-failed"]').exists()).toBe(true)
    expect(wrapper.get('select').exists()).toBe(true)
  })

  it('renders returned promotion totals and orders', async () => {
    api.getSummary.mockResolvedValue({ data: { enabled: true, orders: 1, bonus_credit: 2, final_credit: 12, refunded_bonus_credit: 0 } })
    api.getAdminOrders.mockResolvedValue({ data: { items: [{ payment_order_id: 10, user_id: 4, base_credit: 10, bonus_credit: 2, final_credit: 12, refunded_credit: 0, created_at: '2025-08-11T00:00:00Z', completed_at: '2026-08-12T00:00:00Z' }], total: 1 } })

    const wrapper = mountPanel()
    await flushPromises()

    expect(wrapper.text()).toContain('$2.00')
    expect(wrapper.text()).toContain('#10')
    expect(wrapper.text()).toContain('admin.rechargePromotion.completedAt')
    expect(wrapper.text()).toContain('2026')
    expect(wrapper.text()).not.toContain('2025')
  })

  it('resets pagination when filtering tiered percentage orders', async () => {
    api.getSummary.mockResolvedValue({ data: { enabled: true, orders: 11, bonus_credit: 2, final_credit: 12, refunded_bonus_credit: 0 } })
    api.getAdminOrders.mockResolvedValue({ data: { items: [], total: 11 } })

    const wrapper = mountPanel()
    await flushPromises()
    const buttons = wrapper.findAll('button')
    await buttons.at(-1)!.trigger('click')
    await flushPromises()
    expect(api.getAdminOrders).toHaveBeenLastCalledWith({ page: 2, page_size: 10, mode: undefined })

    await wrapper.get('[data-testid="recharge-promotion-mode-filter"]').setValue('tiered_percentage')
    await flushPromises()
    expect(api.getAdminOrders).toHaveBeenLastCalledWith({ page: 1, page_size: 10, mode: 'tiered_percentage' })
  })
})
