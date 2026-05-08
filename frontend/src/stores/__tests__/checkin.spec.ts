import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const { checkinAPIMock, authStoreMock } = vi.hoisted(() => ({
  checkinAPIMock: {
    checkin: vi.fn(),
    luckCheckin: vi.fn(),
    getCheckinStatus: vi.fn(),
  },
  authStoreMock: {
    user: { balance: 0 },
    refreshUser: vi.fn(),
  },
}))

vi.mock('@/api/checkin', () => ({
  checkinAPI: checkinAPIMock,
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authStoreMock,
}))

import { setLocale } from '@/i18n'
import { useAppStore } from '@/stores/app'
import { useCheckinStore } from '@/stores/checkin'

function createStatus() {
  return {
    enabled: true,
    luck_enabled: true,
    blindbox_enabled: false,
    can_checkin: true,
    streak_days: 1,
    today_reward: null,
    today_checkin_type: undefined,
    today_multiplier: undefined,
    min_reward: 0.5,
    max_reward: 1.5,
    min_multiplier: 0.1,
    max_multiplier: 3,
    balance: 20,
  }
}

describe('useCheckinStore', () => {
  beforeEach(async () => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    localStorage.setItem('sub2api_locale', 'zh')
    await setLocale('zh')
    authStoreMock.user = { balance: 20 }
    authStoreMock.refreshUser.mockResolvedValue(undefined)
  })

  it('luck check-in 成功后显示明确成功提示并同步状态', async () => {
    const store = useCheckinStore()
    const appStore = useAppStore()

    store.status = createStatus()
    checkinAPIMock.luckCheckin.mockResolvedValue({
      reward_amount: -1.25,
      streak_days: 3,
      checked_at: '2026-05-08',
      checkin_type: 'luck',
      bet_amount: 10,
      multiplier: 0.88,
    })

    const result = await store.doLuckCheckin(10)

    expect(result).not.toBeNull()
    expect(store.status?.can_checkin).toBe(false)
    expect(store.status?.streak_days).toBe(3)
    expect(store.status?.today_reward).toBe(-1.25)
    expect(store.status?.today_checkin_type).toBe('luck')
    expect(store.status?.today_multiplier).toBe(0.88)
    expect(appStore.toasts).toHaveLength(1)
    expect(appStore.toasts[0].type).toBe('success')
    expect(appStore.toasts[0].message).toBe('运气签到成功！倍率 $0.88x，失去 $1.25')
  })

  it('luck check-in 失败后显示错误提示并刷新状态', async () => {
    const store = useCheckinStore()
    const appStore = useAppStore()

    store.status = createStatus()
    checkinAPIMock.luckCheckin.mockRejectedValue({
      reason: 'INVALID_BET_AMOUNT',
      message: 'bet amount must be greater than 0 and not exceed your balance',
    })
    checkinAPIMock.getCheckinStatus.mockResolvedValue({
      ...createStatus(),
      balance: 4.56,
    })

    const result = await store.doLuckCheckin(10)

    expect(result).toBeNull()
    expect(checkinAPIMock.getCheckinStatus).toHaveBeenCalledTimes(1)
    expect(store.status?.balance).toBe(4.56)
    expect(appStore.toasts).toHaveLength(1)
    expect(appStore.toasts[0].type).toBe('error')
    expect(appStore.toasts[0].message).toBe('余额不足')
  })
})
