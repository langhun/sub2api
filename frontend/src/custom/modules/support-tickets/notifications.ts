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
  fetchAttentionCount?: () => Promise<number>
  getAttentionCount?: (tickets: readonly SupportTicket[]) => number
  fetchTicket: (publicID: string) => Promise<SupportTicket>
  getTicketPath: (ticket: SupportTicket) => string
  getCopy: (ticket: SupportTicket, isNewTicket: boolean) => SupportTicketNotificationCopy
  getEnabledCopy?: () => SupportTicketNotificationCopy
  playSound?: () => void | Promise<void>
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

const permissionChangedEvent = 'support-ticket-notification-permission-changed'
const ticketSeenEvent = 'support-ticket-seen'

function seenStorageKey(role: SupportTicketNotificationRole) { return `support-ticket-seen:${role}` }
function readSeen(role: SupportTicketNotificationRole): Record<string, string> {
  try {
    const value = JSON.parse(localStorage.getItem(seenStorageKey(role)) || '{}')
    return value && typeof value === 'object' ? value as Record<string, string> : {}
  } catch { return {} }
}
function writeSeen(role: SupportTicketNotificationRole, value: Record<string, string>) {
  try { localStorage.setItem(seenStorageKey(role), JSON.stringify(value)) } catch { /* best effort */ }
}
function isSeen(role: SupportTicketNotificationRole, ticket: SupportTicket): boolean {
  const seenAt = readSeen(role)[ticket.public_id]
  if (!seenAt) return false
  const seenTime = Date.parse(seenAt)
  const activityTime = Date.parse(ticket.last_activity_at)
  return !Number.isNaN(seenTime) && !Number.isNaN(activityTime) ? seenTime >= activityTime : seenAt >= ticket.last_activity_at
}
export function markSupportTicketSeen(role: SupportTicketNotificationRole, ticket: SupportTicket) {
  const seen = readSeen(role)
  if (!seen[ticket.public_id] || Date.parse(ticket.last_activity_at) >= Date.parse(seen[ticket.public_id])) {
    seen[ticket.public_id] = ticket.last_activity_at
    writeSeen(role, seen)
    window.dispatchEvent(new CustomEvent(ticketSeenEvent, { detail: { role } }))
  }
}
export function supportTicketNeedsAttention(role: SupportTicketNotificationRole, ticket: SupportTicket): boolean {
  return (role === 'admin' ? ticket.status === 'open' : ticket.status === 'waiting_user') && !isSeen(role, ticket)
}

interface SupportTicketNotificationPermissionChange {
  permission: SupportTicketNotificationPermission
  enableByDefault: boolean
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

export async function requestSupportTicketNotificationPermission(): Promise<SupportTicketNotificationPermission> {
  const current = notificationConstructor()
  if (!current) return 'unsupported'

  const previous = current.permission
  let next: SupportTicketNotificationPermission = current.permission
  if (next === 'default') {
    try {
      next = await current.requestPermission()
    } catch {
      next = current.permission
    }
  }
  window.dispatchEvent(new CustomEvent<SupportTicketNotificationPermissionChange>(permissionChangedEvent, {
    detail: {
      permission: next,
      enableByDefault: previous === 'default' && next === 'granted',
    },
  }))
  return next
}

export function useSupportTicketBrowserNotifications(
  options: SupportTicketBrowserNotificationOptions,
): {
  supported: Readonly<Ref<boolean>>
  permission: Readonly<Ref<SupportTicketNotificationPermission>>
  enabled: Readonly<Ref<boolean>>
  canNotify: Readonly<Ref<boolean>>
  attentionCount: Readonly<Ref<number>>
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
  const attentionCount = ref(0)
  const previous = new Map<string, SupportTicketSnapshot>()
  const initialized = ref(false)
  let pollTimer: ReturnType<typeof setInterval> | undefined
  let polling = false
  let pollAgain = false

  function sync(tickets: readonly SupportTicket[]) {
    for (const [id, item] of snapshotMap(tickets)) previous.set(id, item)
    initialized.value = true
  }

  async function requestPermission(): Promise<SupportTicketNotificationPermission> {
    const next = await requestSupportTicketNotificationPermission()
    permission.value = next
    supported.value = next !== 'unsupported'
    if (next === 'granted') {
      enabled.value = true
      writeEnabled(storageKey, true)
    }
    return permission.value
  }

  async function toggle(): Promise<SupportTicketNotificationPermission> {
    if (permission.value !== 'granted') return requestPermission()
    const wasEnabled = canNotify.value
    enabled.value = !enabled.value
    writeEnabled(storageKey, enabled.value)
    if (!wasEnabled && canNotify.value) showEnabledNotification()
    return permission.value
  }

  function showBrowserNotification(copy: SupportTicketNotificationCopy, tag: string, onClick?: () => void) {
    const current = notificationConstructor()
    if (!current || !canNotify.value) return
    try {
      const notification = new current(copy.title, {
        body: copy.body,
        tag,
      })
      notification.onclick = () => {
        notification.close()
        window.focus()
        onClick?.()
      }
    } catch {
      // Browser notification constructors may fail when permission changes between polling and display.
    }
  }

  function showEnabledNotification() {
    if (!options.getEnabledCopy) return
    showBrowserNotification(options.getEnabledCopy(), 'support-ticket-notifications-enabled')
  }

  function showNotification(ticket: SupportTicket, isNewTicket: boolean) {
    showBrowserNotification(
      options.getCopy(ticket, isNewTicket),
      `support-ticket-${ticket.public_id}`,
      () => window.location.assign(options.getTicketPath(ticket)),
    )
  }

  function playAlertSound() {
    void Promise.resolve(options.playSound?.()).catch(() => {
      // Audio playback is best-effort because browsers can enforce autoplay policies.
    })
  }

  async function pollNow(): Promise<void> {
    if (polling) { pollAgain = true; return }
    if (options.shouldPoll?.() === false) return
    polling = true
    try {
      const [result, nextAttentionCount] = await Promise.all([
        options.fetchTickets(),
        options.fetchAttentionCount?.(),
      ])
      if (options.getAttentionCount) attentionCount.value = options.getAttentionCount(result.items)
      else if (typeof nextAttentionCount === 'number') attentionCount.value = nextAttentionCount

      const current = notificationConstructor()
      supported.value = Boolean(current)
      permission.value = current?.permission ?? 'unsupported'
      if (!initialized.value) {
        sync(result.items)
        return
      }

      const activities = findSupportTicketActivities(previous, result.items, options.role)
      const oldSnapshots = new Map(previous)
      const nextSnapshots = snapshotMap(result.items)

      if (!canNotify.value && !options.playSound) {
        sync(result.items)
        return
      }
      for (const activity of activities) {
        try {
          const detail = await options.fetchTicket(activity.public_id)
          if (hasNewExternalReplySince(detail, oldSnapshots.get(activity.public_id), options.role)) {
            const isNewTicket = !oldSnapshots.has(activity.public_id) && isNewTicketActivity(detail, options.role)
            showNotification(detail, isNewTicket)
            playAlertSound()
          }
        } catch {
          const old = oldSnapshots.get(activity.public_id)
          if (old) nextSnapshots.set(activity.public_id, old)
          else nextSnapshots.delete(activity.public_id)
        }
      }

      previous.clear()
      for (const [id, item] of nextSnapshots) previous.set(id, item)
    } catch {
      // Polling is best-effort and must never replace the page's own load/error state.
    } finally {
      polling = false
      if (pollAgain) {
        pollAgain = false
        void pollNow()
      }
    }
  }

  function onVisibilityChange() {
    if (document.visibilityState === 'hidden') void pollNow()
  }

  function onPermissionChanged(event: Event) {
    const detail = (event as CustomEvent<SupportTicketNotificationPermissionChange>).detail
    permission.value = detail.permission
    supported.value = detail.permission !== 'unsupported'
    if (detail.enableByDefault) {
      enabled.value = true
      writeEnabled(storageKey, true)
      showEnabledNotification()
    }
  }
  function onTicketSeen(event: Event) {
    const detail = (event as CustomEvent<{ role?: SupportTicketNotificationRole }>).detail
    if (detail?.role === options.role) void pollNow()
  }

  onMounted(() => {
    void pollNow()
    pollTimer = setInterval(() => void pollNow(), options.intervalMs ?? 30_000)
    document.addEventListener('visibilitychange', onVisibilityChange)
    window.addEventListener(permissionChangedEvent, onPermissionChanged)
    window.addEventListener(ticketSeenEvent, onTicketSeen)
  })

  onBeforeUnmount(() => {
    if (pollTimer) clearInterval(pollTimer)
    document.removeEventListener('visibilitychange', onVisibilityChange)
    window.removeEventListener(permissionChangedEvent, onPermissionChanged)
    window.removeEventListener(ticketSeenEvent, onTicketSeen)
  })

  return {
    supported,
    permission,
    enabled,
    canNotify,
    attentionCount,
    requestPermission,
    toggle,
    pollNow,
    sync,
  }
}
