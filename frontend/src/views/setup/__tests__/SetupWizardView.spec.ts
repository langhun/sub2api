import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import SetupWizardView from '../SetupWizardView.vue'

const api = vi.hoisted(() => ({
  testDatabase: vi.fn(),
  testRedis: vi.fn(),
  install: vi.fn(),
}))

vi.mock('@/api/setup', () => api)
vi.mock('@/api/client', () => ({ buildGatewayUrl: (path: string) => path }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

let wrapper: VueWrapper | undefined

function renderWizard() {
  wrapper = mount(SetupWizardView, {
    global: { stubs: { Icon: true, Select: true, Toggle: true } },
  })
  return wrapper
}

function button(text: string) {
  return wrapper!.findAll('button').find((item) => item.text().includes(text))!
}

async function advanceToInstall() {
  await button('setup.status.testConnection').trigger('click')
  await flushPromises()
  await button('common.next').trigger('click')
  await button('setup.status.testConnection').trigger('click')
  await flushPromises()
  await button('common.next').trigger('click')
  const inputs = wrapper!.findAll('input')
  await inputs[0].setValue('admin@example.com')
  await inputs[1].setValue('password123')
  await inputs[2].setValue('password123')
  await button('common.next').trigger('click')
  await button('setup.status.completeInstallation').trigger('click')
  await flushPromises()
}

describe('SetupWizardView state lifecycle', () => {
  beforeEach(() => {
    api.testDatabase.mockReset().mockResolvedValue(undefined)
    api.testRedis.mockReset().mockResolvedValue(undefined)
    api.install.mockReset().mockResolvedValue({ restart: true })
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false }))
  })

  afterEach(() => {
    wrapper?.unmount()
    wrapper = undefined
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  it('only shows Redis TLS settings in the Redis step', async () => {
    renderWizard()
    expect(wrapper!.text()).not.toContain('setup.redis.enableTls')
    await button('setup.status.testConnection').trigger('click')
    await flushPromises()
    await button('common.next').trigger('click')
    expect(wrapper!.text()).toContain('setup.redis.enableTls')
  })

  it('invalidates a successful connection test when its settings change', async () => {
    renderWizard()
    await button('setup.status.testConnection').trigger('click')
    await flushPromises()
    expect(button('common.next').attributes('disabled')).toBeUndefined()
    await wrapper!.find('input').setValue('another-database')
    expect(button('common.next').attributes('disabled')).toBeDefined()
  })

  it('does not accept a stale connection result for changed settings', async () => {
    let finishTest!: () => void
    api.testDatabase.mockReturnValue(new Promise<void>((resolve) => { finishTest = resolve }))
    renderWizard()
    await button('setup.status.testConnection').trigger('click')
    await wrapper!.find('input').setValue('another-database')
    finishTest()
    await flushPromises()
    expect(button('common.next').attributes('disabled')).toBeDefined()
  })

  it('cancels the service restart wait when leaving the wizard', async () => {
    vi.useFakeTimers()
    renderWizard()
    await advanceToInstall()
    wrapper!.unmount()
    wrapper = undefined
    await vi.advanceTimersByTimeAsync(10000)
    expect(fetch).not.toHaveBeenCalled()
  })

  it('aborts an in-flight restart status request on unmount', async () => {
    vi.useFakeTimers()
    vi.mocked(fetch).mockReturnValue(new Promise(() => {}))
    renderWizard()
    await advanceToInstall()
    await vi.advanceTimersByTimeAsync(3000)
    const signal = vi.mocked(fetch).mock.calls[0][1]!.signal as AbortSignal
    expect(signal.aborted).toBe(false)
    wrapper!.unmount()
    wrapper = undefined
    expect(signal.aborted).toBe(true)
  })
})
