import { describe, expect, it, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import TempUnschedStatusModal from '../TempUnschedStatusModal.vue'
import type { Account } from '@/types'

const {
  getTempUnschedulableStatusMock,
  showErrorMock,
  showSuccessMock,
} = vi.hoisted(() => ({
  getTempUnschedulableStatusMock: vi.fn(),
  showErrorMock: vi.fn(),
  showSuccessMock: vi.fn(),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      getTempUnschedulableStatus: getTempUnschedulableStatusMock,
      recoverState: vi.fn()
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: showErrorMock,
    showSuccess: showSuccessMock
  })
}))

const BaseDialogStub = defineComponent({
  name: 'BaseDialogStub',
  props: {
    show: Boolean,
    title: String
  },
  setup(props, { slots }) {
    return () =>
      h('div', { class: 'base-dialog-stub' }, [
        h('div', { class: 'dialog-title' }, props.title || ''),
        slots.default?.(),
        slots.footer?.()
      ])
  }
})

const flushPromises = async () => {
  await Promise.resolve()
  await Promise.resolve()
}

function makeAccount(overrides: Partial<Account> = {}): Account {
  return {
    id: 1,
    name: 'oauth-1',
    platform: 'antigravity',
    type: 'oauth',
    proxy_id: null,
    concurrency: 1,
    priority: 1,
    status: 'active',
    error_message: null,
    last_used_at: null,
    expires_at: null,
    auto_pause_on_expired: false,
    created_at: '2026-03-15T00:00:00Z',
    updated_at: '2026-03-15T00:00:00Z',
    schedulable: true,
    rate_limited_at: null,
    rate_limit_reset_at: null,
    overload_until: null,
    temp_unschedulable_until: null,
    temp_unschedulable_reason: null,
    session_window_start: null,
    session_window_end: null,
    session_window_status: null,
    ...overrides
  }
}

describe('TempUnschedStatusModal', () => {
  beforeEach(() => {
    getTempUnschedulableStatusMock.mockReset()
    showErrorMock.mockReset()
    showSuccessMock.mockReset()
  })

  it('把 401 临时冷却识别结果和自动策略显示出来', async () => {
    const nowUnix = Math.floor(Date.now() / 1000)
    getTempUnschedulableStatusMock.mockResolvedValue({
      active: true,
      state: {
        until_unix: nowUnix + 900,
        triggered_at_unix: nowUnix - 60,
        status_code: 401,
        matched_keyword: 'oauth_401',
        rule_index: -1,
        error_message: 'OAuth 401: invalid or expired credentials'
      }
    })

    const wrapper = mount(TempUnschedStatusModal, {
      props: {
        show: false,
        account: makeAccount()
      },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub
        }
      }
    })

    await wrapper.setProps({ show: true })
    await flushPromises()

    expect(wrapper.text()).toContain('401')
    expect(wrapper.text()).toContain('oauth_401')
    expect(wrapper.text()).toContain('admin.accounts.tempUnschedulable.autoRule')
  })
})
