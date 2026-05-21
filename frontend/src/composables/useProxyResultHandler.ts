import type { Proxy, ProxyQualityCheckResult } from '@/types'

interface LatencyResult {
  success: boolean
  latency_ms?: number
  message?: string
  ip_address?: string
  country?: string
  country_code?: string
  region?: string
  city?: string
}

export function useProxyResultHandler(proxies: { value: Proxy[] }) {
  const findProxyById = (proxyId: number): Proxy | undefined => {
    return proxies.value.find((proxy) => proxy.id === proxyId)
  }

  const applySuccessfulLatencyResult = (
    target: Proxy,
    result: LatencyResult
  ) => {
    target.latency_status = 'success'
    target.latency_ms = result.latency_ms
    target.ip_address = result.ip_address
    target.country = result.country
    target.country_code = result.country_code
    target.region = result.region
    target.city = result.city
    target.health_status = 'healthy'
    target.cooldown_until_unix = undefined
    target.latency_message = result.message
  }

  const applyFailedLatencyResult = (target: Proxy, result: LatencyResult) => {
    target.latency_status = 'failed'
    target.latency_ms = undefined
    target.ip_address = undefined
    target.country = undefined
    target.country_code = undefined
    target.region = undefined
    target.city = undefined
    target.health_status = 'failed'
    target.last_fail_reason = result.message
    target.last_fail_at_unix = Math.floor(Date.now() / 1000)
    target.latency_message = result.message
  }

  const applyLatencyResult = (proxyId: number, result: LatencyResult) => {
    const target = findProxyById(proxyId)
    if (!target) return

    if (result.success) {
      applySuccessfulLatencyResult(target, result)
    } else {
      applyFailedLatencyResult(target, result)
    }
  }

  const summarizeQualityStatus = (
    result: ProxyQualityCheckResult
  ): Proxy['quality_status'] => {
    if (result.challenge_count > 0) return 'challenge'
    if (result.failed_count > 0) return 'failed'
    if (result.warn_count > 0) return 'warn'
    return 'healthy'
  }

  const applyQualityResult = (
    proxyId: number,
    result: ProxyQualityCheckResult
  ) => {
    const target = findProxyById(proxyId)
    if (!target) return

    target.quality_status = summarizeQualityStatus(result)
    target.quality_score = result.score
    target.quality_grade = result.grade
    target.quality_summary = result.summary
    target.quality_checked = result.checked_at
  }

  const extractBaseConnectivityResult = (
    result: ProxyQualityCheckResult
  ): LatencyResult | null => {
    const baseStep = result.items.find(
      (item) => item.target === 'base_connectivity'
    )

    if (!baseStep || baseStep.status !== 'pass') {
      return null
    }

    return {
      success: true,
      latency_ms: result.base_latency_ms,
      message: result.summary,
      ip_address: result.exit_ip,
      country: result.country,
      country_code: result.country_code
    }
  }

  return {
    applyLatencyResult,
    applyQualityResult,
    extractBaseConnectivityResult,
    summarizeQualityStatus
  }
}

