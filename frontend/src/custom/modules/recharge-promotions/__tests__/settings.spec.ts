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
      recharge_promotion_tiered_percentage_rules: [],
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
      recharge_promotion_tiered_percentage_rules: [],
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
      recharge_promotion_tiered_percentage_rules: [],
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
          recharge_promotion_tiered_percentage_rules: [],
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

  it('normalizes and preserves tiered percentage rules', () => {
    const payload = panel.toPayload({
      recharge_promotion_enabled: true,
      recharge_promotion_mode: 'tiered_percentage',
      recharge_promotion_percentage_rate: 0,
      recharge_promotion_minimum_amount: 0,
      recharge_promotion_maximum_bonus: 20.006,
      recharge_promotion_tier_rules: [],
      recharge_promotion_tiered_percentage_rules: [
        { minimum_amount: 50, maximum_amount: 0, percentage_rate: 8 },
        { minimum_amount: 10, maximum_amount: 50, percentage_rate: 5 },
      ],
      recharge_promotion_start_at: '',
      recharge_promotion_end_at: '',
    }) as Record<string, unknown>

    expect(payload.recharge_promotion_mode).toBe('tiered_percentage')
    expect(payload.recharge_promotion_maximum_bonus).toBe(20.01)
    expect(payload.recharge_promotion_tiered_percentage_rules).toEqual([
      { minimum_amount: 10, maximum_amount: 50, percentage_rate: 5 },
      { minimum_amount: 50, maximum_amount: 0, percentage_rate: 8 },
    ])
  })

  it.each([
    {
      name: 'a gap',
      rules: [
        { minimum_amount: 10, maximum_amount: 49, percentage_rate: 5 },
        { minimum_amount: 50, maximum_amount: 0, percentage_rate: 8 },
      ],
      expected: '连续',
    },
    {
      name: 'a decreasing rate',
      rules: [
        { minimum_amount: 10, maximum_amount: 50, percentage_rate: 8 },
        { minimum_amount: 50, maximum_amount: 0, percentage_rate: 5 },
      ],
      expected: '非递减',
    },
    {
      name: 'a bounded final tier',
      rules: [{ minimum_amount: 10, maximum_amount: 50, percentage_rate: 5 }],
      expected: '最后一档',
    },
    {
      name: 'more than two decimals',
      rules: [{ minimum_amount: 10.001, maximum_amount: 0, percentage_rate: 5 }],
      expected: '两位小数',
    },
  ])('rejects tiered percentage rules with $name', ({ rules, expected }) => {
    const error = panel.validate({
      recharge_promotion_enabled: true,
      recharge_promotion_mode: 'tiered_percentage',
      recharge_promotion_percentage_rate: 0,
      recharge_promotion_minimum_amount: 0,
      recharge_promotion_maximum_bonus: 0,
      recharge_promotion_tier_rules: [],
      recharge_promotion_tiered_percentage_rules: rules,
      recharge_promotion_start_at: '',
      recharge_promotion_end_at: '',
    }, {})

    expect(error).toContain(expected)
  })

  it('renders and adds tiered percentage rules independently from fixed-credit tiers', async () => {
    const wrapper = mount(RechargePromotionSettingsPanel, {
      props: {
        modelValue: {
          recharge_promotion_enabled: true,
          recharge_promotion_mode: 'tiered_percentage',
          recharge_promotion_percentage_rate: 0,
          recharge_promotion_minimum_amount: 0,
          recharge_promotion_maximum_bonus: 0,
          recharge_promotion_tier_rules: [],
          recharge_promotion_tiered_percentage_rules: [
            { minimum_amount: 10, maximum_amount: 0, percentage_rate: 5 },
          ],
          recharge_promotion_start_at: '',
          recharge_promotion_end_at: '',
        },
      },
      global: {
        stubs: { Toggle: true, RechargePromotionAdminPanel: true },
        plugins: [createI18n({ legacy: false, locale: 'en', messages: { en: { admin: { rechargePromotion: {
          title: 'Recharge promotion', description: '', enabled: 'Enabled', mode: 'Mode', percentage: 'Percentage', tiered: 'Fixed tiers',
          tieredPercentage: 'Tiered percentage', tieredPercentageRules: 'Tiered percentage rules', tieredPercentageHint: '', addTier: 'Add tier',
          tierMinimum: 'Minimum', tierMaximum: 'Maximum', percentageRate: 'Rate', unlimitedMaximum: 'Unlimited', removeTier: 'Remove',
          maximumBonus: 'Maximum bonus', noCap: 'No cap', startAt: 'Start', endAt: 'End', optionalTime: '',
        } } } } })],
      },
    })

    expect(wrapper.text()).toContain('admin.rechargePromotion.tieredPercentageRules')
    expect(wrapper.findAll('input[type="number"]')).toHaveLength(4)
    expect(wrapper.findAll('input[type="number"]')[1].attributes('readonly')).toBeDefined()
    await wrapper.get('button.btn-sm').trigger('click')
    await flushPromises()
    const emitted = wrapper.emitted('update:modelValue')?.at(-1)?.[0] as { recharge_promotion_tiered_percentage_rules: Array<{ minimum_amount: number; maximum_amount: number }> }
    expect(emitted.recharge_promotion_tiered_percentage_rules).toEqual([
      expect.objectContaining({ minimum_amount: 10, maximum_amount: 20 }),
      expect.objectContaining({ minimum_amount: 20, maximum_amount: 0 }),
    ])
  })
})
