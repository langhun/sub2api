import { computed, onBeforeUnmount, onMounted, ref, type Ref } from 'vue'
import type { SupportTicket, SupportTicketAuthorType, SupportTicketList } from './api'

export type SupportTicketNotificationRole = 'user' | 'admin'
export type SupportTicketNotificationPermission = NotificationPermission | 'unsupported'

export interface SupportTicketNotificationCopy {
  title: string
  body: string
}

export interface SupportTicketBrowserNotificationOptions {
  role: SupportTicketNotificationRole
  fetchTickets: () => Promise<SupportTicketList>
  fetchTicket: (publicID: string) => Promise<SupportTicket>
  getTicketPath: (ticket: SupportTicket) => string
  getCopy: (ticket: SupportTicket, isNewTicket: boolean) => SupportTicketNotificationCopy
  shouldPoll?: () => boolean
  intervalMs?: number
}

export interface SupportTicketSnapshot {
  publicID: string
  lastActivityAt: string
  lastReplyBy: SupportTicketAuthorType
}

function expectedAuthor(role: SupportTicketNotificationRole): SupportTicketAuthorType {
  return role === 'admin' ? 'user' : 'admin'
}

function snapshot(ticket: SupportTicket): SupportTicketSnapshot {
  return {
    publicID: ticket.public_id,
    lastActivityAt: ticket.last_activity_at,
    lastReplyBy: ticket.last_reply_by,
  }
}

function snapshotMap(tickets: readonly SupportTicket[]): Map<string, SupportTicketSnapshot> {
  return new Map(tickets.map((ticket) => [ticket.public_id, snapshot(ticket)]))
}

/**
 * Finds changed tickets that could contain a reply from the other side.
 * Status/priority-only updates are filtered later by the message timestamp.
 */
export function findSupportTicketActivities(
  previous: ReadonlyMap<string, SupportTicketSnapshot>,
  current: readonly SupportTicket[],
  role: SupportTicketNotificationRole,
): SupportTicket[] {
  const author = expectedAuthor(role)
  return current.filter((ticket) => {
    if (ticket.last_reply_by !== author) return false
    const old = previous.get(ticket.public_id)
    return !old || old.lastActivityAt !== ticket.last_activity_at
  })
}

/** Return the newest external reply timestamp, ignoring internal notes. */
export function latestExternalReplyAt(ticket: SupportTicket, role: SupportTicketNotificationRole): string | undefined {
  const author = expectedAuthor(role)
  return ticket.messages
    ?.filter((message) => message.message_type === 'reply' && message.author_type === author)
    .map((message) => message.created_at)
    .sort((a, b) => Date.parse(b) - Date.parse(a))[0]
}

function isNewTicketActivity(ticket: SupportTicket, role: SupportTicketNotificationRole): boolean {
  if (role !== 'admin') return false
  const replies = ticket.messages?.filter((message) => message.message_type === 'reply') ?? []
  return replies.length === 1 && replies[0].author_type === 'user'
}

export function hasNewExternalReplySince(
  ticket: SupportTicket,
  old: SupportTicketSnapshot | undefined,
  role: SupportTicketNotificationRole,
): boolean {
  const latest = latestExternalReplyAt(ticket, role)
  if (!latest) return false

  const latestTime = Date.parse(latest)
  const comparisonActivity = old?.lastActivityAt ?? ticket.last_activity_at
  const comparisonTime = Date.parse(comparisonActivity)
  if (!Number.isNaN(latestTime) && !Number.isNaN(comparisonTime)) {
    return old ? latestTime > comparisonTime : latestTime === comparisonTime
  }
  return old ? latest !== comparisonActivity : latest === comparisonActivity
}

function notificationConstructor(): typeof Notification | undefined {
  if (typeof window === 'undefined' || typeof window.Notification === 'undefined') return undefined
  return window.Notification
}

function readEnabled(storageKey: string): boolean {
  try {
    return localStorage.getItem(storageKey) !== 'disabled'
  } catch {
    return true
  }
}

function writeEnabled(storageKey: string, enabled: boolean) {
  try {
    localStorage.setItem(storageKey, enabled ? 'enabled' : 'disabled')
  } catch {
    // Storage can be unavailable in private browsing; notifications still work for this session.
  }
}

export function useSupportTicketBrowserNotifications(
  options: SupportTicketBrowserNotificationOptions,
): {
  supported: Readonly<Ref<boolean>>
  permission: Readonly<Ref<SupportTicketNotificationPermission>>
  enabled: Readonly<Ref<boolean>>
  canNotify: Readonly<Ref<boolean>>
  requestPermission: () => Promise<SupportTicketNotificationPermission>
  toggle: () => Promise<SupportTicketNotificationPermission>
  pollNow: () => Promise<void>
  sync: (tickets: readonly SupportTicket[]) => void
} {
  const storageKey = `support-ticket-browser-notifications:${options.role}`
  const Constructor = notificationConstructor()
  const supported = ref(Boolean(Constructor))
  const permission = ref<SupportTicketNotificationPermission>(Constructor ? Constructor.permission : 'unsupported')
  const enabled = ref(readEnabled(storageKey))
  const canNotify = computed(() => supported.value && permission.value === 'granted' && enabled.value)
  const previous = new Map<string, SupportTicketSnapshot>()
  const initialized = ref(false)
  let pollTimer: ReturnType<typeof setInterval> | undefined
  let polling = false

  function sync(tickets: readonly SupportTicket[]) {
    for (const [id, item] of snapshotMap(tickets)) previous.set(id, item)
    initialized.value = true
  }

  async function requestPermission(): Promise<SupportTicketNotificationPermission> {
    const current = notificationConstructor()
    if (!current) {
      supported.value = false
      permission.value = 'unsupported'
      return permission.value
    }

    permission.value = current.permission
    if (permission.value === 'denied') return permission.value
    if (permission.value === 'granted') {
      enabled.value = true
      writeEnabled(storageKey, true)
      return permission.value
    }

    const next = await current.requestPermission()
    permission.value = next
    if (next === 'granted') {
      enabled.value = true
      writeEnabled(storageKey, true)
    }
    return permission.value
  }

  async function toggle(): Promise<SupportTicketNotificationPermission> {
    if (permission.value !== 'granted') return requestPermission()
    enabled.value = !enabled.value
    writeEnabled(storageKey, enabled.value)
    return permission.value
  }

  function showNotification(ticket: SupportTicket, isNewTicket: boolean) {
    const current = notificationConstructor()
    if (!current || !canNotify.value || document.visibilityState !== 'hidden') return
    try {
      const copy = options.getCopy(ticket, isNewTicket)
      const notification = new current(copy.title, {
        body: copy.body,
        tag: `support-ticket-${ticket.public_id}`,
      })
      notification.onclick = () => {
        notification.close()
        window.focus()
        window.location.assign(options.getTicketPath(ticket))
      }
    } catch {
      // Browser notification constructors may fail when permission changes between polling and display.
    }
  }

  async function pollNow(): Promise<void> {
    if (polling || options.shouldPoll?.() === false || !supported.value) return
    polling = true
    try {
      const result = await options.fetchTickets()
      const activities = initialized.value
        ? findSupportTicketActivities(previous, result.items, options.role)
        : []
      const oldSnapshots = new Map(previous)
      sync(result.items)

      if (!canNotify.value || document.visibilityState !== 'hidden') return
      for (const activity of activities) {
        try {
          const detail = await options.fetchTicket(activity.public_id)
          if (hasNewExternalReplySince(detail, oldSnapshots.get(activity.public_id), options.role)) {
            showNotification(detail, !oldSnapshots.has(activity.public_id) && isNewTicketActivity(detail, options.role))
          }
        } catch {
          // A transient detail request failure must not break the polling loop.
        }
      }
    } catch {
      // Polling is best-effort and must never replace the page's own load/error state.
    } finally {
      polling = false
    }
  }

  function onVisibilityChange() {
    if (document.visibilityState === 'hidden') void pollNow()
  }

  onMounted(() => {
    void pollNow()
    pollTimer = setInterval(() => void pollNow(), options.intervalMs ?? 30_000)
    document.addEventListener('visibilitychange', onVisibilityChange)
  })

  onBeforeUnmount(() => {
    if (pollTimer) clearInterval(pollTimer)
    document.removeEventListener('visibilitychange', onVisibilityChange)
  })

  return {
    supported,
    permission,
    enabled,
    canNotify,
    requestPermission,
    toggle,
    pollNow,
    sync,
  }
}
