import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, post } = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }))

vi.mock('@/api/client', () => ({ apiClient: { get, post } }))

import {
  getDirectTransferHistory,
  getDirectTransferStats,
  resolveDirectTransferReceiver,
  searchDirectTransferReceivers,
  submitDirectTransfer,
  validateDirectTransfer,
} from '../api'
import {
  transferBalance as legacyTransferBalance,
  validateTransfer as legacyValidateTransfer,
} from '@/api/transfer'

describe('wallet extension direct-transfer API', () => {
  beforeEach(() => {
    get.mockReset().mockResolvedValue({ data: {} })
    post.mockReset().mockResolvedValue({ data: {} })
  })

  it('uses only direct-transfer endpoints', async () => {
    await searchDirectTransferReceivers('alice')
    await resolveDirectTransferReceiver('alice@example.com')
    await getDirectTransferHistory({ role: 'sender', page: 2, page_size: 25 })
    await getDirectTransferStats()

    expect(get).toHaveBeenNthCalledWith(1, '/transfer/receivers', {
      params: expect.objectContaining({ query: 'alice' }),
    })
    expect(get).toHaveBeenNthCalledWith(2, '/transfer/receiver', {
      params: { query: 'alice@example.com' },
    })
    expect(get).toHaveBeenNthCalledWith(3, '/transfer/history', {
      params: { role: 'sender', page: 2, page_size: 25 },
    })
    expect(get).toHaveBeenNthCalledWith(4, '/transfer/stats')
  })

  it('preserves supplied idempotency keys for direct-transfer writes', async () => {
    await submitDirectTransfer(7, 12.5, 'for lunch', 'transfer-stable-key')
    await validateDirectTransfer(7, 12.5, 'validation-stable-key')

    expect(post).toHaveBeenNthCalledWith(
      1,
      '/transfer',
      { receiver_id: 7, amount: 12.5, memo: 'for lunch' },
      { headers: { 'Idempotency-Key': 'transfer-stable-key' } },
    )
    expect(post).toHaveBeenNthCalledWith(
      2,
      '/transfer/validate',
      { receiver_id: 7, amount: 12.5 },
      { headers: { 'Idempotency-Key': 'validation-stable-key' } },
    )
  })

  it('adds an idempotency key when a direct-transfer write starts', async () => {
    await submitDirectTransfer(7, 12.5)
    await validateDirectTransfer(7, 12.5)

    expect(post.mock.calls.map((call) => call[2].headers['Idempotency-Key'])).toEqual([
      expect.stringMatching(/^balance-transfer-.+/),
      expect.stringMatching(/^transfer-validate-.+/),
    ])
  })

  it('keeps the legacy direct-transfer exports as compatibility aliases', () => {
    expect(legacyTransferBalance).toBe(submitDirectTransfer)
    expect(legacyValidateTransfer).toBe(validateDirectTransfer)
  })
})
