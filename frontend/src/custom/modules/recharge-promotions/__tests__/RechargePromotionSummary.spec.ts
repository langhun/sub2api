import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import RechargePromotionSummary from '../RechargePromotionSummary.vue'
import { rechargePromotionsAPI } from '../api'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

vi.mock('../api', () => ({
  rechargePromotionsAPI: {
    preview: vi.fn(),
  },
}))

const preview = vi.mocked(rechargePromotionsAPI.preview)

function mountSummary(amount = 10, baseCredit = 10) {
  return mount(RechargePromotionSummary, {
    props: { amount, baseCredit },
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

  it('retains summary rows for empty, loading, eligible and ineligible amounts', async () => {
    const wrapper = mountSummary(0, 0)
    expect(wrapper.get('[data-testid="promotion-final"]').text()).toBe('$0.00')
    await wrapper.setProps({ amount: 10, baseCredit: 10 })
    expect(wrapper.attributes('aria-busy')).toBe('true')
    expect(wrapper.get('[data-testid="promotion-bonus"]').text()).toBe('\u2014')
    await vi.advanceTimersByTimeAsync(250)
    await flushPromises()
    expect(wrapper.get('[data-testid="promotion-bonus"]').text()).toBe('+$1.00')
    expect(wrapper.get('[data-testid="promotion-final"]').text()).toBe('$11.00')

    preview.mockResolvedValueOnce({ data: { active: false, ineligible_reason: 'minimum_amount', minimum_amount: 100 } } as never)
    await wrapper.setProps({ amount: 20, baseCredit: 20 })
    expect(wrapper.get('[data-testid="promotion-final"]').text()).toBe('\u2014')
    await vi.advanceTimersByTimeAsync(250)
    await flushPromises()
    expect(wrapper.get('[data-testid="promotion-bonus"]').text()).toBe('+$0.00')
    expect(wrapper.get('[data-testid="promotion-final"]').text()).toBe('$20.00')
    expect(wrapper.get('[role="status"]').text()).toBe('rechargePromotion.minimumAmount')
  })

  it('does not present a failed or stale preview as a confirmed credit', async () => {
    let resolveFirst!: (value: never) => void
    preview.mockImplementationOnce(() => new Promise(resolve => { resolveFirst = resolve }))
    const wrapper = mountSummary()
    await vi.advanceTimersByTimeAsync(250)
    preview.mockRejectedValueOnce(new Error('offline'))
    await wrapper.setProps({ amount: 20, baseCredit: 20 })
    await vi.advanceTimersByTimeAsync(250)
    await flushPromises()
    resolveFirst({ data: { active: true, bonus_credit: 99, final_credit: 109 } } as never)
    await flushPromises()
    expect(wrapper.get('[data-testid="promotion-final"]').text()).toBe('\u2014')
    expect(wrapper.get('[role="status"]').text()).toBe('rechargePromotion.loadFailed')
    expect(wrapper.attributes('aria-busy')).toBe('false')
  })
})
