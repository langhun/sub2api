import { flushPromises, mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { describe, expect, it } from 'vitest'

import { rechargePromotionSettingsPanels } from '../settings'
import RechargePromotionSettingsPanel from '../admin/RechargePromotionSettingsPanel.vue'

const panel = rechargePromotionSettingsPanels[0]

describe('recharge promotion settings Overlay', () => {
  it('normalizes tier rules without changing the selected campaign mode', () => {
    const payload = panel.toPayload({
      recharge_promotion_enabled: true,
      recharge_promotion_mode: 'tiered',
      recharge_promotion_percentage_rate: 0,
      recharge_promotion_minimum_amount: 0,
      recharge_promotion_maximum_bonus: 0,
      recharge_promotion_tier_rules: [
        { minimum_amount: 50.004, bonus_credit: 8.006 },
        { minimum_amount: 10.004, bonus_credit: 1.004 },
      ],
      recharge_promotion_start_at: '',
      recharge_promotion_end_at: '',
    }) as Record<string, unknown>

    expect(payload.recharge_promotion_mode).toBe('tiered')
    expect(payload.recharge_promotion_tier_rules).toEqual([
      { minimum_amount: 10, bonus_credit: 1 },
      { minimum_amount: 50, bonus_credit: 8.01 },
    ])
  })

  it('rejects an enabled tiered campaign with an invalid rule', () => {
    const error = panel.validate({
      recharge_promotion_enabled: true,
      recharge_promotion_mode: 'tiered',
      recharge_promotion_percentage_rate: 0,
      recharge_promotion_minimum_amount: 0,
      recharge_promotion_maximum_bonus: 0,
      recharge_promotion_tier_rules: [
        { minimum_amount: 0, bonus_credit: 1 },
      ],
      recharge_promotion_start_at: '',
      recharge_promotion_end_at: '',
    })

    expect(error).toContain('必须大于 0')
  })

  it('does not re-emit unchanged parent values after datetime normalization', async () => {
    const settings = {
      recharge_promotion_enabled: false,
      recharge_promotion_mode: 'percentage' as const,
      recharge_promotion_percentage_rate: 0,
      recharge_promotion_minimum_amount: 0,
      recharge_promotion_maximum_bonus: 0,
      recharge_promotion_tier_rules: [],
      recharge_promotion_start_at: '2026-08-12T00:00:00Z',
      recharge_promotion_end_at: '',
    }
    const wrapper = mount(RechargePromotionSettingsPanel, {
      props: { modelValue: settings },
      global: {
        stubs: { Toggle: true, RechargePromotionAdminPanel: true },
        plugins: [createI18n({ legacy: false, locale: 'en', messages: { en: { admin: { rechargePromotion: {} } } } })],
      },
    })

    await flushPromises()
    expect(wrapper.emitted('update:modelValue')).toBeUndefined()

    await wrapper.setProps({ modelValue: { ...settings } })
    await flushPromises()
    expect(wrapper.emitted('update:modelValue')).toBeUndefined()
  })

  it('renders stored UTC campaign times as browser-local datetime values', () => {
    const stored = '2026-08-12T00:30:00Z'
    const date = new Date(stored)
    const expected = [
      date.getFullYear(),
      String(date.getMonth() + 1).padStart(2, '0'),
      String(date.getDate()).padStart(2, '0'),
      String(date.getHours()).padStart(2, '0'),
      String(date.getMinutes()).padStart(2, '0'),
    ]
    const wrapper = mount(RechargePromotionSettingsPanel, {
      props: {
        modelValue: {
          recharge_promotion_enabled: false,
          recharge_promotion_mode: 'percentage',
          recharge_promotion_percentage_rate: 0,
          recharge_promotion_minimum_amount: 0,
          recharge_promotion_maximum_bonus: 0,
          recharge_promotion_tier_rules: [],
          recharge_promotion_start_at: stored,
          recharge_promotion_end_at: '',
        },
      },
      global: {
        stubs: { Toggle: true, RechargePromotionAdminPanel: true },
        plugins: [createI18n({ legacy: false, locale: 'en', messages: { en: { admin: { rechargePromotion: {} } } } })],
      },
    })

    expect(wrapper.get('input[type="datetime-local"]').element).toHaveProperty(
      'value',
      `${expected[0]}-${expected[1]}-${expected[2]}T${expected[3]}:${expected[4]}`,
    )
  })
})
