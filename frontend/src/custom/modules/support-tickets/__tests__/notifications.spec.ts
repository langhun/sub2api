import { describe, expect, it } from 'vitest'
import type { SupportTicket, SupportTicketMessage } from '../api'
import {
  findSupportTicketActivities,
  hasNewExternalReplySince,
  latestExternalReplyAt,
  type SupportTicketSnapshot,
} from '../notifications'

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
    ...overrides,
  }
}

function message(overrides: Partial<SupportTicketMessage> = {}): SupportTicketMessage {
  return {
    id: 1,
    author_type: 'user',
    message_type: 'reply',
    body: 'Need help',
    created_at: '2026-08-03T02:00:00Z',
    ...overrides,
  }
}

describe('support ticket browser notification activity detection', () => {
  it('only treats a changed reply from the other side as notification activity', () => {
    const previous = new Map<string, SupportTicketSnapshot>([
      ['SUP-0123456789ABCDEF', {
        publicID: 'SUP-0123456789ABCDEF',
        lastActivityAt: '2026-08-03T02:00:00Z',
        lastReplyBy: 'user',
      }],
    ])

    const supportReply = ticket({
      last_reply_by: 'admin',
      last_activity_at: '2026-08-03T03:00:00Z',
    })
    const ownReply = ticket({
      public_id: 'SUP-OWN',
      last_reply_by: 'user',
      last_activity_at: '2026-08-03T03:00:00Z',
    })

    expect(findSupportTicketActivities(previous, [supportReply, ownReply], 'user')).toEqual([supportReply])
    expect(findSupportTicketActivities(previous, [supportReply], 'admin')).toEqual([])
    expect(findSupportTicketActivities(new Map(), [ownReply], 'admin')).toEqual([ownReply])
  })

  it('does not repeat a notification when activity timestamp is unchanged', () => {
    const current = ticket({ last_reply_by: 'admin' })
    const previous = new Map<string, SupportTicketSnapshot>([
      [current.public_id, {
        publicID: current.public_id,
        lastActivityAt: current.last_activity_at,
        lastReplyBy: current.last_reply_by,
      }],
    ])

    expect(findSupportTicketActivities(previous, [current], 'user')).toEqual([])
  })

  it('ignores internal notes when resolving the latest external reply', () => {
    const detail = ticket({
      messages: [
        message({ id: 1, author_type: 'user', created_at: '2026-08-03T02:00:00Z' }),
        message({ id: 2, author_type: 'admin', created_at: '2026-08-03T03:00:00Z' }),
        message({ id: 3, author_type: 'admin', message_type: 'internal_note', created_at: '2026-08-03T04:00:00Z' }),
      ],
    })

    expect(latestExternalReplyAt(detail, 'user')).toBe('2026-08-03T03:00:00Z')
    expect(latestExternalReplyAt(detail, 'admin')).toBe('2026-08-03T02:00:00Z')
  })

  it('filters status-only activity by comparing the latest external reply timestamp', () => {
    const previous: SupportTicketSnapshot = {
      publicID: 'SUP-0123456789ABCDEF',
      lastActivityAt: '2026-08-03T03:00:00Z',
      lastReplyBy: 'admin',
    }
    const statusOnly = ticket({
      last_reply_by: 'admin',
      last_activity_at: '2026-08-03T04:00:00Z',
      messages: [
        message({ author_type: 'admin', created_at: '2026-08-03T03:00:00Z' }),
        message({ id: 2, author_type: 'admin', message_type: 'internal_note', created_at: '2026-08-03T04:00:00Z' }),
      ],
    })
    const reply = ticket({
      last_reply_by: 'admin',
      last_activity_at: '2026-08-03T04:00:00Z',
      messages: [message({ author_type: 'admin', created_at: '2026-08-03T04:00:00Z' })],
    })

    expect(hasNewExternalReplySince(statusOnly, previous, 'user')).toBe(false)
    expect(hasNewExternalReplySince(reply, previous, 'user')).toBe(true)
    expect(hasNewExternalReplySince(statusOnly, undefined, 'user')).toBe(false)
    expect(hasNewExternalReplySince(reply, undefined, 'user')).toBe(true)
  })
})
