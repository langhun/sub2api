/**
 * @fileoverview 代理测试 Composable
 *
 * 提供代理连接测试和质量检查的核心逻辑，支持单个和批量操作，带并发控制。
 *
 * @example
 * ```typescript
 * const {
 *   testingProxyIds,
 *   testSingleProxy,
 *   testMultipleProxies,
 *   checkMultipleProxiesQuality
 * } = useProxyTesting()
 *
 * // 单个测试
 * const result = await testSingleProxy(proxyId)
 *
 * // 批量测试（并发控制）
 * await testMultipleProxies(selectedIds.value, 5)
 *
 * // 质量检查
 * const summary = await checkMultipleProxiesQuality(selectedIds.value, 3)
 * ```
 */

import { ref } from 'vue'
import { adminAPI } from '@/api/admin'
import type { Proxy, ProxyQualityCheckResult } from '@/types'

/**
 * 代理测试结果
 */
interface ProxyTestResult {
  /** 测试是否成功 */
  success: boolean
  /** 延迟（毫秒） */
  latency_ms?: number
  /** 错误消息 */
  message?: string
  /** IP 地址 */
  ip_address?: string
  /** 国家 */
  country?: string
  /** 国家代码 */
  country_code?: string
  /** 地区 */
  region?: string
  /** 城市 */
  city?: string
}

/**
 * 质量检查汇总结果
 */
interface QualityCheckSummary {
  /** 总数 */
  total: number
  /** 健康数量 */
  healthy: number
  /** 警告数量 */
  warn: number
  /** 需要验证数量 */
  challenge: number
  /** 失败数量 */
  failed: number
}

/**
 * 代理测试 Composable
 *
 * 提供代理测试和质量检查功能，支持：
 * - 单个和批量测试
 * - 并发控制
 * - 测试状态跟踪
 * - 质量状态汇总
 *
 * @returns 包含测试状态和操作方法的对象
 */
export function useProxyTesting() {
  const testingProxyIds = ref<Set<number>>(new Set())
  const qualityCheckingProxyIds = ref<Set<number>>(new Set())

  /** 开始测试代理（添加到测试中集合） */
  const startTestingProxy = (proxyId: number) => {
    testingProxyIds.value = new Set([...testingProxyIds.value, proxyId])
  }

  /** 停止测试代理（从测试中集合移除） */
  const stopTestingProxy = (proxyId: number) => {
    const next = new Set(testingProxyIds.value)
    next.delete(proxyId)
    testingProxyIds.value = next
  }

  /** 开始质量检查代理（添加到质量检查中集合） */
  const startQualityCheckingProxy = (proxyId: number) => {
    qualityCheckingProxyIds.value = new Set([...qualityCheckingProxyIds.value, proxyId])
  }

  /** 停止质量检查代理（从质量检查中集合移除） */
  const stopQualityCheckingProxy = (proxyId: number) => {
    const next = new Set(qualityCheckingProxyIds.value)
    next.delete(proxyId)
    qualityCheckingProxyIds.value = next
  }

  /**
   * 测试单个代理
   *
   * @param proxyId 代理 ID
   * @returns 测试结果，失败时返回包含错误信息的对象
   */
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

  /**
   * 批量测试多个代理
   *
   * @param ids 代理 ID 列表
   * @param concurrency 并发数，默认为 5
   */
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

  /**
   * 检查单个代理质量
   *
   * @param proxyId 代理 ID
   * @returns 质量检查结果，失败时返回 null
   */
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

  /**
   * 汇总质量检查结果为状态
   *
   * @param result 质量检查结果
   * @returns 质量状态：'healthy' | 'warn' | 'challenge' | 'failed'
   */
  const summarizeQualityStatus = (
    result: ProxyQualityCheckResult
  ): Proxy['quality_status'] => {
    if (result.challenge_count > 0) return 'challenge'
    if (result.failed_count > 0) return 'failed'
    if (result.warn_count > 0) return 'warn'
    return 'healthy'
  }

  /**
   * 批量检查多个代理质量
   *
   * @param ids 代理 ID 列表
   * @param concurrency 并发数，默认为 3
   * @returns 质量检查汇总结果
   */
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

