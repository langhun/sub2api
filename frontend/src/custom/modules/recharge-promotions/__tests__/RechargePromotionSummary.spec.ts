import { flushPromises, mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import RechargePromotionSummary from '../RechargePromotionSummary.vue'
import { rechargePromotionsAPI } from '../api'

vi.mock('../api', () => ({
  rechargePromotionsAPI: {
    preview: vi.fn(),
  },
}))

const preview = vi.mocked(rechargePromotionsAPI.preview)

function mountSummary(amount = 10, baseCredit = 10) {
  return mount(RechargePromotionSummary, {
    props: { amount, baseCredit },
    global: {
      plugins: [createI18n({
        legacy: false,
        locale: 'en',
        messages: { en: { common: { loading: 'Loading' }, rechargePromotion: {} } },
      })],
    },
  })
}

describe('RechargePromotionSummary', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    preview.mockResolvedValue({
      data: {
        active: true,
        mode: 'percentage',
        base_credit: 10,
        bonus_credit: 1,
        final_credit: 11,
        ineligible_reason: '',
        minimum_amount: 0,
        percentage_rate: 10,
        maximum_bonus: 0,
        tier_rules: [],
        tiered_percentage_rules: [],
        matched_minimum_amount: 0,
        matched_maximum_amount: 0,
        applied_percentage_rate: 10,
        campaign_started_at: '',
        campaign_ended_at: '',
      },
    } as never)
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.clearAllMocks()
  })

  it('debounces amount changes into one advisory preview request', async () => {
    const wrapper = mountSummary()
    await wrapper.setProps({ amount: 20 })
    await wrapper.setProps({ amount: 30 })
    await wrapper.setProps({ amount: 40 })

    await vi.advanceTimersByTimeAsync(249)
    expect(preview).not.toHaveBeenCalled()

    await vi.advanceTimersByTimeAsync(1)
    await flushPromises()
    expect(preview).toHaveBeenCalledTimes(1)
    expect(preview).toHaveBeenCalledWith(40, 10)
  })

  it('cancels a pending preview when unmounted', async () => {
    const wrapper = mountSummary()
    wrapper.unmount()

    await vi.advanceTimersByTimeAsync(250)
    expect(preview).not.toHaveBeenCalled()
  })
})
