import { describe, expect, it } from 'vitest'
import { customNavigation, customRoutes } from '@/custom/registry'
import { supportTicketLocaleMessages } from '../locales'
import { supportTicketNavigation } from '../navigation'
import { supportTicketRoutes } from '../routes'

describe('support tickets module routes and navigation', () => {
  it('keeps authenticated user and administrator surfaces separate', () => {
    expect(supportTicketRoutes).toEqual(expect.arrayContaining([
      expect.objectContaining({ path: '/tickets', meta: expect.objectContaining({ requiresAuth: true, requiresAdmin: false, requiresFeature: 'support_ticket_enabled' }) }),
      expect.objectContaining({ path: '/tickets/:id', meta: expect.objectContaining({ requiresAuth: true, requiresAdmin: false }) }),
      expect.objectContaining({ path: '/admin/support/tickets', meta: expect.objectContaining({ requiresAuth: true, requiresAdmin: true, requiresFeature: 'support_ticket_enabled' }) }),
      expect.objectContaining({ path: '/admin/support/tickets/:id', meta: expect.objectContaining({ requiresAuth: true, requiresAdmin: true }) }),
    ]))
  })

  it('registers both sidebar entries through the custom overlay', () => {
    expect(supportTicketNavigation).toEqual(expect.arrayContaining([
      expect.objectContaining({ path: '/tickets', slot: 'after-transfer' }),
      expect.objectContaining({ path: '/admin/support/tickets', section: 'admin' }),
    ]))
    expect(customRoutes).toEqual(expect.arrayContaining(supportTicketRoutes))
    expect(customNavigation).toEqual(expect.arrayContaining(supportTicketNavigation))
  })

  it('provides bilingual labels for the backend category and safe-input guidance', () => {
    expect(supportTicketLocaleMessages.en.supportTickets).toMatchObject({ categories: { model: 'Model or channel issue' }, sensitiveNotice: expect.stringContaining('API keys') })
    expect(supportTicketLocaleMessages.zh.supportTickets).toMatchObject({ categories: { model: '模型或渠道问题' }, sensitiveNotice: expect.stringContaining('API Key') })
  })
})
