import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import OpsConcurrencyCard from '../OpsConcurrencyCard.vue'

const { getConcurrencyStats, getAccountAvailabilityStats, getUserConcurrencyStats } = vi.hoisted(() => ({
  getConcurrencyStats: vi.fn(),
  getAccountAvailabilityStats: vi.fn(),
  getUserConcurrencyStats: vi.fn(),
}))

vi.mock('@/api/admin/ops', () => ({
  opsAPI: { getConcurrencyStats, getAccountAvailabilityStats, getUserConcurrencyStats },
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

describe('OpsConcurrencyCard', () => {
  it('shows one username-first user label without repeating the email', async () => {
    getConcurrencyStats.mockResolvedValue({ enabled: true, platform: {}, group: {}, account: {} })
    getAccountAvailabilityStats.mockResolvedValue({ enabled: true, platform: {}, group: {}, account: {} })
    getUserConcurrencyStats.mockResolvedValue({
      enabled: true,
      user: {
        7: {
          user_id: 7,
          username: 'Concurrent User',
          user_email: 'concurrent@example.com',
          current_in_use: 1,
          max_capacity: 2,
          waiting_in_queue: 0,
          load_percentage: 50,
        },
      },
    })

    const wrapper = mount(OpsConcurrencyCard, { props: { refreshToken: 0 } })
    await flushPromises()
    await wrapper.get('[title="admin.ops.concurrency.switchToUser"]').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('Concurrent User')
    expect(wrapper.text()).not.toContain('concurrent@example.com')
  })
})
