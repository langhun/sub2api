import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useCheckinStore } from '@/custom/modules/activity/stores/checkin'

const api = vi.hoisted(() => ({
  getCheckinStatus: vi.fn(),
  checkin: vi.fn(),
  luckCheckin: vi.fn(),
}))

vi.mock('@/custom/modules/activity/api/checkin', () => ({
  checkinAPI: api,
}))

describe('checkin store errors', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('keeps a visible status error until a later retry succeeds', async () => {
    const store = useCheckinStore()
    const failure = new Error('status unavailable')
    api.getCheckinStatus.mockRejectedValueOnce(failure)

    await expect(store.fetchStatus()).resolves.toBe(false)
    expect(store.statusError).toBe(failure)
    expect(store.statusLoading).toBe(false)

    api.getCheckinStatus.mockResolvedValueOnce({ enabled: true, can_checkin: true })
    await expect(store.fetchStatus()).resolves.toBe(true)
    expect(store.statusError).toBeNull()
    expect(store.status?.can_checkin).toBe(true)
  })

  it('captures normal and lucky action failures and supports explicit dismissal', async () => {
    const store = useCheckinStore()
    const normalFailure = new Error('normal failed')
    const luckFailure = new Error('luck failed')

    api.checkin.mockRejectedValueOnce(normalFailure)
    await expect(store.doCheckin()).resolves.toBeNull()
    expect(store.actionError).toBe(normalFailure)

    api.luckCheckin.mockRejectedValueOnce(luckFailure)
    await expect(store.doLuckCheckin(2)).resolves.toBeNull()
    expect(api.luckCheckin).toHaveBeenCalledWith(2, false)
    expect(store.actionError).toBe(luckFailure)

    store.clearActionError()
    expect(store.actionError).toBeNull()
  })

  it('keeps a committed lucky check-in successful when profile refresh fails', async () => {
    const store = useCheckinStore()
    store.status = {
      enabled: true,
      luck_enabled: true,
      blindbox_enabled: false,
      can_checkin: true,
      streak_days: 3,
      today_reward: null,
      min_reward: 1,
      max_reward: 5,
      min_multiplier: 0.1,
      max_multiplier: 1.7,
      balance: 10,
    }
    const result = {
      reward_amount: -2.5,
      streak_days: 4,
      checked_at: '2026-07-31',
      checkin_type: 'luck',
      bet_amount: 10,
      multiplier: 0.75,
    }
    api.luckCheckin.mockResolvedValueOnce(result)

    await expect(store.doLuckCheckin(10, true)).resolves.toEqual(result)
    expect(store.actionError).toBeNull()
    expect(store.status?.can_checkin).toBe(false)
    expect(store.status?.balance).toBe(7.5)
  })
})
