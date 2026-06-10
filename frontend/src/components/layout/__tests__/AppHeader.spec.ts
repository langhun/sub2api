import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import AppHeader from '../AppHeader.vue'

const pushMock = vi.fn()

const { appStoreMock, authStoreMock, onboardingStoreMock, adminSettingsStoreMock, checkinStoreMock } = vi.hoisted(() => ({
  appStoreMock: {
    contactInfo: '',
    docUrl: '',
    cachedPublicSettings: null,
    toggleMobileSidebar: vi.fn(),
  },
  authStoreMock: {
    user: {
      role: 'admin',
      username: 'LH',
      email: 'i@daigua.icu',
      balance: 77080135243.14,
      avatar_url: '',
    },
    isAdmin: true,
    isSimpleMode: false,
    logout: vi.fn(),
  },
  onboardingStoreMock: {
    replay: vi.fn(),
  },
  adminSettingsStoreMock: {
    customMenuItems: [],
  },
  checkinStoreMock: {
    loading: false,
    enabled: true,
    normalEnabled: true,
    luckEnabled: true,
    canCheckin: false,
    checkedInToday: true,
    todayReward: 33777082859.35,
    status: {
      min_multiplier: 0.1,
      max_multiplier: 3,
      balance: 20,
    },
    blindboxResult: null,
    fetchStatus: vi.fn(),
    doCheckin: vi.fn(),
    doLuckCheckin: vi.fn(),
    clearBlindboxResult: vi.fn(),
  },
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: pushMock,
  }),
  useRoute: () => ({
    name: 'Dashboard',
    meta: {},
    params: {},
  }),
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

vi.mock('@/stores', () => ({
  useAppStore: () => appStoreMock,
  useAuthStore: () => authStoreMock,
  useOnboardingStore: () => onboardingStoreMock,
}))

vi.mock('@/stores/adminSettings', () => ({
  useAdminSettingsStore: () => adminSettingsStoreMock,
}))

vi.mock('@/stores/checkin', () => ({
  useCheckinStore: () => checkinStoreMock,
}))

function mountHeader() {
  return mount(AppHeader, {
    global: {
      stubs: {
        AnnouncementBell: true,
        BlindboxModal: true,
        Icon: true,
        LocaleSwitcher: true,
        LuckCheckinDialog: true,
        PublicQuickLinksBar: true,
        SubscriptionProgressMini: true,
        RouterLink: {
          template: '<a><slot /></a>',
        },
      },
    },
  })
}

describe('AppHeader', () => {
  beforeEach(() => {
    pushMock.mockReset()
    appStoreMock.toggleMobileSidebar.mockReset()
    authStoreMock.logout.mockReset()
    onboardingStoreMock.replay.mockReset()
    checkinStoreMock.fetchStatus.mockReset()
    checkinStoreMock.doCheckin.mockReset()
    checkinStoreMock.doLuckCheckin.mockReset()
    checkinStoreMock.clearBlindboxResult.mockReset()
    checkinStoreMock.loading = false
    checkinStoreMock.enabled = true
    checkinStoreMock.normalEnabled = true
    checkinStoreMock.luckEnabled = true
    checkinStoreMock.canCheckin = false
    checkinStoreMock.checkedInToday = true
    checkinStoreMock.todayReward = 33777082859.35
    checkinStoreMock.status = {
      min_multiplier: 0.1,
      max_multiplier: 3,
      balance: 20,
    }
  })

  it('does not keep any daily check-in result pinned in the header after sign-in completes', () => {
    const wrapper = mountHeader()

    expect(checkinStoreMock.fetchStatus).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('$77080135243.14')
    expect(wrapper.text()).not.toContain('+$33777082859.35')
    expect(wrapper.text()).not.toContain('checkin.checked')
  })
})
