import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import type { SupportTicket, SupportTicketStatus } from '../api'

const mocks = vi.hoisted(() => ({
  addMessage: vi.fn(),
  createTicket: vi.fn(),
  getTicket: vi.fn(),
  listTickets: vi.fn(),
  updateTicket: vi.fn(),
  searchUsers: vi.fn(),
  routerPush: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
}))

const routeState = vi.hoisted(() => ({
  params: { id: 'SUP-0123456789ABCDEF' } as Record<string, string>,
}))

vi.mock('../api', () => ({
  addAdminSupportTicketMessage: mocks.addMessage,
  createAdminSupportTicket: mocks.createTicket,
  getAdminSupportTicket: mocks.getTicket,
  listAdminSupportTickets: mocks.listTickets,
  updateAdminSupportTicket: mocks.updateTicket,
}))

vi.mock('@/api/admin/usage', () => ({
  searchUsers: mocks.searchUsers,
  default: { searchUsers: mocks.searchUsers },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError: mocks.showError, showSuccess: mocks.showSuccess }),
}))

vi.mock('vue-router', async () => {
  const actual = await vi.importActual<typeof import('vue-router')>('vue-router')
  return {
    ...actual,
    useRoute: () => routeState,
    useRouter: () => ({ push: mocks.routerPush }),
  }
})

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

import AdminSupportTicketsView from '../admin/AdminSupportTicketsView.vue'

function ticket(overrides: Partial<SupportTicket> = {}): SupportTicket {
  return {
    public_id: 'SUP-0123456789ABCDEF',
    user_id: 42,
    user_name: 'Alice',
    user_email: 'alice@example.com',
    category: 'account',
    subject: 'Account notice',
    status: 'open',
    priority: 'normal',
    last_reply_by: 'user',
    last_activity_at: '2026-08-06T08:00:00Z',
    created_at: '2026-08-06T07:00:00Z',
    updated_at: '2026-08-06T08:00:00Z',
    messages: [{ id: 1, author_type: 'user', message_type: 'reply', body: 'Please help', created_at: '2026-08-06T08:00:00Z' }],
    ...overrides,
  }
}

function mountView(value: SupportTicket) {
  routeState.params = { id: value.public_id }
  mocks.getTicket.mockResolvedValue(value)

  return mount(AdminSupportTicketsView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        BaseDialog: { template: '<div />' },
        Icon: true,
        LoadingSpinner: true,
      },
    },
  })
}

async function submitReply(wrapper: ReturnType<typeof mount>, body: string) {
  await wrapper.get('#admin-ticket-reply').setValue(body)
  await wrapper.findAll('form')[0].trigger('submit')
  await flushPromises()
}

async function selectInternalNoteMode(wrapper: ReturnType<typeof mount>) {
  await wrapper.findAll('[role="group"] button')[1].trigger('click')
}

describe('AdminSupportTicketsView', () => {
  beforeEach(() => {
    Object.values(mocks).forEach((mock) => mock.mockReset())
    mocks.addMessage.mockResolvedValue({})
    mocks.createTicket.mockResolvedValue({})
    mocks.listTickets.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 20 })
    mocks.updateTicket.mockResolvedValue({})
    mocks.searchUsers.mockResolvedValue([])
  })

  it.each([
    ['username', { user_name: 'Alice', user_email: 'alice@example.com' }, '#42 · Alice'],
    ['email', { user_name: '   ', user_email: 'alice@example.com' }, '#42 · alice@example.com'],
    ['id', { user_name: '   ', user_email: '   ' }, '#42'],
  ])('renders the requester using the %s fallback', async (_case, identity, expected) => {
    const wrapper = mountView(ticket(identity))
    await flushPromises()

    expect(wrapper.text()).toContain(expected)
    wrapper.unmount()
  })

  it('switches between external replies and internal notes without mixing payloads', async () => {
    const wrapper = mountView(ticket())
    await flushPromises()

    expect(wrapper.get('#admin-ticket-reply').attributes('placeholder')).toBe('supportTickets.replyPlaceholder')
    await submitReply(wrapper, 'Visible reply')
    expect(mocks.addMessage).toHaveBeenLastCalledWith('SUP-0123456789ABCDEF', { body: 'Visible reply', internal: false })

    await selectInternalNoteMode(wrapper)
    expect(wrapper.get('#admin-ticket-reply').attributes('placeholder')).toBe('admin.supportTickets.internalNotePlaceholder')
    await submitReply(wrapper, 'Private note')
    expect(mocks.addMessage).toHaveBeenLastCalledWith('SUP-0123456789ABCDEF', { body: 'Private note', internal: true })

    wrapper.unmount()
  })

  it.each([
    ['resolved', 'admin.supportTickets.replyRequiresReopen'],
    ['closed', 'admin.supportTickets.replyClosed'],
  ] satisfies Array<[SupportTicketStatus, string]>)('blocks external replies for %s tickets but still permits internal notes', async (status, hint) => {
    const wrapper = mountView(ticket({ status }))
    await flushPromises()

    expect(wrapper.get('#admin-ticket-reply').attributes()).toHaveProperty('disabled')
    expect(wrapper.text()).toContain(hint)
    await submitReply(wrapper, 'Must not be visible')
    expect(mocks.addMessage).not.toHaveBeenCalled()

    await selectInternalNoteMode(wrapper)
    expect(wrapper.get('#admin-ticket-reply').attributes()).not.toHaveProperty('disabled')
    await submitReply(wrapper, 'Private follow-up')
    expect(mocks.addMessage).toHaveBeenCalledOnce()
    expect(mocks.addMessage).toHaveBeenCalledWith('SUP-0123456789ABCDEF', { body: 'Private follow-up', internal: true })

    wrapper.unmount()
  })
})
