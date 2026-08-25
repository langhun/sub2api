<template>
  <div class="relative bg-gray-50/80 dark:bg-dark-900/40">
    <div
      ref="scrollContainer"
      data-testid="ticket-conversation"
      role="log"
      aria-live="polite"
      class="h-[clamp(20rem,55vh,36rem)] overflow-y-auto overscroll-contain px-4 py-6 sm:px-6"
      @scroll="onScroll"
    >
      <div v-if="messages.length === 0" class="flex min-h-full flex-col items-center justify-center text-center">
      <span class="flex h-11 w-11 items-center justify-center rounded-full bg-white text-gray-400 shadow-sm ring-1 ring-gray-200 dark:bg-dark-800 dark:ring-dark-700">
        <Icon name="chat" size="lg" />
      </span>
      <p class="mt-3 text-sm text-gray-500 dark:text-dark-400">{{ emptyLabel }}</p>
    </div>

      <div v-else class="space-y-6">
      <article
        v-for="message in messages"
        :key="message.id"
        data-testid="ticket-message"
        class="flex"
        :class="message.message_type === 'internal_note' ? 'justify-center' : isOwn(message) ? 'justify-end' : 'justify-start'"
      >
        <div v-if="message.message_type === 'internal_note'" class="flex max-w-2xl items-start gap-2 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-900 dark:border-amber-900/70 dark:bg-amber-900/20 dark:text-amber-100">
          <Icon name="infoCircle" size="sm" class="mt-0.5 shrink-0" />
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-x-2 gap-y-1 font-medium">
              <span>{{ internalNoteLabel }}</span>
              <time :datetime="message.created_at" class="font-normal opacity-70">{{ formatDateTimeToMinute(message.created_at) }}</time>
            </div>
            <p class="mt-1 whitespace-pre-wrap break-words leading-5">{{ message.body }}</p>
          </div>
        </div>

        <div v-else class="flex max-w-[92%] items-start gap-2.5 sm:max-w-[82%]" :class="isOwn(message) ? 'flex-row-reverse' : ''">
          <span
            class="flex h-9 w-9 shrink-0 items-center justify-center rounded-full ring-1"
            :class="isOwn(message) ? 'bg-primary-600 text-white ring-primary-600' : 'bg-white text-gray-500 ring-gray-200 dark:bg-dark-800 dark:text-dark-300 dark:ring-dark-700'"
          >
            <Icon :name="authorIcon(message)" size="sm" />
          </span>
          <div class="min-w-0">
            <div class="mb-1.5 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs" :class="isOwn(message) ? 'justify-end' : ''">
              <span class="font-medium text-gray-700 dark:text-dark-200">{{ authorLabel(message) }}</span>
              <time :datetime="message.created_at" class="text-gray-400 dark:text-dark-500">{{ formatDateTimeToMinute(message.created_at) }}</time>
            </div>
            <div
              data-testid="ticket-message-bubble"
              class="rounded-lg px-4 py-3 text-sm leading-6 shadow-sm ring-1"
              :class="isOwn(message) ? 'bg-primary-600 text-white ring-primary-600' : 'bg-white text-gray-800 ring-gray-200 dark:bg-dark-800 dark:text-dark-100 dark:ring-dark-700'"
            >
              <p class="whitespace-pre-wrap break-words">{{ message.body }}</p>
            </div>
          </div>
        </div>
      </article>
      </div>
    </div>

    <button
      v-if="hasNewMessages"
      type="button"
      data-testid="ticket-new-messages"
      class="btn btn-primary btn-sm absolute bottom-4 left-1/2 z-10 -translate-x-1/2 shadow-lg"
      @click="scrollToLatest('smooth')"
    >
      {{ newMessagesLabel }}
      <Icon name="arrowDown" size="xs" />
    </button>
  </div>
</template>

<script setup lang="ts">
import { nextTick, onMounted, ref, watch } from 'vue'
import Icon from '@/components/icons/Icon.vue'
import { formatDateTimeToMinute } from '@/utils/format'
import type { SupportTicketMessage } from '../api'

const props = defineProps<{
  messages: SupportTicketMessage[]
  ownAuthor: 'user' | 'admin'
  userLabel: string
  adminLabel: string
  systemLabel: string
  internalNoteLabel: string
  emptyLabel: string
  newMessagesLabel: string
}>()

const scrollContainer = ref<HTMLElement | null>(null)
const hasNewMessages = ref(false)
let followsLatest = true

function isNearBottom(element: HTMLElement): boolean {
  return element.scrollHeight - element.scrollTop - element.clientHeight <= 72
}

function onScroll() {
  const element = scrollContainer.value
  if (!element) return
  followsLatest = isNearBottom(element)
  if (followsLatest) hasNewMessages.value = false
}

function scrollToLatest(behavior: 'auto' | 'smooth' = 'auto') {
  const element = scrollContainer.value
  if (!element) return
  if (typeof element.scrollTo === 'function') element.scrollTo({ top: element.scrollHeight, behavior })
  else element.scrollTop = element.scrollHeight
  followsLatest = true
  hasNewMessages.value = false
}

watch(
  () => props.messages.map((message) => message.id).join(','),
  async (current, previous) => {
    if (current === previous) return
    const shouldFollow = followsLatest
    await nextTick()
    if (shouldFollow) scrollToLatest()
    else hasNewMessages.value = true
  },
)

onMounted(async () => {
  await nextTick()
  scrollToLatest()
})

function isOwn(message: SupportTicketMessage) { return message.author_type === props.ownAuthor }
function authorLabel(message: SupportTicketMessage) {
  if (message.author_type === 'user') return props.userLabel
  if (message.author_type === 'admin') return props.adminLabel
  return props.systemLabel
}
function authorIcon(message: SupportTicketMessage): 'user' | 'chat' | 'infoCircle' {
  if (message.author_type === 'user') return 'user'
  if (message.author_type === 'admin') return 'chat'
  return 'infoCircle'
}

defineExpose({ scrollToLatest })
</script>
