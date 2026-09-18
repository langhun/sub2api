import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import BaseDialog from '../BaseDialog.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key })
}))

describe('BaseDialog', () => {
  afterEach(() => {
    document.body.innerHTML = ''
    document.body.classList.remove('modal-open')
  })

  it('resets body scroll position when reopened', async () => {
    const wrapper = mount(BaseDialog, {
      attachTo: document.body,
      props: { show: false, title: 'Details' },
      slots: { default: '<div style="height: 2000px">content</div>' },
      global: { stubs: { Icon: true } }
    })

    await wrapper.setProps({ show: true })
    await nextTick()
    const body = document.body.querySelector<HTMLElement>('.modal-body')
    expect(body).not.toBeNull()
    body!.scrollTop = 480

    await wrapper.setProps({ show: false })
    await wrapper.setProps({ show: true })
    await nextTick()

    expect(document.body.querySelector<HTMLElement>('.modal-body')?.scrollTop).toBe(0)
    wrapper.unmount()
  })

  it('keeps the body locked while another dialog is open and Escape closes only the top dialog', async () => {
    const first = mount(BaseDialog, { attachTo: document.body, props: { show: true, title: 'First' } })
    const second = mount(BaseDialog, { attachTo: document.body, props: { show: true, title: 'Second' } })
    const hidden = mount(BaseDialog, { props: { show: false, title: 'Hidden' } })
    await nextTick()
    const ids = [...document.querySelectorAll('[role="dialog"]')].map((el) => el.getAttribute('aria-labelledby'))
    expect(new Set(ids).size).toBe(2)
    expect(document.body.classList.contains('modal-open')).toBe(true)
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', cancelable: true }))
    expect(first.emitted('close')).toBeUndefined()
    expect(second.emitted('close')).toHaveLength(1)
    hidden.unmount()
    await second.setProps({ show: false })
    expect(document.body.classList.contains('modal-open')).toBe(true)
    first.unmount()
    second.unmount()
    expect(document.body.classList.contains('modal-open')).toBe(false)
  })

  it('cycles keyboard focus and restores the opener after closing', async () => {
    const opener = document.createElement('button')
    document.body.appendChild(opener)
    opener.focus()
    const wrapper = mount(BaseDialog, {
      attachTo: document.body,
      props: { show: true, title: 'Focus', showCloseButton: false },
      slots: { default: '<button disabled>Disabled</button><button id="first">First</button><button id="last">Last</button>' },
    })
    await nextTick()
    const first = document.getElementById('first')!
    const last = document.getElementById('last')!
    expect(document.activeElement).toBe(first)
    first.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', shiftKey: true, bubbles: true, cancelable: true }))
    expect(document.activeElement).toBe(last)
    last.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', bubbles: true, cancelable: true }))
    expect(document.activeElement).toBe(first)
    await wrapper.setProps({ show: false })
    expect(document.activeElement).toBe(opener)
    wrapper.unmount()
  })

  it('preserves the original opener when a lower dialog unmounts before its child', async () => {
    const opener = document.createElement('button')
    document.body.appendChild(opener)
    opener.focus()
    const first = mount(BaseDialog, { attachTo: document.body, props: { show: true, title: 'First' } })
    await nextTick()
    const second = mount(BaseDialog, { attachTo: document.body, props: { show: true, title: 'Second' } })
    await nextTick()
    first.unmount()
    expect(document.body.classList.contains('modal-open')).toBe(true)
    second.unmount()
    expect(document.activeElement).toBe(opener)
    expect(document.body.classList.contains('modal-open')).toBe(false)
  })

  it('skips controls hidden by ancestors and brings escaped focus back into the dialog', async () => {
    const outside = document.createElement('button')
    document.body.appendChild(outside)
    const wrapper = mount(BaseDialog, {
      attachTo: document.body,
      props: { show: true, title: 'Focus', showCloseButton: false },
      slots: { default: '<div style="display:none"><button>Hidden</button></div><button id="visible-control">Visible</button>' },
    })
    await nextTick()
    const visible = document.getElementById('visible-control')!
    expect(document.activeElement).toBe(visible)
    outside.focus()
    outside.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', bubbles: true, cancelable: true }))
    expect(document.activeElement).toBe(visible)
    wrapper.unmount()
  })

  it('respects dynamic z-index and never closes a lower dialog when top Escape is disabled', async () => {
    const first = mount(BaseDialog, { attachTo: document.body, props: { show: true, title: 'First' } })
    const second = mount(BaseDialog, { attachTo: document.body, props: { show: true, title: 'Second' } })
    await first.setProps({ zIndex: 80, closeOnEscape: false })
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', cancelable: true }))
    expect(first.emitted('close')).toBeUndefined()
    expect(second.emitted('close')).toBeUndefined()
    await first.setProps({ closeOnEscape: true })
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', cancelable: true }))
    expect(first.emitted('close')).toHaveLength(1)
    expect(second.emitted('close')).toBeUndefined()
    second.unmount()
    first.unmount()
  })
})
