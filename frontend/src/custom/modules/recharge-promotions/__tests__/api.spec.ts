import { beforeEach, describe, expect, it, vi } from 'vitest'

const { getMock, postMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
  postMock: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get: getMock,
    post: postMock,
  },
}))

import { rechargePromotionsAPI } from '../api'

describe('recharge promotion API', () => {
  beforeEach(() => vi.clearAllMocks())

  it('keeps the checkout preview inside the authenticated Overlay API', () => {
    rechargePromotionsAPI.preview(20, 24)

    expect(postMock).toHaveBeenCalledWith('/recharge-promotions/preview', {
      amount: 20,
      base_credit: 24,
    })
  })

  it('uses separate user and administrator order routes', () => {
    rechargePromotionsAPI.getUserOrder(18)
    rechargePromotionsAPI.getAdminOrder(18)
    rechargePromotionsAPI.getAdminOrders({ page: 2, page_size: 10, mode: 'tiered_percentage' })

    expect(getMock).toHaveBeenNthCalledWith(1, '/recharge-promotions/orders/18')
    expect(getMock).toHaveBeenNthCalledWith(2, '/custom/admin/recharge-promotions/orders/18')
    expect(getMock).toHaveBeenNthCalledWith(3, '/custom/admin/recharge-promotions/orders', {
      params: { page: 2, page_size: 10, mode: 'tiered_percentage' },
    })
  })
})
