import { describe, expect, it } from 'vitest'

import { formatDualDisplayAmount, formatSignedCurrency } from '../format'

describe('formatSignedCurrency', () => {
  it('formats positive negative and zero values without duplicate signs', () => {
    expect(formatSignedCurrency(12.34)).toBe('+$12.34')
    expect(formatSignedCurrency(-12.34)).toBe('-$12.34')
    expect(formatSignedCurrency(0)).toBe('$0.00')
  })
})

describe('formatDualDisplayAmount', () => {
  it('formats small values without compact suffix', () => {
    expect(formatDualDisplayAmount(999.12, { currencySymbol: '$' })).toEqual({
      compact: '$999.12',
      chinese: '$999.12',
      display: '$999.12',
      full: '$999.12',
    })
  })

  it('formats thousands and ten-thousands', () => {
    expect(formatDualDisplayAmount(1500)).toEqual({
      compact: '1.5K',
      chinese: '1500',
      display: '1.5K（1500）',
      full: '1,500',
    })
    expect(formatDualDisplayAmount(15000)).toEqual({
      compact: '15K',
      chinese: '1.5万',
      display: '15K（1.5万）',
      full: '15,000',
    })
  })

  it('formats millions billions and trillions', () => {
    expect(formatDualDisplayAmount(1_250_000)).toEqual({
      compact: '1.25M',
      chinese: '125万',
      display: '1.25M（125万）',
      full: '1,250,000',
    })
    expect(formatDualDisplayAmount(585_217_700)).toEqual({
      compact: '585.22M',
      chinese: '5.85亿',
      display: '585.22M（5.85亿）',
      full: '585,217,700',
    })
    expect(formatDualDisplayAmount(15_800_000_000)).toEqual({
      compact: '15.8B',
      chinese: '158亿',
      display: '15.8B（158亿）',
      full: '15,800,000,000',
    })
    expect(formatDualDisplayAmount(1_200_000_000_000)).toEqual({
      compact: '1.2T',
      chinese: '1.2万亿',
      display: '1.2T（1.2万亿）',
      full: '1,200,000,000,000',
    })
  })
})
