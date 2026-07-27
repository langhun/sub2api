import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, put } = vi.hoisted(() => ({
  get: vi.fn(),
  put: vi.fn(),
}))

vi.mock('@/api/client', () => ({ apiClient: { get, put } }))

import { getGameHallUserAccess, updateGameHallUserAccess } from '../admin'

describe('game hall admin API', () => {
  beforeEach(() => {
    get.mockReset()
    put.mockReset()
    get.mockResolvedValue({ data: { user_id: 7, disabled: false, updated_at: '2026-07-27T00:00:00Z' } })
    put.mockResolvedValue({ data: { user_id: 7, disabled: true, updated_at: '2026-07-27T00:01:00Z' } })
  })

  it('reads module-owned user access through the custom admin route', async () => {
    await expect(getGameHallUserAccess(7)).resolves.toMatchObject({ user_id: 7, disabled: false })

    expect(get).toHaveBeenCalledWith('/admin/game-hall/users/7/access')
  })

  it('updates module-owned user access without the core user API', async () => {
    await expect(updateGameHallUserAccess(7, true)).resolves.toMatchObject({ user_id: 7, disabled: true })

    expect(put).toHaveBeenCalledWith('/admin/game-hall/users/7/access', { disabled: true })
  })
})
