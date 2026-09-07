import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import TurnstileWidget from '../TurnstileWidget.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key })
}))

describe('TurnstileWidget action binding', () => {
  afterEach(() => {
    delete window.turnstile
  })

  it('passes the endpoint action to Cloudflare render options', async () => {
    const render = vi.fn(() => 'widget-id')
    window.turnstile = {
      render,
      reset: vi.fn(),
      remove: vi.fn(),
    }

    const wrapper = mount(TurnstileWidget, {
      props: { siteKey: 'site-key', action: 'checkin' },
    })
    await flushPromises()

    expect(render).toHaveBeenCalledOnce()
    expect(render.mock.calls[0]?.[1]).toMatchObject({
      sitekey: 'site-key',
      action: 'checkin',
    })
    wrapper.unmount()
  })
})
