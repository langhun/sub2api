import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, post } = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: { get, post },
}))

import { listRegistrationRisk, lockRegistrationRisk, unlockRegistrationRisk } from '@/api/admin/registrationRisk'

describe('registration risk admin API', () => {
  beforeEach(() => {
    get.mockReset()
    post.mockReset()
  })

  it('loads the custom risk endpoint with its requested limit', async () => {
    get.mockResolvedValue({ data: { items: [] } })

    await expect(listRegistrationRisk(250)).resolves.toEqual({ items: [] })
    expect(get).toHaveBeenCalledWith('/custom/admin/registration-risk', { params: { limit: 250 } })
  })

  it('submits a required review reason for both override actions', async () => {
    post.mockResolvedValue({ data: { user_id: 603, action: 'manual_unlock' } })

    await unlockRegistrationRisk(603, '已完成充值并人工复核')
    await lockRegistrationRisk(603, '重复注册特征复核')

    expect(post).toHaveBeenNthCalledWith(1, '/custom/admin/registration-risk/603/unlock', { reason: '已完成充值并人工复核' })
    expect(post).toHaveBeenNthCalledWith(2, '/custom/admin/registration-risk/603/lock', { reason: '重复注册特征复核' })
  })
})
