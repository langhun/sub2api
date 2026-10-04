import { describe, expect, it } from 'vitest'

import { paymentDisplaySettingsPanels } from '../settings'
import { filterQuickRechargeAmounts } from '../quickAmounts'

describe('payment display settings Overlay', () => {
  const panel = paymentDisplaySettingsPanels[0]

  it('defaults to hiding the large preset amounts', () => {
    expect(panel.createForm()).toEqual({ payment_quick_amount_max: 500 })
    expect(panel.fromSettings({})).toEqual({ payment_quick_amount_max: 500 })
  })

  it('accepts zero as unlimited and rejects negative values', () => {
    expect(panel.toPayload({ payment_quick_amount_max: 0 })).toEqual({ payment_quick_amount_max: 0 })
    expect(panel.validate({ payment_quick_amount_max: 0 }, {})).toBe('')
    expect(panel.validate({ payment_quick_amount_max: -1 }, {})).not.toBe('')
  })

  it('keeps only the supported preset buttons even when the setting is unlimited', () => {
    expect(filterQuickRechargeAmounts(500)).toEqual([10, 20, 50, 100, 200, 500])
    expect(filterQuickRechargeAmounts(0)).toEqual([10, 20, 50, 100, 200, 500])
    expect(filterQuickRechargeAmounts('invalid')).toEqual([10, 20, 50, 100, 200, 500])
  })
})
