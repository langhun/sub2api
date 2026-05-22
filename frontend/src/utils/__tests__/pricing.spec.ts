import { describe, expect, it } from 'vitest'
import { formatScaled } from '../pricing'

describe('formatScaled', () => {
  it('returns dash for null values', () => {
    expect(formatScaled(null, 1_000_000)).toBe('-')
  })

  it('formats ordinary unscaled values', () => {
    expect(formatScaled(0.5, 1)).toBe('$0.5')
  })

  it('formats per-million scaled values', () => {
    expect(formatScaled(0.000003, 1_000_000)).toBe('$3')
  })

  it('trims trailing zeros after scaling', () => {
    expect(formatScaled(0.00000123, 1_000_000)).toBe('$1.23')
    expect(formatScaled(1.2, 1)).toBe('$1.2')
  })

  it('avoids obvious floating point display noise', () => {
    expect(formatScaled(0.1 + 0.2, 1)).toBe('$0.3')
    expect(formatScaled(0.0000001 + 0.0000002, 1_000_000)).toBe('$0.3')
  })
})
