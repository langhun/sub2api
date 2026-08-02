import { describe, expect, it } from 'vitest'
import { ticketCategories, ticketPriorityClass, ticketStatusClass } from '../presentation'

describe('support ticket presentation', () => {
  it('covers the complete backend category enum and visually distinguishes work states', () => {
    expect(ticketCategories).toContain('model')
    expect(ticketStatusClass('open')).toBe('badge-warning')
    expect(ticketStatusClass('waiting_user')).toBe('badge-primary')
    expect(ticketStatusClass('resolved')).toBe('badge-success')
    expect(ticketPriorityClass('urgent')).toBe('badge-danger')
  })
})
