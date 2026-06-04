import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SSQLotteryView from '../SSQLotteryView.vue'

const { createBet, getCurrent, getOrders, getResults, showError, showSuccess } = vi.hoisted(() => ({
  createBet: vi.fn(),
  getCurrent: vi.fn(),
  getOrders: vi.fn(),
  getResults: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
}))

vi.mock('@/api/lottery', () => ({
  lotteryAPI: {
    createBet,
    getCurrent,
    getOrders,
    getResults,
  },
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    showError,
    showSuccess,
  }),
}))

vi.mock('@/utils/format', () => ({
  formatDateTime: (value?: string | null) => (value ? `时间:${value}` : ''),
  formatDualDisplayAmount: (value: number) => ({
    display: String(value),
    full: String(value),
  }),
}))

const openCurrent = {
  lottery_type: 'ssq',
  issue_no: '2026060',
  open_time: '2026-06-04T13:15:00Z',
  cutoff_time: '2026-06-04T13:05:00Z',
  is_closed: false,
  jackpot_balance: '10000070',
}

const closedCurrent = {
  ...openCurrent,
  issue_no: '2026061',
  is_closed: true,
}

const order = {
  order_id: 7,
  lottery_type: 'ssq',
  issue_no: '2026060',
  red_balls: ['01', '08', '12', '18', '25', '33'],
  blue_ball: '09',
  cost: '100',
  status: 'pending',
  reward: '0',
  prize_level: '',
  red_hits: 0,
  blue_hit: false,
  created_at: '2026-06-04T12:00:00Z',
}

const recentResult = {
  lottery_type: 'ssq',
  issue_no: '2026062',
  red_balls: ['02', '04', '07', '14', '28', '29'],
  blue_ball: '09',
  opened_at: '2026-06-02T13:15:00Z',
  source: 'fucai',
  source_ref: 'https://example.test/result/2026062',
  created_at: '2026-06-02T13:30:00Z',
}

function mountPage() {
  return mount(SSQLotteryView, {
    global: {
      stubs: {
        AppLayout: {
          template: '<div data-testid="app-layout"><slot /></div>',
        },
        RouterLink: {
          props: ['to'],
          template: '<a :href="to"><slot /></a>',
        },
      },
    },
  })
}

async function settlePage() {
  await flushPromises()
  await flushPromises()
}

describe('SSQLotteryView', () => {
  beforeEach(() => {
    createBet.mockReset()
    getCurrent.mockReset()
    getOrders.mockReset()
    getResults.mockReset()
    showError.mockReset()
    showSuccess.mockReset()

    getCurrent.mockResolvedValue(openCurrent)
    getOrders.mockResolvedValue([order])
    getResults.mockResolvedValue([recentResult])
    createBet.mockResolvedValue({
      order_id: 8,
      issue_no: '2026060',
      lottery_type: 'ssq',
      cost: '100',
      status: 'pending',
      created_at: '2026-06-04T12:01:00Z',
    })
  })

  it('renders the standalone SSQ lottery page without slot controls', async () => {
    const wrapper = mountPage()
    await settlePage()

    expect(wrapper.text()).toContain('双色球')
    expect(wrapper.text()).toContain('2026060')
    expect(wrapper.text()).toContain('时间:2026-06-04T13:05:00Z')
    expect(wrapper.text()).toContain('10000070 DG')
    expect(wrapper.text()).toContain('可投注')
    expect(wrapper.findAll('.red-ball')).toHaveLength(33)
    expect(wrapper.findAll('.blue-ball')).toHaveLength(16)
    expect(wrapper.text()).toContain('第 2026060 期')
    expect(wrapper.text()).toContain('待开奖')
    expect(wrapper.text()).toContain('最近开奖')
    expect(wrapper.text()).toContain('第 2026062 期')
    expect(wrapper.text()).toContain('来源：fucai')
    expect(wrapper.find('[data-testid="slot-machine"]').exists()).toBe(false)
    expect(getOrders).toHaveBeenCalledWith('2026060')
    expect(getResults).toHaveBeenCalledWith(100)
  })

  it('submits one SSQ bet and refreshes the lottery data', async () => {
    const wrapper = mountPage()
    await settlePage()

    for (const ball of [1, 2, 3, 4, 5, 6]) {
      await wrapper.get(`[data-testid="red-ball-${ball}"]`).trigger('click')
    }
    await wrapper.get('[data-testid="red-ball-7"]').trigger('click')
    await wrapper.get('[data-testid="blue-ball-9"]').trigger('click')

    expect(wrapper.findAll('.red-ball.selected')).toHaveLength(6)
    expect(wrapper.get('[data-testid="red-ball-7"]').classes('selected')).toBe(false)
    expect(wrapper.get('[data-testid="submit-bet"]').attributes('disabled')).toBeUndefined()

    await wrapper.get('[data-testid="submit-bet"]').trigger('click')
    await settlePage()

    expect(createBet).toHaveBeenCalledWith({
      red_balls: [1, 2, 3, 4, 5, 6],
      blue_ball: 9,
    }, '2026060')
    expect(showSuccess).toHaveBeenCalledWith('投注成功，已刷新投注记录')
    expect(getCurrent).toHaveBeenCalledTimes(2)
    expect(getOrders).toHaveBeenCalledTimes(2)
    expect(getResults).toHaveBeenCalledTimes(2)
    expect(wrapper.findAll('.red-ball.selected')).toHaveLength(0)
    expect(wrapper.findAll('.blue-ball.selected')).toHaveLength(0)
  })

  it('disables betting when the current issue is closed', async () => {
    getCurrent.mockResolvedValue(closedCurrent)

    const wrapper = mountPage()
    await settlePage()

    for (const ball of [1, 2, 3, 4, 5, 6]) {
      await wrapper.get(`[data-testid="red-ball-${ball}"]`).trigger('click')
    }
    await wrapper.get('[data-testid="blue-ball-9"]').trigger('click')

    expect(wrapper.text()).toContain('已截止')
    expect(wrapper.text()).toContain('本期已截止投注。')
    expect(wrapper.get('[data-testid="submit-bet"]').attributes('disabled')).toBeDefined()
    await wrapper.get('[data-testid="submit-bet"]').trigger('click')
    expect(createBet).not.toHaveBeenCalled()
  })

  it('keeps betting available when recent results fail to load', async () => {
    getResults.mockRejectedValue(new Error('result api failed'))

    const wrapper = mountPage()
    await settlePage()

    expect(wrapper.text()).toContain('result api failed')
    expect(wrapper.text()).toContain('可投注')

    for (const ball of [1, 2, 3, 4, 5, 6]) {
      await wrapper.get(`[data-testid="red-ball-${ball}"]`).trigger('click')
    }
    await wrapper.get('[data-testid="blue-ball-9"]').trigger('click')
    expect(wrapper.get('[data-testid="submit-bet"]').attributes('disabled')).toBeUndefined()
    expect(showError).not.toHaveBeenCalled()
  })
})
