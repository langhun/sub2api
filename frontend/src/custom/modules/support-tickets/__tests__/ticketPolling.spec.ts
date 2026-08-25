import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, ref } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { SupportTicket } from '../api'
import { mergeSupportTicket, supportTicketChanged, useSupportTicketDetailPolling } from '../ticketPolling'

function ticket(overrides: Partial<SupportTicket> = {}): SupportTicket {
  return {
    public_id: 'SUP-0123456789ABCDEF',
    user_id: 7,
    category: 'api',
    subject: 'Request failed',
    status: 'open',
    priority: 'normal',
    last_reply_by: 'user',
    last_activity_at: '2026-08-03T02:00:00Z',
    created_at: '2026-08-03T01:00:00Z',
    updated_at: '2026-08-03T02:00:00Z',
    messages: [{ id: 1, author_type: 'user', message_type: 'reply', body: 'Need help', created_at: '2026-08-03T02:00:00Z' }],
    ...overrides,
  }
}

function pollingHarness(fetchTicket: (publicID: string) => Promise<SupportTicket>, onTicket = vi.fn()) {
  return defineComponent({
    setup() {
      const currentTicket = ref<SupportTicket | null>(ticket())
      const publicID = ref(currentTicket.value.public_id)
      const polling = useSupportTicketDetailPolling({
        ticketID: () => publicID.value,
        currentTicket,
        fetchTicket,
        onTicket: (next) => {
          currentTicket.value = next
          onTicket(next)
        },
      })
      return { currentTicket, publicID, ...polling }
    },
    template: '<div />',
  })
}

describe('support ticket detail polling', () => {
  let visibility: DocumentVisibilityState

  beforeEach(() => {
    vi.useFakeTimers()
    visibility = 'visible'
    vi.spyOn(document, 'visibilityState', 'get').mockImplementation(() => visibility)
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  it('refreshes a changed ticket every five seconds without reapplying unchanged data', async () => {
    const changed = ticket({ updated_at: '2026-08-03T02:00:05Z', last_activity_at: '2026-08-03T02:00:05Z' })
    const fetchTicket = vi.fn().mockResolvedValueOnce(ticket()).mockResolvedValueOnce(changed)
    const onTicket = vi.fn()
    const wrapper = mount(pollingHarness(fetchTicket, onTicket))

    await vi.advanceTimersByTimeAsync(5_000)
    expect(onTicket).not.toHaveBeenCalled()
    await vi.advanceTimersByTimeAsync(5_000)
    expect(onTicket).toHaveBeenCalledOnce()
    expect(onTicket).toHaveBeenCalledWith(changed)
    wrapper.unmount()
  })

  it('pauses while hidden and refreshes immediately when the page becomes visible', async () => {
    const fetchTicket = vi.fn().mockResolvedValue(ticket())
    const wrapper = mount(pollingHarness(fetchTicket))
    visibility = 'hidden'

    await vi.advanceTimersByTimeAsync(10_000)
    expect(fetchTicket).not.toHaveBeenCalled()

    visibility = 'visible'
    document.dispatchEvent(new Event('visibilitychange'))
    await flushPromises()
    expect(fetchTicket).toHaveBeenCalledOnce()
    wrapper.unmount()
  })

  it('does not overlap requests and ignores background failures', async () => {
    let resolveRequest: ((value: SupportTicket) => void) | undefined
    const pending = new Promise<SupportTicket>((resolve) => { resolveRequest = resolve })
    const fetchTicket = vi.fn().mockReturnValueOnce(pending).mockRejectedValueOnce(new Error('network'))
    const onTicket = vi.fn()
    const wrapper = mount(pollingHarness(fetchTicket, onTicket))

    await vi.advanceTimersByTimeAsync(10_000)
    expect(fetchTicket).toHaveBeenCalledOnce()
    resolveRequest?.(ticket())
    await flushPromises()
    expect(fetchTicket).toHaveBeenCalledTimes(2)
    expect(onTicket).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('detects message-only changes even when timestamps are unchanged', () => {
    const current = ticket()
    const next = ticket({ messages: [...(current.messages ?? []), { id: 2, author_type: 'admin', message_type: 'reply', body: 'Reply', created_at: current.updated_at }] })
    expect(supportTicketChanged(current, next)).toBe(true)
    expect(supportTicketChanged(current, ticket())).toBe(false)
  })

  it('appends messages in place without replacing the ticket object', () => {
    const current = ticket()
    const next = ticket({ messages: [...(current.messages ?? []), { id: 2, author_type: 'admin', message_type: 'reply', body: 'Reply', created_at: current.updated_at }] })
    const result = mergeSupportTicket(current, next)
    expect(result).toBe(current)
    expect(current.messages?.map((message) => message.id)).toEqual([1, 2])
  })
})
