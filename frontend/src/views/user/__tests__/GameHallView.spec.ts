import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import GameHallView from '../GameHallView.vue'

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

const winOrder = {
  ...order,
  order_id: 8,
  status: 'win',
  reward: '5',
  prize_level: 'sixth',
  red_hits: 2,
  blue_hit: true,
}

const loseOrder = {
  ...order,
  order_id: 9,
  status: 'lose',
  reward: '0',
  prize_level: '',
  red_hits: 1,
  blue_hit: false,
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
  return mount(GameHallView, {
    global: {
      stubs: {
        AppLayout: {
          template: '<div data-testid="app-layout"><slot /></div>',
        },
      },
    },
  })
}

async function settlePage() {
  await flushPromises()
  await flushPromises()
}

describe('GameHallView lottery page', () => {
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

  it('renders current issue, jackpot, selector grids, and my orders', async () => {
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
    expect(wrapper.text()).toContain('命中：0 红')
    expect(wrapper.text()).toContain('最近开奖')
    expect(wrapper.text()).toContain('第 2026062 期')
    expect(wrapper.text()).toContain('02')
    expect(wrapper.text()).toContain('来源：fucai')
    expect(getOrders).toHaveBeenCalledWith('2026060')
    expect(getResults).toHaveBeenCalledWith(100)
  })

  it('renders win and lose order settlement states', async () => {
    getOrders.mockResolvedValue([winOrder, loseOrder, order])

    const wrapper = mountPage()
    await settlePage()

    expect(wrapper.text()).toContain('已中奖')
    expect(wrapper.text()).toContain('未中奖')
    expect(wrapper.text()).toContain('待开奖')
    expect(wrapper.text()).toContain('奖级：六等奖')
    expect(wrapper.text()).toContain('奖励：5 DG')
    expect(wrapper.text()).toContain('命中：2 红 + 蓝')
    expect(wrapper.text()).toContain('命中：1 红')
  })

  it('limits red balls, keeps one blue ball, and prevents duplicate submit while pending', async () => {
    let resolveBet!: (value: unknown) => void
    createBet.mockReturnValue(new Promise((resolve) => {
      resolveBet = resolve
    }))

    const wrapper = mountPage()
    await settlePage()

    expect(wrapper.get('[data-testid="submit-bet"]').attributes('disabled')).toBeDefined()

    for (const ball of [1, 2, 3, 4, 5, 6]) {
      await wrapper.get(`[data-testid="red-ball-${ball}"]`).trigger('click')
    }
    await wrapper.get('[data-testid="red-ball-7"]').trigger('click')
    expect(wrapper.findAll('.red-ball.selected')).toHaveLength(6)
    expect(wrapper.get('[data-testid="red-ball-7"]').classes('selected')).toBe(false)

    await wrapper.get('[data-testid="blue-ball-9"]').trigger('click')
    await wrapper.get('[data-testid="blue-ball-10"]').trigger('click')
    expect(wrapper.findAll('.blue-ball.selected')).toHaveLength(1)
    expect(wrapper.get('[data-testid="blue-ball-9"]').classes('selected')).toBe(false)
    expect(wrapper.get('[data-testid="blue-ball-10"]').classes('selected')).toBe(true)
    expect(wrapper.get('[data-testid="submit-bet"]').attributes('disabled')).toBeUndefined()

    await wrapper.get('[data-testid="submit-bet"]').trigger('click')
    await wrapper.get('[data-testid="submit-bet"]').trigger('click')

    expect(createBet).toHaveBeenCalledTimes(1)
    expect(createBet).toHaveBeenCalledWith({
      red_balls: [1, 2, 3, 4, 5, 6],
      blue_ball: 10,
    }, '2026060')

    resolveBet({
      order_id: 8,
      issue_no: '2026060',
      lottery_type: 'ssq',
      cost: '100',
      status: 'pending',
    })
    await settlePage()

    expect(showSuccess).toHaveBeenCalledWith('投注成功，已刷新投注记录')
    expect(getCurrent).toHaveBeenCalledTimes(2)
    expect(getOrders).toHaveBeenCalledTimes(2)
    expect(getResults).toHaveBeenCalledTimes(2)
    expect(wrapper.findAll('.red-ball.selected')).toHaveLength(0)
    expect(wrapper.findAll('.blue-ball.selected')).toHaveLength(0)
  })

  it('disables betting when current issue is closed', async () => {
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

  it('shows readable error when current issue fails to load', async () => {
    getCurrent.mockRejectedValue({ reason: 'LOTTERY_ISSUE_CLOSED' })
    getOrders.mockResolvedValue([])

    const wrapper = mountPage()
    await settlePage()

    expect(wrapper.text()).toContain('本期已截止投注，请等待下一期。')
    expect(showError).toHaveBeenCalledWith('本期已截止投注，请等待下一期。')
    expect(wrapper.get('[data-testid="submit-bet"]').attributes('disabled')).toBeDefined()
  })

  it('keeps betting available when result loading fails', async () => {
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
