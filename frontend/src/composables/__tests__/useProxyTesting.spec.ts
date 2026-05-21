import { describe, it, expect, beforeEach, vi } from 'vitest'
import { useProxyTesting } from '../useProxyTesting'
import { adminAPI } from '@/api/admin'

vi.mock('@/api/admin', () => ({
  adminAPI: {
    proxies: {
      testProxy: vi.fn(),
      checkProxyQuality: vi.fn()
    }
  }
}))

describe('useProxyTesting', () => {
  let proxyTesting: ReturnType<typeof useProxyTesting>

  beforeEach(() => {
    proxyTesting = useProxyTesting()
    vi.clearAllMocks()
  })

  describe('testSingleProxy', () => {
    it('should test proxy successfully', async () => {
      const mockResult = {
        success: true,
        latency_ms: 100,
        message: 'OK'
      }
      vi.mocked(adminAPI.proxies.testProxy).mockResolvedValue(mockResult)

      const result = await proxyTesting.testSingleProxy(1)

      expect(result).toEqual(mockResult)
      expect(adminAPI.proxies.testProxy).toHaveBeenCalledWith(1)
      expect(proxyTesting.testingProxyIds.value.has(1)).toBe(false)
    })

    it('should handle test failure', async () => {
      const error = {
        response: {
          data: {
            detail: 'Connection timeout'
          }
        }
      }
      vi.mocked(adminAPI.proxies.testProxy).mockRejectedValue(error)

      const result = await proxyTesting.testSingleProxy(1)

      expect(result).toEqual({
        success: false,
        message: 'Connection timeout'
      })
      expect(proxyTesting.testingProxyIds.value.has(1)).toBe(false)
    })

    it('should handle test error without detail', async () => {
      vi.mocked(adminAPI.proxies.testProxy).mockRejectedValue(new Error('Network error'))

      const result = await proxyTesting.testSingleProxy(1)

      expect(result).toEqual({
        success: false,
        message: 'Test failed'
      })
    })
  })

  describe('testMultipleProxies', () => {
    it('should test multiple proxies with concurrency', async () => {
      vi.mocked(adminAPI.proxies.testProxy).mockResolvedValue({
        success: true,
        latency_ms: 100
      })

      await proxyTesting.testMultipleProxies([1, 2, 3], 2)

      expect(adminAPI.proxies.testProxy).toHaveBeenCalledTimes(3)
      expect(adminAPI.proxies.testProxy).toHaveBeenCalledWith(1)
      expect(adminAPI.proxies.testProxy).toHaveBeenCalledWith(2)
      expect(adminAPI.proxies.testProxy).toHaveBeenCalledWith(3)
    })

    it('should handle empty array', async () => {
      await proxyTesting.testMultipleProxies([])

      expect(adminAPI.proxies.testProxy).not.toHaveBeenCalled()
    })

    it('should respect concurrency limit', async () => {
      let activeTests = 0
      let maxActiveTests = 0

      vi.mocked(adminAPI.proxies.testProxy).mockImplementation(async () => {
        activeTests++
        maxActiveTests = Math.max(maxActiveTests, activeTests)
        await new Promise(resolve => setTimeout(resolve, 10))
        activeTests--
        return { success: true, latency_ms: 100 }
      })

      await proxyTesting.testMultipleProxies([1, 2, 3, 4, 5], 2)

      expect(maxActiveTests).toBeLessThanOrEqual(2)
    })
  })

  describe('checkSingleProxyQuality', () => {
    it('should check proxy quality successfully', async () => {
      const mockResult = {
        proxy_id: 1,
        score: 85,
        grade: 'A',
        summary: 'Good',
        passed_count: 5,
        warn_count: 0,
        failed_count: 0,
        challenge_count: 0,
        checked_at: Date.now(),
        items: []
      }
      vi.mocked(adminAPI.proxies.checkProxyQuality).mockResolvedValue(mockResult)

      const result = await proxyTesting.checkSingleProxyQuality(1)

      expect(result).toEqual(mockResult)
      expect(adminAPI.proxies.checkProxyQuality).toHaveBeenCalledWith(1)
      expect(proxyTesting.qualityCheckingProxyIds.value.has(1)).toBe(false)
    })

    it('should handle quality check failure', async () => {
      vi.mocked(adminAPI.proxies.checkProxyQuality).mockRejectedValue(new Error('Failed'))

      const result = await proxyTesting.checkSingleProxyQuality(1)

      expect(result).toBeNull()
      expect(proxyTesting.qualityCheckingProxyIds.value.has(1)).toBe(false)
    })
  })

  describe('checkMultipleProxiesQuality', () => {
    it('should check multiple proxies and return summary', async () => {
      vi.mocked(adminAPI.proxies.checkProxyQuality)
        .mockResolvedValueOnce({
          proxy_id: 1,
          score: 100,
          grade: 'A',
          summary: 'Healthy',
          passed_count: 5,
          warn_count: 0,
          failed_count: 0,
          challenge_count: 0,
          checked_at: Date.now(),
          items: []
        })
        .mockResolvedValueOnce({
          proxy_id: 2,
          score: 70,
          grade: 'B',
          summary: 'Warning',
          passed_count: 4,
          warn_count: 1,
          failed_count: 0,
          challenge_count: 0,
          checked_at: Date.now(),
          items: []
        })
        .mockResolvedValueOnce({
          proxy_id: 3,
          score: 50,
          grade: 'C',
          summary: 'Challenge',
          passed_count: 3,
          warn_count: 0,
          failed_count: 0,
          challenge_count: 1,
          checked_at: Date.now(),
          items: []
        })

      const summary = await proxyTesting.checkMultipleProxiesQuality([1, 2, 3])

      expect(summary).toEqual({
        total: 3,
        healthy: 1,
        warn: 1,
        challenge: 1,
        failed: 0
      })
    })

    it('should handle empty array', async () => {
      const summary = await proxyTesting.checkMultipleProxiesQuality([])

      expect(summary).toEqual({
        total: 0,
        healthy: 0,
        warn: 0,
        challenge: 0,
        failed: 0
      })
    })

    it('should count failed checks', async () => {
      vi.mocked(adminAPI.proxies.checkProxyQuality).mockRejectedValue(new Error('Failed'))

      const summary = await proxyTesting.checkMultipleProxiesQuality([1, 2])

      expect(summary).toEqual({
        total: 2,
        healthy: 0,
        warn: 0,
        challenge: 0,
        failed: 2
      })
    })
  })

  describe('summarizeQualityStatus', () => {
    it('should prioritize challenge status', () => {
      const result = {
        proxy_id: 1,
        score: 50,
        grade: 'C',
        summary: 'Challenge',
        passed_count: 3,
        warn_count: 1,
        failed_count: 1,
        challenge_count: 1,
        checked_at: Date.now(),
        items: []
      }

      expect(proxyTesting.summarizeQualityStatus(result)).toBe('challenge')
    })

    it('should return failed when no challenges', () => {
      const result = {
        proxy_id: 1,
        score: 30,
        grade: 'F',
        summary: 'Failed',
        passed_count: 2,
        warn_count: 0,
        failed_count: 2,
        challenge_count: 0,
        checked_at: Date.now(),
        items: []
      }

      expect(proxyTesting.summarizeQualityStatus(result)).toBe('failed')
    })

    it('should return warn when no failures or challenges', () => {
      const result = {
        proxy_id: 1,
        score: 70,
        grade: 'B',
        summary: 'Warning',
        passed_count: 4,
        warn_count: 1,
        failed_count: 0,
        challenge_count: 0,
        checked_at: Date.now(),
        items: []
      }

      expect(proxyTesting.summarizeQualityStatus(result)).toBe('warn')
    })

    it('should return healthy when all pass', () => {
      const result = {
        proxy_id: 1,
        score: 100,
        grade: 'A',
        summary: 'Healthy',
        passed_count: 5,
        warn_count: 0,
        failed_count: 0,
        challenge_count: 0,
        checked_at: Date.now(),
        items: []
      }

      expect(proxyTesting.summarizeQualityStatus(result)).toBe('healthy')
    })
  })
})


