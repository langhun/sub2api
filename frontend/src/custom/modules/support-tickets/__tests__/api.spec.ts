import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, patch, post } = vi.hoisted(() => ({ get: vi.fn(), patch: vi.fn(), post: vi.fn() }))

vi.mock('@/api/client', () => ({ apiClient: { get, patch, post } }))

import {
  addAdminSupportTicketMessage,
  addSupportTicketMessage,
  createSupportTicket,
  getAdminSupportTicket,
  getSupportTicket,
  listAdminSupportTickets,
  listSupportTickets,
  reopenSupportTicket,
  updateAdminSupportTicket,
} from '../api'

describe('support tickets API', () => {
  beforeEach(() => {
    get.mockReset()
    post.mockReset()
    patch.mockReset()
    get.mockResolvedValue({ data: {} })
    post.mockResolvedValue({ data: {} })
    patch.mockResolvedValue({ data: {} })
  })

  it('uses the user paths and opaque public ticket identifier', async () => {
    await listSupportTickets({ page: 2, page_size: 25 })
    await createSupportTicket({ subject: 'Request failed', category: 'api', body: 'Request ID: req_123' })
    await getSupportTicket('SUP-0123456789ABCDEF')
    await addSupportTicketMessage('SUP-0123456789ABCDEF', { body: 'More detail' })
    await reopenSupportTicket('SUP-0123456789ABCDEF')

    expect(get).toHaveBeenNthCalledWith(1, '/support/tickets', { params: { page: 2, page_size: 25 } })
    expect(post).toHaveBeenNthCalledWith(1, '/support/tickets', { subject: 'Request failed', category: 'api', body: 'Request ID: req_123' })
    expect(get).toHaveBeenNthCalledWith(2, '/support/tickets/SUP-0123456789ABCDEF')
    expect(post).toHaveBeenNthCalledWith(2, '/support/tickets/SUP-0123456789ABCDEF/messages', { body: 'More detail' })
    expect(post).toHaveBeenNthCalledWith(3, '/support/tickets/SUP-0123456789ABCDEF/reopen')
  })

  it('uses the administrator paths for queue management and internal notes', async () => {
    await listAdminSupportTickets({ status: 'open', priority: 'urgent' })
    await getAdminSupportTicket('SUP-0123456789ABCDEF')
    await addAdminSupportTicketMessage('SUP-0123456789ABCDEF', { body: 'Investigating', internal: true })
    await updateAdminSupportTicket('SUP-0123456789ABCDEF', { status: 'waiting_user' })

    expect(get).toHaveBeenNthCalledWith(1, '/admin/support/tickets', { params: { page: 1, page_size: 20, status: 'open', priority: 'urgent' } })
    expect(get).toHaveBeenNthCalledWith(2, '/admin/support/tickets/SUP-0123456789ABCDEF')
    expect(post).toHaveBeenCalledWith('/admin/support/tickets/SUP-0123456789ABCDEF/messages', { body: 'Investigating', internal: true })
    expect(patch).toHaveBeenCalledWith('/admin/support/tickets/SUP-0123456789ABCDEF', { status: 'waiting_user' })
  })
})
