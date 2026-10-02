import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import RegistrationRiskView from '../RegistrationRiskAdminView.vue'

const { list, lock, unlock, showError } = vi.hoisted(() => ({
  list: vi.fn(),
  lock: vi.fn(),
  unlock: vi.fn(),
  showError: vi.fn(),
}))

vi.mock('../admin-api', () => ({
  default: {
    list,
    lock,
    unlock,
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess: vi.fn(),
  }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
      te: (key: string) => !key.endsWith('.future_signal'),
    }),
  }
})

function observation(userID = 44) {
  return {
    user_id: userID,
    email: 'l***@example.com',
    score: 2,
    action: 'observe',
    signals: ['exact_fingerprint_repeat'],
    evidence: [{ signal: 'exact_fingerprint_repeat', detail: 'fingerprint_repeat' }],
    related_accounts: [{ user_id: 43, email: 'duplicate@example.com' }],
    expires_at: '2026-09-24T00:00:00Z',
    payment_unlocked: false,
  }
}

function mountView() {
  return mount(RegistrationRiskView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        BaseDialog: { props: ['show'], template: '<div v-if="show"><slot /><slot name="footer" /></div>' },
        Icon: true,
      },
    },
  })
}

describe('admin RegistrationRiskView', () => {
  beforeEach(() => {
    list.mockReset()
    lock.mockReset()
    unlock.mockReset()
    showError.mockReset()
  })

  it('renders localized signals and descriptive evidence', async () => {
    list.mockResolvedValue({
      items: [{
        user_id: 42,
        email: 'u***@example.com',
        score: 5,
        action: 'high',
        signals: ['exact_fingerprint_repeat', 'future_signal'],
        evidence: [
          { signal: 'exact_fingerprint_repeat', detail: 'fingerprint_repeat' },
          { signal: 'future_signal', detail: 'unknown' },
        ],
        related_accounts: [{ user_id: 43, email: 'duplicate@example.com' }],
        expires_at: null,
        payment_unlocked: false,
      }],
    })

    const wrapper = mount(RegistrationRiskView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          BaseDialog: true,
          Icon: true,
        },
      },
    })
    await flushPromises()

    expect(wrapper.text()).toContain('admin.registrationRisk.signalLabels.exact_fingerprint_repeat')
    expect(wrapper.text()).toContain('future_signal')
    expect(wrapper.text()).toContain('admin.registrationRisk.evidenceDetails.fingerprint_repeat')
    expect(wrapper.text()).toContain('admin.registrationRisk.evidenceDetails.unknown')
    expect(wrapper.text()).toContain('duplicate@example.com')
  })

  it('supports older responses without evidence', async () => {
    list.mockResolvedValue({
      items: [{
        user_id: 43,
        email: 'o***@example.com',
        score: 2,
        action: 'medium',
        signals: [],
        expires_at: null,
        payment_unlocked: false,
      }],
    })

    const wrapper = mount(RegistrationRiskView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          BaseDialog: true,
          Icon: true,
        },
      },
    })
    await flushPromises()

    expect(wrapper.text()).toContain('admin.registrationRisk.noEvidence')
  })

  it('shows low-score signal records as observation and offers lock and allow actions', async () => {
    list.mockResolvedValue({
      items: [{
        user_id: 44,
        email: 'l***@example.com',
        score: 2,
        action: 'observe',
        signals: ['exact_fingerprint_repeat'],
        evidence: [{ signal: 'exact_fingerprint_repeat', detail: 'fingerprint_repeat' }],
        expires_at: '2026-09-24T00:00:00Z',
        payment_unlocked: false,
      }],
    })

    const wrapper = mount(RegistrationRiskView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          BaseDialog: true,
          Icon: true,
        },
      },
    })
    await flushPromises()

    expect(wrapper.text()).toContain('admin.registrationRisk.observe')
    expect(wrapper.text()).toContain('admin.registrationRisk.lock')
    expect(wrapper.text()).toContain('admin.registrationRisk.unlock')
  })

  it('paginates observations without duplicating rows when the list changes', async () => {
    list.mockResolvedValueOnce({ items: [observation(44), observation(43)], has_more: true })
      .mockResolvedValueOnce({ items: [observation(43), observation(42)], has_more: true })
      .mockResolvedValueOnce({ items: [], has_more: false })
    const wrapper = mountView()
    const more = () => wrapper.findAll('button').find(button => button.text() === 'admin.registrationRisk.loadMore')!
    await flushPromises()
    await more().trigger('click')
    await flushPromises()
    expect(list).toHaveBeenNthCalledWith(2, 100, 2)
    expect(wrapper.findAll('tbody tr')).toHaveLength(3)
    await more().trigger('click')
    await flushPromises()
    expect(list).toHaveBeenNthCalledWith(3, 100, 4)
    expect(more()).toBeUndefined()
  })

  it('preserves observations and retries the same page after a loading failure', async () => {
    list.mockResolvedValueOnce({ items: [observation()], has_more: true })
      .mockRejectedValueOnce(new Error('List unavailable'))
      .mockResolvedValueOnce({ items: [observation(43)], has_more: false })
    const wrapper = mountView()
    const more = () => wrapper.findAll('button').find(button => button.text() === 'admin.registrationRisk.loadMore')!
    await flushPromises()
    await more().trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('List unavailable')
    expect(wrapper.findAll('tbody tr')).toHaveLength(1)
    await more().trigger('click')
    await flushPromises()
    expect(list).toHaveBeenNthCalledWith(3, 100, 1)
    expect(wrapper.findAll('tbody tr')).toHaveLength(2)
    expect(wrapper.text()).not.toContain('List unavailable')
  })

  it('requires a reason and keeps a failed lock open for retry', async () => {
    list.mockResolvedValueOnce({ items: [observation()] })
      .mockResolvedValueOnce({ items: [{ ...observation(), action: 'manual_lock' }] })
    lock.mockRejectedValueOnce(new Error('Audit unavailable')).mockResolvedValueOnce({})
    const wrapper = mountView()
    await flushPromises()
    await wrapper.find('tbody tr').findAll('button')[0].trigger('click')
    const confirm = () => wrapper.findAll('button').find(button => button.text() === 'admin.registrationRisk.confirm')!
    await confirm().trigger('click')
    expect(lock).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('admin.registrationRisk.reasonRequired')
    await wrapper.find('textarea').setValue(' reviewed duplicate ')
    await confirm().trigger('click')
    await flushPromises()
    expect(showError).toHaveBeenCalledWith('Audit unavailable')
    expect(wrapper.find('textarea').exists()).toBe(true)
    expect(wrapper.find('tbody tr').text()).toContain('admin.registrationRisk.observe')
    await confirm().trigger('click')
    await flushPromises()
    expect(lock).toHaveBeenLastCalledWith(44, 'reviewed duplicate')
    expect(wrapper.find('textarea').exists()).toBe(false)
    expect(wrapper.find('tbody tr').text()).toContain('admin.registrationRisk.manualLock')
    expect(wrapper.find('tbody tr').findAll('button')).toHaveLength(1)
  })

  it('allows an observation directly without first locking the account', async () => {
    list.mockResolvedValueOnce({ items: [observation()] })
      .mockResolvedValueOnce({ items: [{ ...observation(), action: 'manual_unlock' }] })
    unlock.mockResolvedValue({})
    const wrapper = mountView()
    await flushPromises()
    await wrapper.find('tbody tr').findAll('button')[1].trigger('click')
    await wrapper.find('textarea').setValue('verified legitimate customer')
    await wrapper.findAll('button').find(button => button.text() === 'admin.registrationRisk.confirm')!.trigger('click')
    await flushPromises()
    expect(unlock).toHaveBeenCalledWith(44, 'verified legitimate customer')
    expect(lock).not.toHaveBeenCalled()
    expect(wrapper.find('tbody tr').text()).toContain('admin.registrationRisk.manualUnlock')
  })
})
