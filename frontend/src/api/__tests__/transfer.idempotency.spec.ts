import { beforeEach, describe, expect, it, vi } from 'vitest'

const { post } = vi.hoisted(() => ({ post: vi.fn() }))
vi.mock('@/api/client', () => ({ apiClient: { post, get: vi.fn() } }))

import { claimRedPacket, createRedPacket, transferBalance, validateTransfer } from '@/api/transfer'

describe('activity write API idempotency', () => {
  beforeEach(() => post.mockReset().mockResolvedValue({ data: {} }))

  it.each([
    ['transfer', () => transferBalance(2, 10)],
    ['validation', () => validateTransfer(2, 10)],
    ['red packet create', () => createRedPacket({ total_amount: 10, count: 2 })],
    ['red packet claim', () => claimRedPacket('RP-CODE')],
  ])('adds an Idempotency-Key to %s POST', async (_name, request) => {
    await request()
    const config = post.mock.calls[0][2]
    expect(config.headers['Idempotency-Key']).toMatch(/^[a-z-]+-.+/)
  })
})
