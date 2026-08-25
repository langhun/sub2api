import { onBeforeUnmount, onMounted, watch, type Ref } from 'vue'
import type { SupportTicket } from './api'

export interface SupportTicketDetailPollingOptions {
  ticketID: () => string
  currentTicket: Ref<SupportTicket | null>
  fetchTicket: (publicID: string) => Promise<SupportTicket>
  onTicket: (ticket: SupportTicket) => void
  shouldPause?: () => boolean
  intervalMs?: number
}

function latestMessageID(ticket: SupportTicket): number | undefined {
  return ticket.messages?.[ticket.messages.length - 1]?.id
}

export function mergeSupportTicket(current: SupportTicket, next: SupportTicket): SupportTicket {
  const currentMessages = current.messages ?? []
  const nextMessages = next.messages ?? []
  const sharedPrefix = currentMessages.length <= nextMessages.length
    && currentMessages.every((message, index) => message.id === nextMessages[index]?.id)

  if (sharedPrefix) currentMessages.push(...nextMessages.slice(currentMessages.length))
  else currentMessages.splice(0, currentMessages.length, ...nextMessages)

  const messages = currentMessages
  Object.assign(current, next)
  current.messages = messages
  return current
}

export function supportTicketChanged(current: SupportTicket | null, next: SupportTicket): boolean {
  if (!current || current.public_id !== next.public_id) return true
  return current.updated_at !== next.updated_at
    || current.last_activity_at !== next.last_activity_at
    || current.status !== next.status
    || current.priority !== next.priority
    || current.messages?.length !== next.messages?.length
    || latestMessageID(current) !== latestMessageID(next)
}

export function useSupportTicketDetailPolling(options: SupportTicketDetailPollingOptions) {
  let timer: ReturnType<typeof setInterval> | undefined
  let polling = false
  let pollAgain = false
  let generation = 0
  let disposed = false

  async function refresh(): Promise<void> {
    const publicID = options.ticketID()
    if (!publicID || disposed || document.visibilityState === 'hidden' || options.shouldPause?.()) return
    if (polling) {
      pollAgain = true
      return
    }

    polling = true
    const requestGeneration = generation
    try {
      const next = await options.fetchTicket(publicID)
      if (requestGeneration !== generation || publicID !== options.ticketID()) return
      if (supportTicketChanged(options.currentTicket.value, next)) options.onTicket(next)
    } catch {
      // Background refresh is best-effort and must not replace the current conversation.
    } finally {
      polling = false
      if (pollAgain) {
        pollAgain = false
        void refresh()
      }
    }
  }

  function onVisibilityChange() {
    if (document.visibilityState === 'visible') void refresh()
  }

  const stopRouteWatch = watch(options.ticketID, () => {
    generation += 1
    if (polling) pollAgain = true
  })

  onMounted(() => {
    timer = setInterval(() => void refresh(), options.intervalMs ?? 5_000)
    document.addEventListener('visibilitychange', onVisibilityChange)
  })

  onBeforeUnmount(() => {
    disposed = true
    if (timer) clearInterval(timer)
    document.removeEventListener('visibilitychange', onVisibilityChange)
    stopRouteWatch()
  })

  return { refresh }
}
