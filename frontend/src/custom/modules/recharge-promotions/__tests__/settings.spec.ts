import { describe, expect, it } from 'vitest'
import { rechargePromotionSettingsPanels } from '../settings'

describe('recharge promotion settings', () => {
  const panel = rechargePromotionSettingsPanels[0]

  it('keeps public-compatible campaign values from admin settings', () => {
    const form = panel.fromSettings({
      recharge_promotion_enabled: true,
      recharge_promotion_bonus_rate: 20,
      recharge_promotion_min_amount: 10,
      recharge_promotion_max_bonus: 30,
      recharge_promotion_start_at: '2026-08-05T00:00:00Z',
    }) as Record<string, unknown>

    expect(form.recharge_promotion_enabled).toBe(true)
    expect(form.recharge_promotion_bonus_rate).toBe(20)
    expect(form.recharge_promotion_min_amount).toBe(10)
  })

  it('rejects an enabled campaign without a usable rate or time window', () => {
    expect(panel.validate({ recharge_promotion_enabled: true, recharge_promotion_bonus_rate: 0 } as object, {})).toContain('赠送比例')
    expect(panel.validate({
      recharge_promotion_enabled: true,
      recharge_promotion_bonus_rate: 10,
      recharge_promotion_min_amount: 0,
      recharge_promotion_max_bonus: 0,
      recharge_promotion_start_at: '2026-08-06T00:00:00Z',
      recharge_promotion_end_at: '2026-08-05T00:00:00Z',
    } as object, {})).toContain('开始时间')
  })
})
