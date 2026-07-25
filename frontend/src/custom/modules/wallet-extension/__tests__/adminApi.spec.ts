import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, post, put } = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
}))

vi.mock('@/api/client', () => ({ apiClient: { get, post, put } }))

import {
  batchDistributeTransfers,
  freezeAdminTransfer,
  getAdminTransferFeeStats,
  listAdminRedPackets,
  listAdminTransfers,
  revokeAdminTransfer,
} from '../api/admin'

describe('wallet extension administrator transfer API', () => {
  beforeEach(() => {
    get.mockReset().mockResolvedValue({ data: {} })
    post.mockReset().mockResolvedValue({ data: {} })
    put.mockReset().mockResolvedValue({})
  })

  it('keeps transfer administration requests in the wallet extension', async () => {
    const transferParams = { page: 2, page_size: 25, status: 'completed' }
    const feeParams = { start_time: '2026-07-01', end_time: '2026-07-31' }
    const redPacketParams = { page: 3, page_size: 10 }

    await listAdminTransfers(transferParams)
    await getAdminTransferFeeStats(feeParams)
    await listAdminRedPackets(redPacketParams)
    await freezeAdminTransfer(12)
    await revokeAdminTransfer(13, 'manual review')
    await batchDistributeTransfers([{ user_id: 7, amount: 12.5 }], 'bonus')

    expect(get).toHaveBeenNthCalledWith(1, '/admin/transfers', { params: transferParams })
    expect(get).toHaveBeenNthCalledWith(2, '/admin/transfers/stats', { params: feeParams })
    expect(get).toHaveBeenNthCalledWith(3, '/admin/redpackets', { params: redPacketParams })
    expect(put).toHaveBeenNthCalledWith(1, '/admin/transfers/12/freeze')
    expect(put).toHaveBeenNthCalledWith(2, '/admin/transfers/13/revoke', { reason: 'manual review' })
    expect(post).toHaveBeenCalledWith('/admin/transfers/batch', {
      targets: [{ user_id: 7, amount: 12.5 }],
      memo: 'bonus',
    })
  })
})
