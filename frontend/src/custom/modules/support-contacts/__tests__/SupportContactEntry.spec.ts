import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SupportContactEntry from '../SupportContactEntry.vue'

const { listPublic } = vi.hoisted(() => ({ listPublic: vi.fn() }))
vi.mock('../api', () => ({ supportContactsAPI: { listPublic } }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showSuccess: vi.fn(), showError: vi.fn() }) }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

const BaseDialog = { props: ['show', 'title'], emits: ['close'], template: '<div v-if="show"><slot /></div>' }
const Icon = { template: '<i />' }

describe('SupportContactEntry', () => {
  beforeEach(() => listPublic.mockReset())

  it('falls back to the legacy contact text when no Overlay contacts exist', async () => {
    listPublic.mockResolvedValue({ data: [] })
    const wrapper = mount(SupportContactEntry, { props: { placement: 'profile', legacyContactInfo: 'QQ: 123' }, global: { stubs: { BaseDialog, Icon } } })
    await flushPromises()
    expect(wrapper.text()).toContain('QQ: 123')
  })

  it('shows the shared support entry when contacts exist', async () => {
    listPublic.mockResolvedValue({ data: [{ id: '1', channel: 'wechat', display_name: '客服', contact: 'service-id', link: '', qr_image_url: '', has_qr_image: false, enabled: true, sort_order: 0 }] })
    const wrapper = mount(SupportContactEntry, { props: { placement: 'redeem', legacyContactInfo: 'QQ: 123' }, global: { stubs: { BaseDialog, Icon } } })
    await flushPromises()
    expect(wrapper.text()).toContain('common.contactSupport')
    expect(wrapper.text()).not.toContain('QQ: 123')
  })

  it('always identifies a contact channel even when a display name is configured', async () => {
    listPublic.mockResolvedValue({ data: [{ id: '1', channel: 'wechat', display_name: 'why', contact: '_-why-_', link: '', qr_image_url: '', has_qr_image: false, enabled: true, sort_order: 0 }] })
    const wrapper = mount(SupportContactEntry, { props: { placement: 'profile', legacyContactInfo: '' }, global: { stubs: { BaseDialog, Icon } } })
    await flushPromises()
    await wrapper.get('button').trigger('click')
    expect(wrapper.text()).toContain('supportContacts.channels.wechat')
    expect(wrapper.text()).toContain('why')
  })

  it('keeps the docs support button visible and explains when contacts are missing', async () => {
    listPublic.mockResolvedValue({ data: [] })
    const wrapper = mount(SupportContactEntry, { props: { placement: 'docs', legacyContactInfo: '' }, global: { stubs: { BaseDialog, Icon } } })
    await flushPromises()
    const button = wrapper.get('button')
    expect(button.text()).toContain('common.contactSupport')
    await button.trigger('click')
    expect(wrapper.text()).toContain('supportContacts.notConfigured')
  })

  it('opens configured contacts from the docs support button', async () => {
    listPublic.mockResolvedValue({ data: [{ id: '1', channel: 'email', display_name: '客服', contact: 'support@example.com', link: '', qr_image_url: '', has_qr_image: false, enabled: true, sort_order: 0 }] })
    const wrapper = mount(SupportContactEntry, { props: { placement: 'docs', legacyContactInfo: '' }, global: { stubs: { BaseDialog, Icon } } })
    await flushPromises()
    await wrapper.get('button').trigger('click')
    expect(wrapper.text()).toContain('supportContacts.channels.email')
    expect(wrapper.text()).toContain('support@example.com')
  })
})
