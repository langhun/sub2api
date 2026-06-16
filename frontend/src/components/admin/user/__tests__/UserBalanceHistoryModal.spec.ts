import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

const apiMocks = vi.hoisted(() => ({
  getById: vi.fn(),
  getUserBalanceHistory: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    users: {
      getById: apiMocks.getById,
      getUserBalanceHistory: apiMocks.getUserBalanceHistory,
    },
  },
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

vi.mock('@/components/common/BaseDialog.vue', () => ({
  default: {
    name: 'BaseDialog',
    props: ['show', 'title'],
    template: '<div v-if="show"><slot /></div>',
  },
}))

import UserBalanceHistoryModal from '../UserBalanceHistoryModal.vue'

describe('UserBalanceHistoryModal', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    apiMocks.getById.mockResolvedValue({
      id: 3,
      email: '3023208139@tiu.edu.cn',
      username: '',
      role: 'user',
      balance: 9092.08,
      concurrency: 1,
      status: 'active',
      allowed_groups: [],
      balance_notify_enabled: false,
      balance_notify_threshold: null,
      balance_notify_extra_emails: [],
      created_at: '2026-06-13T00:12:59Z',
      updated_at: '2026-06-13T00:12:59Z',
      notes: '',
      signup_source: 'google',
      inviter_user: {
        id: 88,
        email: 'inviter@example.com',
        username: 'boss-user',
      },
    })
    apiMocks.getUserBalanceHistory.mockResolvedValue({
      items: [
        {
          id: 1,
          code: 'REG-001',
          type: 'registration',
          value: 50,
          status: 'used',
          used_by: 3,
          used_at: '2026-06-13T00:12:59Z',
          created_at: '2026-06-13T00:12:59Z',
          group_id: null,
          validity_days: 0,
          notes: '',
        },
      ],
      total: 1,
      page: 1,
      page_size: 15,
      pages: 1,
      total_recharged: 120.5,
      amount_sources: {
        recharge: 120.5,
        registration_bonus: 50,
        invitation_bonus: 18,
        checkin_bonus: 9.6,
        affiliate_transfer: 33.3,
        admin_adjustment: -5,
        total_credited: 226.4,
      },
    })
  })

  it('loads user context and renders signup, inviter, and amount sources', async () => {
    const wrapper = mount(UserBalanceHistoryModal, {
      props: {
        show: false,
        user: {
          id: 3,
          email: '3023208139@tiu.edu.cn',
          username: '',
          role: 'user',
          balance: 9092.08,
          concurrency: 1,
          status: 'active',
          allowed_groups: [],
          balance_notify_enabled: false,
          balance_notify_threshold: null,
          balance_notify_extra_emails: [],
          created_at: '2026-06-13T00:12:59Z',
          updated_at: '2026-06-13T00:12:59Z',
          notes: '',
        },
      },
      global: {
        stubs: {
          Select: {
            props: ['modelValue', 'options'],
            template: '<select />',
          },
          Icon: true,
        },
      },
    })

    await wrapper.setProps({ show: true })
    await flushPromises()

    expect(apiMocks.getById).toHaveBeenCalledWith(3)
    expect(apiMocks.getUserBalanceHistory).toHaveBeenCalledWith(3, 1, 15, undefined)

    const text = wrapper.text()
    expect(text).toContain('admin.users.signupSourceGoogle')
    expect(text).toContain('boss-user')
    expect(text).toContain('admin.users.amountSourceRecharge')
    expect(text).toContain('$120.50')
    expect(text).toContain('admin.users.amountSourceRegistration')
    expect(text).toContain('$50.00')
  })
})
