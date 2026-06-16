import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import TopUsersLeaderboard from '../TopUsersLeaderboard.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

describe('TopUsersLeaderboard', () => {
  it('prefers username and keeps email as secondary text', () => {
    const wrapper = mount(TopUsersLeaderboard, {
      props: {
        users: [
          { user_id: 1, username: 'Alice', email: 'alice@example.com', amount: 88.8 },
        ],
      },
    })

    expect(wrapper.text()).toContain('Alice')
    expect(wrapper.text()).toContain('alice@example.com')
  })

  it('falls back to email when username is empty', () => {
    const wrapper = mount(TopUsersLeaderboard, {
      props: {
        users: [
          { user_id: 2, username: '', email: 'fallback@example.com', amount: 12.3 },
        ],
      },
    })

    expect(wrapper.text()).toContain('fallback@example.com')
  })
})
