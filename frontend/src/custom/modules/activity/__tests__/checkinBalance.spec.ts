import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useCheckinStore } from '@/custom/modules/activity/stores/checkin'

const api = vi.hoisted(() => ({
  getCheckinStatus: vi.fn(),
  checkin: vi.fn(),
  luckCheckin: vi.fn(),
}))

const auth = vi.hoisted(() => ({
  user: { balance: 10 },
  refreshUser: vi.fn(),
}))

vi.mock('@/custom/modules/activity/api/checkin', () => ({
  checkinAPI: api,
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => auth,
}))

describe('check-in balance display', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    vi.spyOn(console, 'warn').mockImplementation(() => undefined)
    auth.user = { balance: 10 }
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('updates the displayed balance immediately after a lucky reward', async () => {
    const store = useCheckinStore()
    store.status = {
      enabled: false,
      luck_enabled: true,
      blindbox_enabled: false,
      can_checkin: true,
      streak_days: 0,
      today_reward: null,
      min_reward: 0,
      max_reward: 0,
      min_multiplier: 0.1,
      max_multiplier: 1.7,
      balance: 10,
    }
    api.luckCheckin.mockResolvedValueOnce({
      reward_amount: 0.6,
      streak_days: 1,
      checked_at: '2026-07-30',
      checkin_type: 'luck',
      bet_amount: 2,
      multiplier: 1.3,
    })
    auth.refreshUser.mockRejectedValueOnce(new Error('temporary profile refresh failure'))

    await expect(store.doLuckCheckin(2)).resolves.toMatchObject({ reward_amount: 0.6 })
    expect(auth.user.balance).toBeCloseTo(10.6)
    expect(store.status?.balance).toBeCloseTo(10.6)
    expect(auth.refreshUser).toHaveBeenCalledOnce()
  })
})
