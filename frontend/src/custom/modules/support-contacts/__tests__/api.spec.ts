import { describe, expect, it, vi } from 'vitest'

const { get, post, put, remove } = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn(), remove: vi.fn() }))
vi.mock('@/api/client', () => ({ apiClient: { get, post, put, delete: remove } }))

import { supportContactsAPI } from '../api'

describe('supportContactsAPI', () => {
  it('keeps administration routes inside the custom namespace', () => {
    supportContactsAPI.listPublic()
    supportContactsAPI.listAdmin()
    supportContactsAPI.create({ channel: 'wechat', display_name: '', contact: 'support', link: '', qr_image_url: '', enabled: true, sort_order: 0 })
    supportContactsAPI.update('contact-id', { channel: 'qq', display_name: '', contact: '123', link: '', qr_image_url: '', enabled: true, sort_order: 10 })
    supportContactsAPI.remove('contact-id')

    expect(get).toHaveBeenNthCalledWith(1, '/custom/support-contacts')
    expect(get).toHaveBeenNthCalledWith(2, '/custom/admin/support-contacts')
    expect(post).toHaveBeenCalledWith('/custom/admin/support-contacts', expect.objectContaining({ channel: 'wechat' }))
    expect(put).toHaveBeenCalledWith('/custom/admin/support-contacts/contact-id', expect.objectContaining({ channel: 'qq' }))
    expect(remove).toHaveBeenCalledWith('/custom/admin/support-contacts/contact-id')
  })
})
