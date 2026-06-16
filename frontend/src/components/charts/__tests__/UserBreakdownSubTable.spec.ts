import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import UserBreakdownSubTable from '../UserBreakdownSubTable.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

describe('UserBreakdownSubTable', () => {
  it('prefers username and keeps email as secondary text', () => {
    const wrapper = mount(UserBreakdownSubTable, {
      props: {
        items: [
          {
            user_id: 1,
            email: 'alpha@example.com',
            username: 'Alpha',
            requests: 5,
            total_tokens: 1000,
            cost: 1.2,
            actual_cost: 1.0,
            account_cost: 0.8,
          },
        ],
      },
      global: {
        stubs: {
          LoadingSpinner: true,
        },
      },
    })

    expect(wrapper.text()).toContain('Alpha')
    expect(wrapper.text()).toContain('alpha@example.com')
  })

  it('falls back to email when username is empty', () => {
    const wrapper = mount(UserBreakdownSubTable, {
      props: {
        items: [
          {
            user_id: 2,
            email: 'beta@example.com',
            username: '',
            requests: 3,
            total_tokens: 600,
            cost: 0.7,
            actual_cost: 0.6,
            account_cost: 0.5,
          },
        ],
      },
      global: {
        stubs: {
          LoadingSpinner: true,
        },
      },
    })

    expect(wrapper.text()).toContain('beta@example.com')
  })
})
