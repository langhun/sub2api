import { ref } from 'vue'
import { adminAPI } from '@/api/admin'
import type { Proxy, ProxyQualityCheckResult } from '@/types'

interface ProxyTestResult {
  success: boolean
  latency_ms?: number
  message?: string
  ip_address?: string
  country?: string
  country_code?: string
  region?: string
  city?: string
}

interface QualityCheckSummary {
  total: number
  healthy: number
  warn: number
  challenge: number
  failed: number
}

export function useProxyTesting() {
  const testingProxyIds = ref<Set<number>>(new Set())
  const qualityCheckingProxyIds = ref<Set<number>>(new Set())

  const startTestingProxy = (proxyId: number) => {
    testingProxyIds.value = new Set([...testingProxyIds.value, proxyId])
  }

  const stopTestingProxy = (proxyId: number) => {
    const next = new Set(testingProxyIds.value)
    next.delete(proxyId)
    testingProxyIds.value = next
  }

  const startQualityCheckingProxy = (proxyId: number) => {
    qualityCheckingProxyIds.value = new Set([...qualityCheckingProxyIds.value, proxyId])
  }

  const stopQualityCheckingProxy = (proxyId: number) => {
    const next = new Set(qualityCheckingProxyIds.value)
    next.delete(proxyId)
    qualityCheckingProxyIds.value = next
  }

  const testSingleProxy = async (
    proxyId: number
  ): Promise<ProxyTestResult | null> => {
    startTestingProxy(proxyId)
    try {
      const result = await adminAPI.proxies.testProxy(proxyId)
      return result
    } catch (error: any) {
      const message = error.response?.data?.detail || 'Test failed'
      return { success: false, message }
    } finally {
      stopTestingProxy(proxyId)
    }
  }

  const testMultipleProxies = async (
    ids: number[],
    concurrency: number = 5
  ): Promise<void> => {
    if (ids.length === 0) return

    let index = 0
    const worker = async () => {
      while (index < ids.length) {
        const current = ids[index]
        index++
        await testSingleProxy(current)
      }
    }

    const workers = Array.from(
      { length: Math.min(concurrency, ids.length) },
      () => worker()
    )
    await Promise.all(workers)
  }

  const checkSingleProxyQuality = async (
    proxyId: number
  ): Promise<ProxyQualityCheckResult | null> => {
    startQualityCheckingProxy(proxyId)
    try {
      const result = await adminAPI.proxies.checkProxyQuality(proxyId)
      return result
    } catch (error) {
      return null
    } finally {
      stopQualityCheckingProxy(proxyId)
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

  const checkMultipleProxiesQuality = async (
    ids: number[],
    concurrency: number = 3
  ): Promise<QualityCheckSummary> => {
    if (ids.length === 0) {
      return { total: 0, healthy: 0, warn: 0, challenge: 0, failed: 0 }
    }

    let index = 0
    let healthy = 0
    let warn = 0
    let challenge = 0
    let failed = 0

    const worker = async () => {
      while (index < ids.length) {
        const current = ids[index]
        index++
        const result = await checkSingleProxyQuality(current)

        if (!result) {
          failed++
          continue
        }

        const status = summarizeQualityStatus(result)
        if (status === 'challenge') {
          challenge++
        } else if (status === 'failed') {
          failed++
        } else if (status === 'warn') {
          warn++
        } else {
          healthy++
        }
      }
    }

    const workers = Array.from(
      { length: Math.min(concurrency, ids.length) },
      () => worker()
    )
    await Promise.all(workers)

    return {
      total: ids.length,
      healthy,
      warn,
      challenge,
      failed
    }
  }

  return {
    testingProxyIds,
    qualityCheckingProxyIds,
    testSingleProxy,
    testMultipleProxies,
    checkSingleProxyQuality,
    checkMultipleProxiesQuality,
    summarizeQualityStatus
  }
}

