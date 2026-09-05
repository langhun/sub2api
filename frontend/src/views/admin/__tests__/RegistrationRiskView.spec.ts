import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import RegistrationRiskView from '../RegistrationRiskView.vue'

const { list } = vi.hoisted(() => ({
  list: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    registrationRisk: {
      list,
      lock: vi.fn(),
      unlock: vi.fn(),
    },
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
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

describe('admin RegistrationRiskView', () => {
  beforeEach(() => {
    list.mockReset()
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
})
