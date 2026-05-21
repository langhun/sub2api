import { describe, expect, it } from 'vitest'
import { formatSignedCurrency } from '../format'

describe('formatSignedCurrency', () => {
  it('places one sign before the currency symbol', () => {
    expect(formatSignedCurrency(1.25)).toBe('+$1.25')
    expect(formatSignedCurrency(-1.25)).toBe('-$1.25')
    expect(formatSignedCurrency(0)).toBe('$0.00')
  })

  it('handles nullish values as zero', () => {
    expect(formatSignedCurrency(null)).toBe('$0.00')
    expect(formatSignedCurrency(undefined)).toBe('$0.00')
  })
})
