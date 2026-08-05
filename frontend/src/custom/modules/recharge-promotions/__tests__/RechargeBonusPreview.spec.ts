import { createI18n } from 'vue-i18n'
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import RechargeBonusPreview from '../RechargeBonusPreview.vue'

const appStore = { cachedPublicSettings: {} as Record<string, unknown> }
vi.mock('@/stores', () => ({ useAppStore: () => appStore }))

function render(settings: Record<string, unknown>, amount = 100, creditedAmount = 120) {
  appStore.cachedPublicSettings = settings
  return mount(RechargeBonusPreview, {
    props: { amount, creditedAmount },
    global: { plugins: [createI18n({ legacy: false, locale: 'en', messages: { en: {} } })] },
  })
}

describe('RechargeBonusPreview', () => {
  it('shows the capped bonus and final credit for an active campaign', () => {
    const wrapper = render({
      recharge_promotion_enabled: true,
      recharge_promotion_bonus_rate: 20,
      recharge_promotion_min_amount: 10,
      recharge_promotion_max_bonus: 20,
      recharge_promotion_start_at: '',
      recharge_promotion_end_at: '',
    })
    expect(wrapper.text()).toContain('Promotion bonus')
    expect(wrapper.text()).toContain('+$20.00')
    expect(wrapper.text()).toContain('$140.00')
  })

  it('hides the bonus when the campaign is unavailable', () => {
    const wrapper = render({ recharge_promotion_enabled: false })
    expect(wrapper.text()).not.toContain('Promotion bonus')
  })
})
