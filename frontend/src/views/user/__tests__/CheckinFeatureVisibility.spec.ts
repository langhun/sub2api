import { flushPromises, mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import CheckinView from '@/views/user/CheckinView.vue'

const api = vi.hoisted(() => ({
  getBlindboxRecords: vi.fn(),
  getCheckinCalendar: vi.fn(),
}))

const appStore = vi.hoisted(() => ({
  cachedPublicSettings: { checkin_blindbox_enabled: false } as Record<string, boolean>,
}))

const checkinStore = vi.hoisted(() => ({
  status: {
    enabled: true,
    luck_enabled: false,
    blindbox_enabled: false,
    can_checkin: true,
    streak_days: 0,
    min_reward: 0.1,
    max_reward: 1,
    balance: 10,
  } as Record<string, unknown>,
  statusLoading: false,
  statusError: null,
  loading: false,
  actionError: null,
  blindboxResult: { rarity: 'rare', prize_name: 'stale prize', reward_type: 'balance', reward_value: 1 },
  enabled: true,
  normalEnabled: true,
  luckEnabled: false,
  canCheckin: true,
  streakDays: 0,
  todayReward: null,
  todayCheckinType: null,
  todayMultiplier: null,
  fetchStatus: vi.fn().mockResolvedValue(true),
  doCheckin: vi.fn(),
  doLuckCheckin: vi.fn(),
  clearBlindboxResult: vi.fn(),
  clearActionError: vi.fn(),
}))

vi.mock('@/api/checkin', () => ({
  getBlindboxRecords: api.getBlindboxRecords,
  getCheckinCalendar: api.getCheckinCalendar,
}))
vi.mock('@/stores/app', () => ({ useAppStore: () => appStore }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => ({ user: { balance: 10, concurrency: 2 } }) }))
vi.mock('@/stores/checkin', () => ({ useCheckinStore: () => checkinStore }))

const AppLayoutStub = defineComponent({ template: '<main><slot /></main>' })
const BaseDialogStub = defineComponent({
  props: { show: Boolean },
  template: '<section v-if="show"><slot /><footer><slot name="footer" /></footer></section>',
})

function mountView() {
  return mount(CheckinView, {
    global: {
      plugins: [createI18n({ legacy: false, locale: 'en', missingWarn: false, fallbackWarn: false, messages: { en: {} } })],
      stubs: { AppLayout: AppLayoutStub, BaseDialog: BaseDialogStub },
    },
  })
}

describe('CheckinView feature visibility', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    appStore.cachedPublicSettings = { checkin_blindbox_enabled: false }
    Object.assign(checkinStore.status, {
      enabled: true,
      luck_enabled: false,
      blindbox_enabled: false,
      can_checkin: true,
      streak_days: 0,
      min_reward: 0.1,
      max_reward: 1,
      min_multiplier: 0.5,
      max_multiplier: 1.5,
      balance: 10,
    })
    checkinStore.statusLoading = false
    checkinStore.statusError = null
    checkinStore.loading = false
    checkinStore.actionError = null
    checkinStore.enabled = true
    checkinStore.normalEnabled = true
    checkinStore.luckEnabled = false
    checkinStore.canCheckin = true
    checkinStore.fetchStatus.mockResolvedValue(true)
    checkinStore.doCheckin.mockResolvedValue(null)
    checkinStore.doLuckCheckin.mockResolvedValue(null)
    api.getCheckinCalendar.mockResolvedValue({ days: [] })
    api.getBlindboxRecords.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 20 })
  })

  it('hides every blind-box surface and skips history requests when disabled', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(api.getBlindboxRecords).not.toHaveBeenCalled()
    expect(wrapper.text()).not.toContain('checkin.page.blindboxCount')
    expect(wrapper.text()).not.toContain('checkin.page.rarityBreakdown')
    expect(wrapper.text()).not.toContain('checkin.page.blindboxInfo')
    expect(wrapper.text()).not.toContain('checkin.blindboxHistory')
    expect(wrapper.text()).not.toContain('stale prize')
  })

  it('loads and renders blind-box sections only when both status and public switch allow them', async () => {
    appStore.cachedPublicSettings = { checkin_blindbox_enabled: true }
    checkinStore.status.blindbox_enabled = true

    const wrapper = mountView()
    await flushPromises()

    expect(api.getBlindboxRecords).toHaveBeenCalledWith(1, 20)
    expect(wrapper.text()).toContain('checkin.page.blindboxCount')
    expect(wrapper.text()).toContain('checkin.blindboxHistory')
  })

  it('shows a status error and lets the user retry loading it', async () => {
    checkinStore.statusError = new Error('status service offline')

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('status service offline')
    await wrapper.get('[data-testid="checkin-status-retry"]').trigger('click')
    expect(checkinStore.fetchStatus).toHaveBeenCalledTimes(2)
  })

  it('shows a failed normal check-in and retries the same action', async () => {
    checkinStore.actionError = new Error('normal check-in failed')

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('normal check-in failed')
    await wrapper.get('[data-testid="checkin-action-retry"]').trigger('click')
    expect(checkinStore.doCheckin).toHaveBeenCalledTimes(1)
  })

  it('submits a lucky check-in directly after entering the amount', async () => {
    checkinStore.status.enabled = false
    checkinStore.status.luck_enabled = true
    checkinStore.normalEnabled = false
    checkinStore.luckEnabled = true

    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="luck-checkin-open"]').trigger('click')
    await wrapper.get('[data-testid="luck-bet-input"]').setValue('4')
    await wrapper.get('[data-testid="luck-submit"]').trigger('click')
    expect(checkinStore.doLuckCheckin).toHaveBeenCalledOnce()
    expect(checkinStore.doLuckCheckin).toHaveBeenCalledWith(4)
  })
})
