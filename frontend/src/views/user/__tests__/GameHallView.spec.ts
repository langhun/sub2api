import { flushPromises, mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import GameHallView from '@/views/user/GameHallView.vue'

const api = vi.hoisted(() => ({
  getGameTransactions: vi.fn(),
  getGameRounds: vi.fn(),
}))

const appStore = vi.hoisted(() => ({ showSuccess: vi.fn(), showError: vi.fn() }))
const { gameStore } = vi.hoisted(() => {
  const slots = {
    type: 'slots',
    name: 'Audited Slots',
    min_bet: 1,
    max_bet: 10,
    multipliers: [0, 18.7],
    rule_version: 'slots-v1',
    theoretical_rtp: 0.953,
    payout_rules: [
      { symbol: 'cherry', match_count: 3, multiplier: 18.7, probability: 0.0166 },
    ],
  }
  return {
    slots,
    gameStore: {
      status: {
        main_balance: 100,
        dg_balance: 20,
        jackpot_balance: 500,
        exchange_min_amount: 1,
        exchange_max_amount: 50,
        exchange_daily_limit: 100,
        exchange_daily_used: 25,
        exchange_daily_remaining: 75,
        exchange_allow_dg_to_balance: true,
        games: [slots],
      },
      lastExchange: null,
      lastRound: null,
      loading: false,
      submitting: false,
      error: '',
      enabledGames: [slots],
      refresh: vi.fn(),
      exchange: vi.fn(),
      play: vi.fn(),
    },
  }
})

vi.mock('@/api/gameHall', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/api/gameHall')>()
  return {
    ...actual,
    getGameTransactions: api.getGameTransactions,
    getGameRounds: api.getGameRounds,
  }
})
vi.mock('@/stores/app', () => ({ useAppStore: () => appStore }))
vi.mock('@/stores/gameHall', () => ({ useGameHallStore: () => gameStore }))

const AppLayoutStub = defineComponent({ template: '<main><slot /></main>' })
const ConfirmDialogStub = defineComponent({
  props: { show: Boolean },
  emits: ['confirm', 'cancel'],
  template: '<button v-if="show" data-testid="exchange-confirm" @click="$emit(\'confirm\')">confirm</button>',
})

function mountView() {
  return mount(GameHallView, {
    global: {
      plugins: [createI18n({ legacy: false, locale: 'en', missingWarn: false, fallbackWarn: false, messages: { en: {} } })],
      stubs: {
        AppLayout: AppLayoutStub,
        ConfirmDialog: ConfirmDialogStub,
        Icon: true,
        LoadingSpinner: true,
        Pagination: true,
      },
    },
  })
}

describe('GameHallView rules and operation errors', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    gameStore.refresh.mockResolvedValue(undefined)
    gameStore.exchange.mockResolvedValue(undefined)
    gameStore.play.mockResolvedValue(undefined)
    api.getGameTransactions.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 10 })
    api.getGameRounds.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 10 })
  })

  it('renders the exact server-provided rule version, RTP, payout and probability', async () => {
    const wrapper = mountView()
    await flushPromises()

    const rules = wrapper.get('[data-testid="slot-payout-rules"]')
    expect(wrapper.text()).toContain('slots-v1')
    expect(wrapper.text()).toContain('95.30%')
    expect(rules.text()).toContain('18.70x')
    expect(rules.text()).toContain('1.66%')
  })

  it('keeps an exchange failure visible and requires confirmation before retrying', async () => {
    gameStore.exchange.mockRejectedValue({ response: { data: { detail: 'exchange temporarily locked' } } })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="game-exchange-form"]').trigger('submit')
    await wrapper.get('[data-testid="exchange-confirm"]').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('exchange temporarily locked')
    expect(gameStore.exchange).toHaveBeenCalledTimes(1)

    const retry = wrapper.get('[role="alert"] button')
    await retry.trigger('click')
    await wrapper.get('[data-testid="exchange-confirm"]').trigger('click')
    await flushPromises()
    expect(gameStore.exchange).toHaveBeenCalledTimes(2)
  })

  it('renders server exchange limits and removes the return direction when disabled', async () => {
    gameStore.status.exchange_allow_dg_to_balance = false
    const wrapper = mountView()
    await flushPromises()

    const amount = wrapper.get('[data-testid="exchange-amount"]')
    expect(amount.attributes('min')).toBe('1')
    expect(amount.attributes('max')).toBe('50')
    expect(wrapper.find('[data-testid="exchange-direction-balance_to_dg"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="exchange-direction-dg_to_balance"]').exists()).toBe(false)
  })
})
