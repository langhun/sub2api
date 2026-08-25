import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import TicketConversation from '../components/TicketConversation.vue'
import type { SupportTicketMessage } from '../api'

const messages: SupportTicketMessage[] = [
  { id: 1, author_type: 'user', message_type: 'reply', body: 'I need help', created_at: '2026-08-06T01:00:00Z' },
  { id: 2, author_type: 'admin', message_type: 'reply', body: 'We are checking it', created_at: '2026-08-06T01:05:00Z' },
  { id: 3, author_type: 'admin', message_type: 'internal_note', body: 'Escalated', created_at: '2026-08-06T01:06:00Z' },
]

function mountConversation(ownAuthor: 'user' | 'admin') {
  return mount(TicketConversation, {
    props: {
      messages,
      ownAuthor,
      userLabel: '#7 · Alice',
      adminLabel: 'Support',
      systemLabel: 'System',
      internalNoteLabel: 'Internal note',
      emptyLabel: 'No messages',
      newMessagesLabel: 'New messages',
    },
  })
}

describe('TicketConversation', () => {
  it('renders requester, support, and internal-note identities', () => {
    const wrapper = mountConversation('admin')
    expect(wrapper.text()).toContain('#7 · Alice')
    expect(wrapper.text()).toContain('Support')
    expect(wrapper.text()).toContain('Internal note')
    expect(wrapper.findAll('[data-testid="ticket-message"]')).toHaveLength(3)
  })

  it('reverses the visual ownership without changing message content', () => {
    const wrapper = mountConversation('user')
    const rows = wrapper.findAll('[data-testid="ticket-message"]')
    expect(rows[0].classes()).toContain('justify-end')
    expect(rows[1].classes()).toContain('justify-start')
    expect(wrapper.text()).toContain('We are checking it')
  })

  it('keeps history position and offers a jump when a new message arrives', async () => {
    const wrapper = mountConversation('user')
    await flushPromises()
    const conversation = wrapper.get('[data-testid="ticket-conversation"]').element as HTMLElement
    Object.defineProperties(conversation, {
      scrollHeight: { configurable: true, value: 1000 },
      clientHeight: { configurable: true, value: 400 },
    })
    conversation.scrollTop = 100
    await wrapper.get('[data-testid="ticket-conversation"]').trigger('scroll')

    await wrapper.setProps({
      messages: [...messages, { id: 4, author_type: 'admin', message_type: 'reply', body: 'New reply', created_at: '2026-08-06T01:10:00Z' }],
    })
    await flushPromises()

    expect(conversation.scrollTop).toBe(100)
    expect(wrapper.get('[data-testid="ticket-new-messages"]').text()).toContain('New messages')

    await wrapper.get('[data-testid="ticket-new-messages"]').trigger('click')
    expect(conversation.scrollTop).toBe(1000)
    expect(wrapper.find('[data-testid="ticket-new-messages"]').exists()).toBe(false)
  })

  it('follows new messages automatically while already at the bottom', async () => {
    const wrapper = mountConversation('admin')
    await flushPromises()
    const conversation = wrapper.get('[data-testid="ticket-conversation"]').element as HTMLElement
    Object.defineProperties(conversation, {
      scrollHeight: { configurable: true, value: 1200 },
      clientHeight: { configurable: true, value: 400 },
    })
    conversation.scrollTop = 800
    await wrapper.get('[data-testid="ticket-conversation"]').trigger('scroll')

    await wrapper.setProps({
      messages: [...messages, { id: 4, author_type: 'user', message_type: 'reply', body: 'Latest reply', created_at: '2026-08-06T01:10:00Z' }],
    })
    await flushPromises()

    expect(conversation.scrollTop).toBe(1200)
    expect(wrapper.find('[data-testid="ticket-new-messages"]').exists()).toBe(false)
  })
})
