import { describe, it, expect, beforeEach } from 'vitest'
import { ref } from 'vue'
import { useProxyResultHandler } from '../useProxyResultHandler'
import type { Proxy, ProxyQualityCheckResult } from '@/types'

describe('useProxyResultHandler', () => {
  let proxies: { value: Proxy[] }
  let handler: ReturnType<typeof useProxyResultHandler>

  const createMockProxy = (id: number, overrides: Partial<Proxy> = {}): Proxy => ({
    id,
    name: `Proxy ${id}`,
    protocol: 'http',
    host: 'localhost',
    port: 8080,
    username: null,
    status: 'active',
    created_at: '2024-01-01',
    updated_at: '2024-01-01',
    ...overrides
  })

  beforeEach(() => {
    proxies = ref([
      createMockProxy(1),
      createMockProxy(2),
      createMockProxy(3)
    ])
    handler = useProxyResultHandler(proxies)
  })

  describe('applyLatencyResult', () => {
    it('should apply successful latency result', () => {
      handler.applyLatencyResult(1, {
        success: true,
        latency_ms: 100,
        message: 'OK',
        ip_address: '1.2.3.4',
        country: 'US',
        country_code: 'US',
        region: 'California',
        city: 'San Francisco'
      })

      const proxy = proxies.value[0]
      expect(proxy.latency_status).toBe('success')
      expect(proxy.latency_ms).toBe(100)
      expect(proxy.ip_address).toBe('1.2.3.4')
      expect(proxy.country).toBe('US')
      expect(proxy.country_code).toBe('US')
      expect(proxy.region).toBe('California')
      expect(proxy.city).toBe('San Francisco')
      expect(proxy.health_status).toBe('healthy')
      expect(proxy.cooldown_until_unix).toBeUndefined()
      expect(proxy.latency_message).toBe('OK')
    })

    it('should apply failed latency result', () => {
      const beforeTime = Math.floor(Date.now() / 1000)

      handler.applyLatencyResult(1, {
        success: false,
        message: 'Connection timeout'
      })

      const proxy = proxies.value[0]
      expect(proxy.latency_status).toBe('failed')
      expect(proxy.latency_ms).toBeUndefined()
      expect(proxy.ip_address).toBeUndefined()
      expect(proxy.country).toBeUndefined()
      expect(proxy.country_code).toBeUndefined()
      expect(proxy.region).toBeUndefined()
      expect(proxy.city).toBeUndefined()
      expect(proxy.health_status).toBe('failed')
      expect(proxy.last_fail_reason).toBe('Connection timeout')
      expect(proxy.last_fail_at_unix).toBeGreaterThanOrEqual(beforeTime)
      expect(proxy.latency_message).toBe('Connection timeout')
    })

    it('should do nothing when proxy not found', () => {
      const originalProxies = [...proxies.value]

      handler.applyLatencyResult(999, {
        success: true,
        latency_ms: 100
      })

      expect(proxies.value).toEqual(originalProxies)
    })
  })

  describe('summarizeQualityStatus', () => {
    it('should return challenge when challenge_count > 0', () => {
      const result: ProxyQualityCheckResult = {
        proxy_id: 1,
        score: 50,
        grade: 'C',
        summary: 'Challenge detected',
        passed_count: 2,
        warn_count: 1,
        failed_count: 0,
        challenge_count: 1,
        checked_at: Date.now(),
        items: []
      }
      expect(handler.summarizeQualityStatus(result)).toBe('challenge')
    })

    it('should return failed when failed_count > 0 and no challenges', () => {
      const result: ProxyQualityCheckResult = {
        proxy_id: 1,
        score: 30,
        grade: 'F',
        summary: 'Failed',
        passed_count: 1,
        warn_count: 0,
        failed_count: 2,
        challenge_count: 0,
        checked_at: Date.now(),
        items: []
      }
      expect(handler.summarizeQualityStatus(result)).toBe('failed')
    })

    it('should return warn when warn_count > 0 and no failures or challenges', () => {
      const result: ProxyQualityCheckResult = {
        proxy_id: 1,
        score: 70,
        grade: 'B',
        summary: 'Warning',
        passed_count: 3,
        warn_count: 1,
        failed_count: 0,
        challenge_count: 0,
        checked_at: Date.now(),
        items: []
      }
      expect(handler.summarizeQualityStatus(result)).toBe('warn')
    })

    it('should return healthy when all checks pass', () => {
      const result: ProxyQualityCheckResult = {
        proxy_id: 1,
        score: 100,
        grade: 'A',
        summary: 'All good',
        passed_count: 5,
        warn_count: 0,
        failed_count: 0,
        challenge_count: 0,
        checked_at: Date.now(),
        items: []
      }
      expect(handler.summarizeQualityStatus(result)).toBe('healthy')
    })
  })

  describe('applyQualityResult', () => {
    it('should apply quality check result to proxy', () => {
      const result: ProxyQualityCheckResult = {
        proxy_id: 1,
        score: 85,
        grade: 'A',
        summary: 'Excellent',
        passed_count: 5,
        warn_count: 0,
        failed_count: 0,
        challenge_count: 0,
        checked_at: 1234567890,
        items: []
      }

      handler.applyQualityResult(1, result)

      const proxy = proxies.value[0]
      expect(proxy.quality_status).toBe('healthy')
      expect(proxy.quality_score).toBe(85)
      expect(proxy.quality_grade).toBe('A')
      expect(proxy.quality_summary).toBe('Excellent')
      expect(proxy.quality_checked).toBe(1234567890)
    })

    it('should do nothing when proxy not found', () => {
      const result: ProxyQualityCheckResult = {
        proxy_id: 999,
        score: 85,
        grade: 'A',
        summary: 'Excellent',
        passed_count: 5,
        warn_count: 0,
        failed_count: 0,
        challenge_count: 0,
        checked_at: 1234567890,
        items: []
      }

      const originalProxies = [...proxies.value]
      handler.applyQualityResult(999, result)
      expect(proxies.value).toEqual(originalProxies)
    })
  })

  describe('extractBaseConnectivityResult', () => {
    it('should extract base connectivity result when pass', () => {
      const result: ProxyQualityCheckResult = {
        proxy_id: 1,
        score: 85,
        grade: 'A',
        summary: 'Good connection',
        base_latency_ms: 120,
        exit_ip: '5.6.7.8',
        country: 'UK',
        country_code: 'GB',
        passed_count: 5,
        warn_count: 0,
        failed_count: 0,
        challenge_count: 0,
        checked_at: Date.now(),
        items: [
          {
            target: 'base_connectivity',
            status: 'pass',
            latency_ms: 120
          }
        ]
      }

      const latencyResult = handler.extractBaseConnectivityResult(result)

      expect(latencyResult).toEqual({
        success: true,
        latency_ms: 120,
        message: 'Good connection',
        ip_address: '5.6.7.8',
        country: 'UK',
        country_code: 'GB'
      })
    })

    it('should return null when base_connectivity not found', () => {
      const result: ProxyQualityCheckResult = {
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

      expect(handler.extractBaseConnectivityResult(result)).toBeNull()
    })

    it('should return null when base_connectivity status is not pass', () => {
      const result: ProxyQualityCheckResult = {
        proxy_id: 1,
        score: 50,
        grade: 'C',
        summary: 'Failed',
        passed_count: 3,
        warn_count: 0,
        failed_count: 1,
        challenge_count: 0,
        checked_at: Date.now(),
        items: [
          {
            target: 'base_connectivity',
            status: 'fail',
            message: 'Connection failed'
          }
        ]
      }

      expect(handler.extractBaseConnectivityResult(result)).toBeNull()
    })
  })
})



