import { flushPromises, mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { defineComponent } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import CheckinCaptchaGate from '../components/CheckinCaptchaGate.vue'

const appStore = vi.hoisted((): {
  cachedPublicSettings: Record<string, unknown> | null
  publicSettingsLoading: boolean
  fetchPublicSettings: ReturnType<typeof vi.fn>
} => ({
  cachedPublicSettings: {
    turnstile_enabled: true,
    turnstile_site_key: 'site-key',
  } as Record<string, unknown>,
  publicSettingsLoading: false,
  fetchPublicSettings: vi.fn(),
}))

vi.mock('@/stores/app', () => ({ useAppStore: () => appStore }))

const reset = vi.fn()
const CaptchaChallengeStub = defineComponent({
  props: {
    action: String,
    turnstileEnabled: Boolean,
    turnstileSiteKey: String,
  },
  emits: ['verify', 'expire', 'error'],
  setup(_, { expose }) {
    expose({ reset, verifyAction: vi.fn() })
    return {}
  },
  template: '<button data-testid="captcha-verify" @click="$emit(\'verify\', \'proof-token\', \'\')">verify</button>',
})

function mountGate() {
  return mount(CheckinCaptchaGate, {
    props: { active: true },
    global: {
      plugins: [createI18n({ legacy: false, locale: 'en', missingWarn: false, fallbackWarn: false, messages: { en: {} } })],
      stubs: { CaptchaChallenge: CaptchaChallengeStub },
    },
  })
}

describe('CheckinCaptchaGate', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    appStore.cachedPublicSettings = {
      turnstile_enabled: true,
      turnstile_site_key: 'site-key',
    }
    appStore.publicSettingsLoading = false
    appStore.fetchPublicSettings.mockResolvedValue(appStore.cachedPublicSettings)
  })

  it('binds Turnstile to the checkin action and returns a fresh proof', async () => {
    const wrapper = mountGate()
    await flushPromises()
    const challenge = wrapper.getComponent(CaptchaChallengeStub)

    expect(challenge.props('action')).toBe('checkin')
    expect(wrapper.emitted('readyChange')?.at(-1)).toEqual([false])

    await wrapper.get('[data-testid="captcha-verify"]').trigger('click')
    expect(wrapper.emitted('readyChange')?.at(-1)).toEqual([true])
    await expect((wrapper.vm as unknown as { resolveProof: () => Promise<unknown> }).resolveProof()).resolves.toEqual({
      turnstile_token: 'proof-token',
    })
  })

  it('does not silently bypass an enabled provider with missing public configuration', async () => {
    appStore.cachedPublicSettings = {
      turnstile_enabled: true,
      turnstile_site_key: '',
    }
    const wrapper = mountGate()
    await flushPromises()

    expect(wrapper.get('[role="alert"]').text()).toContain('checkin.verificationUnavailable')
    expect(wrapper.emitted('readyChange')?.at(-1)).toEqual([false])
  })

  it('waits for public settings before allowing a check-in', async () => {
    appStore.cachedPublicSettings = null
    const wrapper = mountGate()

    expect(wrapper.emitted('readyChange')?.at(-1)).toEqual([false])
    await flushPromises()
    expect(appStore.fetchPublicSettings).toHaveBeenCalledWith(true)
  })

  it('shows retry when loading public settings fails', async () => {
    appStore.cachedPublicSettings = null
    appStore.fetchPublicSettings.mockResolvedValueOnce(null)
    const wrapper = mountGate()

    await vi.waitFor(() => expect(wrapper.get('[role="status"]').text()).toContain('checkin.verificationSettingsFailed'))
    const retry = wrapper.get('button')
    expect(retry.text()).toContain('common.retry')
    await retry.trigger('click')
    expect(appStore.fetchPublicSettings).toHaveBeenLastCalledWith(true)
  })

  it('replaces stale disabled settings before allowing submission', async () => {
    appStore.cachedPublicSettings = {
      turnstile_enabled: false,
      turnstile_site_key: '',
    }
    appStore.fetchPublicSettings.mockImplementationOnce(async () => {
      appStore.cachedPublicSettings = {
        turnstile_enabled: true,
        turnstile_site_key: 'fresh-site-key',
      }
      return appStore.cachedPublicSettings
    })

    const wrapper = mountGate()
    expect(wrapper.emitted('readyChange')?.at(-1)).toEqual([false])
    await flushPromises()

    expect(appStore.fetchPublicSettings).toHaveBeenCalledWith(true)
    expect(wrapper.getComponent(CaptchaChallengeStub).props('turnstileEnabled')).toBe(true)
    expect(wrapper.emitted('readyChange')?.at(-1)).toEqual([false])
  })
})
