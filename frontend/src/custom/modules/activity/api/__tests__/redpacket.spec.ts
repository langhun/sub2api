import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, post } = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }))
vi.mock('@/api/client', () => ({ apiClient: { post, get } }))

import {
  claimRedPacket as legacyClaimRedPacket,
  createRedPacket as legacyCreateRedPacket,
  getMyRedPackets as legacyGetMyRedPackets,
  getRedPacketDetail as legacyGetRedPacketDetail,
} from '@/api/transfer'
import legacyTransferAPI from '@/api/transfer'
import {
  claimRedPacket,
  createRedPacket,
  getMyRedPackets,
  getRedPacketDetail,
} from '../redpacket'

describe('activity red packet API', () => {
  beforeEach(() => {
    get.mockReset().mockResolvedValue({ data: {} })
    post.mockReset().mockResolvedValue({ data: {} })
  })

  it('keeps the transfer API exports as direct compatibility aliases', () => {
    expect(legacyCreateRedPacket).toBe(createRedPacket)
    expect(legacyClaimRedPacket).toBe(claimRedPacket)
    expect(legacyGetMyRedPackets).toBe(getMyRedPackets)
    expect(legacyGetRedPacketDetail).toBe(getRedPacketDetail)
    expect(legacyTransferAPI.createRedPacket).toBe(createRedPacket)
    expect(legacyTransferAPI.claimRedPacket).toBe(claimRedPacket)
    expect(legacyTransferAPI.getMyRedPackets).toBe(getMyRedPackets)
    expect(legacyTransferAPI.getRedPacketDetail).toBe(getRedPacketDetail)
  })

  it.each([
    ['create', () => createRedPacket({ total_amount: 10, count: 2 })],
    ['claim', () => claimRedPacket('RP-CODE')],
  ])('adds an Idempotency-Key to red packet %s requests', async (_name, request) => {
    await request()
    expect(post.mock.calls[0][2].headers['Idempotency-Key']).toMatch(/^redpacket-[a-z-]+-.+/)
  })

  it('uses caller-supplied keys for retried writes', async () => {
    await createRedPacket({ total_amount: 10, count: 2 }, 'redpacket-create-stable')
    await claimRedPacket('RP-CODE', 'redpacket-claim-stable')

    expect(post.mock.calls.map((call) => call[2].headers['Idempotency-Key'])).toEqual([
      'redpacket-create-stable',
      'redpacket-claim-stable',
    ])
  })

  it('preserves the red packet request paths', async () => {
    await getMyRedPackets({ role: 'sent' })
    await getRedPacketDetail(7)

    expect(get.mock.calls).toEqual([
      ['/redpacket/my', { params: { role: 'sent' } }],
      ['/redpacket/7'],
    ])
  })
})
