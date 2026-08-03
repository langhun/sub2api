import { describe, expect, it } from 'vitest'
import { ticketCategories, ticketPriorityClass, ticketStatusClass, ticketStatusLocaleKey } from '../presentation'
import { supportTicketAdminLocaleMessages, supportTicketLocaleMessages } from '../locales'

describe('support ticket presentation', () => {
  it('covers the complete backend category enum and visually distinguishes work states', () => {
    expect(ticketCategories).toContain('model')
    expect(ticketStatusClass('open')).toBe('badge-warning')
    expect(ticketStatusClass('waiting_user')).toBe('badge-primary')
    expect(ticketStatusClass('resolved')).toBe('badge-success')
    expect(ticketPriorityClass('urgent')).toBe('badge-danger')
  })

  it('keeps user and admin workflow labels explicit to the current viewer', () => {
    expect(ticketStatusLocaleKey('waiting_user', 'user')).toBe('supportTickets.statuses.waiting_user')
    expect(ticketStatusLocaleKey('waiting_user', 'admin')).toBe('admin.supportTickets.statuses.waiting_user')
    expect(supportTicketLocaleMessages.zh.supportTickets.statuses).toMatchObject({
      open: '等待客服回复',
      waiting_user: '等待你回复',
    })
    expect(supportTicketAdminLocaleMessages.zh.supportTickets.statuses).toMatchObject({
      open: '等待客服处理',
      waiting_user: '等待客户回复',
    })
  })
})
