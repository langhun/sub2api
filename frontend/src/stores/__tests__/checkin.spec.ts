import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useCheckinStore } from '@/stores/checkin'

const api = vi.hoisted(() => ({
  getCheckinStatus: vi.fn(),
  checkin: vi.fn(),
  luckCheckin: vi.fn(),
}))

vi.mock('@/api/checkin', () => ({
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
})
