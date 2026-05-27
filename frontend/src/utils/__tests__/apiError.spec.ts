import { describe, expect, it } from 'vitest'
import { buildPrefixedApiErrorMessage, extractApiErrorMessage } from '@/utils/apiError'

describe('extractApiErrorMessage', () => {
  it('prefers interceptor message from plain API error objects', () => {
    const message = extractApiErrorMessage(
      {
        status: 502,
        message: 'Upstream model list request failed with HTTP 502',
        error: 'ignored fallback'
      },
      'fallback'
    )

    expect(message).toBe('Upstream model list request failed with HTTP 502')
  })
})

describe('buildPrefixedApiErrorMessage', () => {
  it('returns only fallback when extracted message equals fallback', () => {
    const message = buildPrefixedApiErrorMessage(
      {
        message: '同步上游模型失败'
      },
      '同步上游模型失败'
    )

    expect(message).toBe('同步上游模型失败')
  })

  it('appends detailed upstream message when available', () => {
    const message = buildPrefixedApiErrorMessage(
      {
        message: 'Upstream model list request failed with HTTP 502'
      },
      '同步上游模型失败'
    )

    expect(message).toBe('同步上游模型失败：Upstream model list request failed with HTTP 502')
  })
})
