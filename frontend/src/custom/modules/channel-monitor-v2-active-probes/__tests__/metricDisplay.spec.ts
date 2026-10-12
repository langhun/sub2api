import { describe, expect, it } from 'vitest'
import { hasCacheSample, latencyMetricForDisplay } from '../metricDisplay'

describe('channel monitor active probe metric display', () => {
  it('uses real TTFT before duration', () => {
    expect(latencyMetricForDisplay({ ttft: { p50_ms: 120 }, duration: { p50_ms: 900 } })).toEqual({
      kind: 'ttft',
      metric: { p50_ms: 120 },
    })
  })

  it('uses response duration when a synchronous probe has no TTFT', () => {
    expect(latencyMetricForDisplay({ ttft: { p50_ms: null }, duration: { p50_ms: 900 } })).toEqual({
      kind: 'duration',
      metric: { p50_ms: 900 },
    })
  })

  it('does not turn missing cache samples into a zero rate', () => {
    expect(hasCacheSample({ cache_rate: 0, cache_rate_denominator: 0 })).toBe(false)
    expect(hasCacheSample({ cache_rate: 0.5, cache_rate_denominator: 2 })).toBe(true)
  })
})
