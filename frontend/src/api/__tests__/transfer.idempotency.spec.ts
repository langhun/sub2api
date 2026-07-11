import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, post } = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }))
vi.mock('@/api/client', () => ({ apiClient: { post, get } }))

import {
  claimRedPacket,
  createRedPacket,
  getMyRedPackets,
  getRedPacketDetail,
  getTransferHistory,
  getTransferLeaderboard,
  getTransferStats,
  resolveTransferReceiver,
  transferBalance,
  validateTransfer,
} from '@/api/transfer'

describe('activity write API idempotency', () => {
  beforeEach(() => {
    get.mockReset().mockResolvedValue({ data: {} })
    post.mockReset().mockResolvedValue({ data: {} })
  })

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

  it('uses the operation key supplied by the UI for retried writes', async () => {
    await transferBalance(2, 10, 'memo', 'balance-transfer-stable')
    await createRedPacket({ total_amount: 10, count: 2 }, 'redpacket-create-stable')
    await claimRedPacket('RP-CODE', 'redpacket-claim-stable')

    expect(post.mock.calls.map((call) => call[2].headers['Idempotency-Key'])).toEqual([
      'balance-transfer-stable',
      'redpacket-create-stable',
      'redpacket-claim-stable',
    ])
  })

  it('uses the user-scoped receiver resolver endpoint', async () => {
    get.mockResolvedValueOnce({ data: { receiver_id: 2, receiver_display: 'a***e' } })

    await resolveTransferReceiver('alice@example.com')

    expect(get).toHaveBeenCalledWith('/transfer/receiver', {
      params: { query: 'alice@example.com' },
    })
  })

  it('matches the backend transfer and red packet route groups', async () => {
    get.mockResolvedValue({ data: {} })

    await getTransferHistory({ page: 1 })
    await getTransferStats()
    await getTransferLeaderboard({ limit: 10 })
    await getMyRedPackets({ role: 'sent' })
    await getRedPacketDetail(7)

    expect(get.mock.calls.map((call) => call[0])).toEqual([
      '/transfer/history',
      '/transfer/stats',
      '/transfer/leaderboard',
      '/redpacket/my',
      '/redpacket/7',
    ])
  })
})
