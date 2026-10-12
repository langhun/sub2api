export interface MonitorLatencyValue {
  sample_count?: number
  p50_ms?: number | null
  p90_ms?: number | null
  p95_ms?: number | null
  avg_ms?: number | null
}

export interface MonitorMetricForDisplay {
  ttft?: MonitorLatencyValue | null
  duration?: MonitorLatencyValue | null
  cache_rate?: number | null
  cache_rate_denominator?: number | null
}

export type MonitorLatencyKind = 'ttft' | 'duration' | 'unavailable'

/**
 * Active probes are synchronous and therefore only provide total response
 * duration. Prefer a real user-stream TTFT sample whenever one exists; fall
 * back to the measured response duration without calling it TTFT.
 */
export function latencyMetricForDisplay(metric: MonitorMetricForDisplay): {
  kind: MonitorLatencyKind
  metric: MonitorLatencyValue
} {
  if (metric.ttft?.p50_ms != null) {
    return { kind: 'ttft', metric: metric.ttft }
  }
  if (metric.duration?.p50_ms != null) {
    return { kind: 'duration', metric: metric.duration }
  }
  return { kind: 'unavailable', metric: metric.ttft || metric.duration || {} }
}

export function hasCacheSample(metric: MonitorMetricForDisplay): boolean {
  return Number(metric.cache_rate_denominator || 0) > 0
}
