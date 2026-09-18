import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import ProfileBalanceNotifyCard from '../ProfileBalanceNotifyCard.vue'

const { sendNotifyEmailCode, verifyNotifyEmail, getProfile, updateProfile, toggleNotifyEmail, removeNotifyEmail } = vi.hoisted(() => ({
  sendNotifyEmailCode: vi.fn(),
  verifyNotifyEmail: vi.fn(),
  getProfile: vi.fn(),
  updateProfile: vi.fn(),
  toggleNotifyEmail: vi.fn(),
  removeNotifyEmail: vi.fn()
}))

vi.mock('@/api', () => ({
  userAPI: { sendNotifyEmailCode, verifyNotifyEmail, getProfile, updateProfile, toggleNotifyEmail, removeNotifyEmail }
}))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => ({ user: null }) }))
vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showSuccess: vi.fn(), showError: vi.fn() })
}))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

enableAutoUnmount(afterEach)

const deferred = () => {
  let resolve!: () => void
  const promise = new Promise<void>((done) => { resolve = done })
  return { promise, resolve }
}

const pendingRows = (wrapper: VueWrapper) => wrapper.findAll('.bg-yellow-50')
const pendingEmails = (wrapper: VueWrapper) => pendingRows(wrapper).map(row => row.get('span').text())

const button = (wrapper: VueWrapper, text: string) =>
  wrapper.findAll('button').find(item => item.text() === text)!

describe('ProfileBalanceNotifyCard', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.resetAllMocks()
    sendNotifyEmailCode.mockResolvedValue({})
    getProfile.mockResolvedValue({ balance_notify_extra_emails: [] })
    updateProfile.mockResolvedValue({})
    toggleNotifyEmail.mockResolvedValue({ balance_notify_extra_emails: [] })
    removeNotifyEmail.mockResolvedValue({})
  })

  afterEach(() => {
    vi.clearAllTimers()
    vi.useRealTimers()
  })

  it.each([0, 1])('removes only verified emails when request %i finishes first', async (first) => {
    const emails = ['first@example.com', 'second@example.com', 'third@example.com']
    const requests = [deferred(), deferred()]
    verifyNotifyEmail.mockImplementation((email: string) => requests[emails.indexOf(email)]!.promise)
    const wrapper = mount(ProfileBalanceNotifyCard, {
      props: { enabled: true, threshold: null, extraEmails: [], systemDefaultThreshold: 5, userEmail: '' }
    })

    for (const email of emails) {
      await wrapper.get('input[type="email"]').setValue(email)
      await button(wrapper, 'common.add').trigger('click')
    }
    for (const row of pendingRows(wrapper)) {
      await row.findAll('button').find(item => item.text() === 'profile.balanceNotify.sendCode')!.trigger('click')
    }
    await flushPromises()
    for (const row of pendingRows(wrapper)) {
      await row.get('input').setValue('123456')
    }
    for (const row of pendingRows(wrapper).slice(0, 2)) {
      await row.findAll('button').find(item => item.text() === 'profile.balanceNotify.verify')!.trigger('click')
    }
    expect(verifyNotifyEmail.mock.calls).toEqual(emails.slice(0, 2).map(email => [email, '123456']))

    requests[first]!.resolve()
    await flushPromises()
    expect(pendingEmails(wrapper)).toEqual(emails.filter((_, index) => index !== first))

    requests[1 - first]!.resolve()
    await flushPromises()
    expect(pendingEmails(wrapper)).toEqual([emails[2]])
    expect((pendingRows(wrapper)[0]!.get('input').element as HTMLInputElement).value).toBe('123456')
    expect(getProfile).toHaveBeenCalledTimes(2)
  })

  it('exposes accessible switch semantics and keeps saved email rows mobile-safe', () => {
    const wrapper = mount(ProfileBalanceNotifyCard, {
      props: {
        enabled: true,
        threshold: null,
        extraEmails: [{ email: 'long-address@example.com', disabled: false, verified: true }],
        systemDefaultThreshold: 5,
        userEmail: ''
      }
    })

    const mainSwitch = wrapper.get('input[role="switch"]')
    expect(mainSwitch.attributes('aria-checked')).toBe('true')
    expect(mainSwitch.attributes('aria-label')).toBe('profile.balanceNotify.enabled')
    expect(wrapper.get('[data-glass-switch-track]').attributes('data-checked')).toBe('true')

    const row = wrapper.get('[data-testid="notify-saved-email"]')
    expect(row.classes()).toContain('min-w-0')
    expect(row.classes()).toContain('flex-col')
    expect(row.classes()).toContain('sm:flex-row')
    expect(row.get('span').classes()).toContain('break-all')
  })

  it('rejects malformed addresses before creating a pending verification row', async () => {
    const wrapper = mount(ProfileBalanceNotifyCard, {
      props: { enabled: true, threshold: null, extraEmails: [], systemDefaultThreshold: 5, userEmail: '' }
    })

    await wrapper.get('input[type="email"]').setValue('not-an-email')
    await button(wrapper, 'common.add').trigger('click')

    expect(pendingRows(wrapper)).toHaveLength(0)
    expect(wrapper.get('input[type="email"]').element.value).toBe('not-an-email')
    expect(sendNotifyEmailCode).not.toHaveBeenCalled()
  })

  it('removes a pending email and clears its countdown timer', async () => {
    const wrapper = mount(ProfileBalanceNotifyCard, {
      props: { enabled: true, threshold: null, extraEmails: [], systemDefaultThreshold: 5, userEmail: '' }
    })

    await wrapper.get('input[type="email"]').setValue('pending@example.com')
    await button(wrapper, 'common.add').trigger('click')
    const row = pendingRows(wrapper)[0]!
    await row.findAll('button').find(item => item.text() === 'profile.balanceNotify.sendCode')!.trigger('click')
    await flushPromises()
    expect(sendNotifyEmailCode).toHaveBeenCalledWith('pending@example.com')
    expect(row.find('span').text()).toContain('pending@example.com')

    await row.findAll('button').find(item => item.text() === 'profile.balanceNotify.removeEmail')!.trigger('click')
    expect(pendingRows(wrapper)).toHaveLength(0)
    await vi.advanceTimersByTimeAsync(61_000)
    expect(sendNotifyEmailCode).toHaveBeenCalledTimes(1)
  })
})
