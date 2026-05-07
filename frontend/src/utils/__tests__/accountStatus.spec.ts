import { describe, expect, it } from 'vitest'

import {
  getAccountMainStatusState,
  getAccountRuntimeStatusState,
  getAccountSchedulingStatusState,
  getTempUnschedRuntimeCode,
  parseTempUnschedReason,
} from '../accountStatus'

describe('accountStatus utils', () => {
  it('parses structured temp-unsched reason', () => {
    const result = parseTempUnschedReason('{"status_code":401,"matched_keyword":"oauth_401","error_message":"OAuth 401: unauthorized","rule_index":-1}')

    expect(result.statusCode).toBe(401)
    expect(result.matchedKeyword).toBe('oauth_401')
    expect(result.errorMessage).toBe('OAuth 401: unauthorized')
    expect(result.ruleIndex).toBe(-1)
  })

  it('infers status code from legacy plain-text reason', () => {
    const result = parseTempUnschedReason('OAuth 401: invalid or expired credentials')

    expect(result.statusCode).toBe(401)
    expect(result.matchedKeyword).toBe('oauth_401')
    expect(result.errorMessage).toBe('OAuth 401: invalid or expired credentials')
    expect(result.ruleIndex).toBe(-1)
  })

  it('maps parsed 401 temp-unsched reasons to the dedicated runtime code', () => {
    const code = getTempUnschedRuntimeCode({
      statusCode: 401,
      matchedKeyword: 'oauth_401',
      errorMessage: 'OAuth 401: invalid or expired credentials',
      ruleIndex: -1,
    })

    expect(code).toBe('runtime_oauth401_cooldown')
  })

  it('distinguishes main state, scheduling switch, and runtime cooldown', () => {
    const main = getAccountMainStatusState({ status: 'active' })
    const scheduling = getAccountSchedulingStatusState({
      schedulable: true,
      auto_pause_on_expired: false,
      expires_at: null,
    })
    const runtime = getAccountRuntimeStatusState({
      rate_limit_reset_at: null,
      overload_until: null,
      temp_unschedulable_until: '2099-03-15T00:00:00Z',
      temp_unschedulable_reason: 'OAuth 401: invalid or expired credentials',
      quota_limit: null,
      quota_used: null,
      quota_daily_limit: null,
      quota_daily_used: null,
      quota_weekly_limit: null,
      quota_weekly_used: null,
    })

    expect(main.code).toBe('main_active')
    expect(scheduling.code).toBe('schedule_enabled')
    expect(runtime.code).toBe('runtime_oauth401_cooldown')
    expect(runtime.clickable).toBe(true)
  })

  it('marks expired auto-paused accounts as a scheduling-layer pause', () => {
    const scheduling = getAccountSchedulingStatusState({
      schedulable: false,
      auto_pause_on_expired: true,
      expires_at: Math.floor(Date.now() / 1000) - 60,
    })

    expect(scheduling.code).toBe('schedule_expired_paused')
  })
})
