import { flushPromises, mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import AppHeader from '@/components/layout/AppHeader.vue'

const checkinStore = vi.hoisted(() => ({
  status: {
    enabled: true,
    luck_enabled: true,
    can_checkin: true,
    balance: 20,
    min_multiplier: 0.5,
    max_multiplier: 1.5,
  } as Record<string, unknown> | null,
  statusLoading: false,
  loading: false,
  actionError: null,
  enabled: true,
  normalEnabled: true,
  luckEnabled: true,
  canCheckin: true,
  checkedInToday: false,
  fetchStatus: vi.fn().mockResolvedValue(true),
  doCheckin: vi.fn(),
  doLuckCheckin: vi.fn(),
  clearActionError: vi.fn(),
}))

const appStore = vi.hoisted(() => ({
  contactInfo: '',
  docUrl: '',
  cachedPublicSettings: { custom_menu_items: [] },
  toggleMobileSidebar: vi.fn(),
  showSuccess: vi.fn(),
}))
const authStore = vi.hoisted(() => ({
  user: { id: 1, username: 'alice', email: 'alice@example.com', role: 'user', balance: 20, frozen_balance: 0 },
  isAdmin: false,
  isSimpleMode: false,
  logout: vi.fn(),
}))

vi.mock('@/stores', () => ({
  useAppStore: () => appStore,
  useAuthStore: () => authStore,
  useOnboardingStore: () => ({ replay: vi.fn() }),
}))
vi.mock('@/stores/checkin', () => ({ useCheckinStore: () => checkinStore }))
vi.mock('@/stores/adminSettings', () => ({ useAdminSettingsStore: () => ({ customMenuItems: [] }) }))
vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn() }),
  useRoute: () => ({ name: 'Dashboard', params: {}, meta: {} }),
}))

const BaseDialogStub = defineComponent({
  props: { show: Boolean },
  template: '<section v-if="show"><slot /><footer><slot name="footer" /></footer></section>',
})

function mountHeader() {
  return mount(AppHeader, {
    global: {
      plugins: [createI18n({ legacy: false, locale: 'en', missingWarn: false, fallbackWarn: false, messages: { en: {} } })],
      stubs: {
        BaseDialog: BaseDialogStub,
        AnnouncementBell: true,
        LocaleSwitcher: true,
        SubscriptionProgressMini: true,
        Icon: true,
        RouterLink: true,
      },
    },
  })
}

describe('AppHeader check-in actions', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    checkinStore.status = { enabled: true, luck_enabled: true, can_checkin: true, balance: 20, min_multiplier: 0.5, max_multiplier: 1.5 }
    checkinStore.enabled = true
    checkinStore.normalEnabled = true
    checkinStore.luckEnabled = true
    checkinStore.canCheckin = true
    checkinStore.checkedInToday = false
    checkinStore.loading = false
    checkinStore.actionError = null
    checkinStore.doCheckin.mockImplementation(async () => {
      checkinStore.canCheckin = false
      checkinStore.checkedInToday = true
      return { reward_amount: 1 }
    })
    checkinStore.doLuckCheckin.mockImplementation(async () => {
      checkinStore.canCheckin = false
      checkinStore.checkedInToday = true
      return { reward_amount: 2 }
    })
  })

  it('renders both header actions and performs normal check-in directly', async () => {
    const wrapper = mountHeader()
    await flushPromises()

    expect(wrapper.find('[data-testid="header-normal-checkin"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="header-luck-checkin"]').exists()).toBe(true)
    await wrapper.get('[data-testid="header-normal-checkin"]').trigger('click')
    expect(checkinStore.doCheckin).toHaveBeenCalledOnce()
  })

  it('submits a header lucky check-in directly after entering an amount', async () => {
    const wrapper = mountHeader()
    await wrapper.get('[data-testid="header-luck-checkin"]').trigger('click')
    await wrapper.get('[data-testid="luck-bet-input"]').setValue('5')
    await wrapper.get('[data-testid="luck-submit"]').trigger('click')

    expect(checkinStore.doLuckCheckin).toHaveBeenCalledWith(5)
  })

  it('hides header actions when check-in is disabled', () => {
    checkinStore.enabled = false
    const wrapper = mountHeader()
    expect(wrapper.find('[data-testid="header-normal-checkin"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="header-luck-checkin"]').exists()).toBe(false)
  })

  it('shows a temporary reward tip after a successful check-in', async () => {
    vi.useFakeTimers()

    const wrapper = mountHeader()
    await wrapper.get('[data-testid="header-normal-checkin"]').trigger('click')
    await flushPromises()
    wrapper.vm.$forceUpdate()
    await wrapper.vm.$nextTick()

    expect(wrapper.find('[data-testid="header-normal-checkin"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="header-luck-checkin"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="header-checkin-tip"]').text()).toContain('+$1.00')

    vi.advanceTimersByTime(5000)
    await wrapper.vm.$nextTick()
    expect(wrapper.find('[data-testid="header-checkin-tip"]').exists()).toBe(false)
    vi.useRealTimers()
  })
})
