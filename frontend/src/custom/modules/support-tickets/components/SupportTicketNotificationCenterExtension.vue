<template>
  <Teleport v-if="open && featureEnabled" to="#custom-notification-center-extensions">
    <section class="overflow-hidden border-b border-gray-100 bg-gray-50/70 px-6 py-4 dark:border-dark-700 dark:bg-dark-900/30">
      <div class="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3">
        <router-link
          :to="ticketPath"
          class="group flex min-w-0 flex-1 items-center gap-3"
          @click="emit('close')"
        >
          <span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-primary-100 text-primary-700 dark:bg-primary-900/40 dark:text-primary-300">
            <Icon name="chat" size="sm" />
          </span>
          <span class="min-w-0">
            <span class="flex items-center gap-2">
              <span class="truncate text-sm font-medium text-gray-900 dark:text-white">
                {{ t('supportTickets.attentionRequired') }}
              </span>
              <span
                v-if="attentionCount > 0"
                class="inline-flex min-w-5 items-center justify-center rounded-full bg-red-500 px-1.5 py-0.5 text-xs font-semibold text-white"
              >
                {{ attentionCount }}
              </span>
            </span>
            <span class="mt-0.5 block text-xs text-gray-500 dark:text-gray-400">
              {{ notificationStatus }}
            </span>
          </span>
          <Icon name="chevronRight" size="sm" class="ml-auto shrink-0 text-gray-400 transition-transform group-hover:translate-x-0.5" />
        </router-link>

        <div class="flex shrink-0 items-center gap-2">
          <button
            type="button"
            class="flex h-8 w-8 items-center justify-center rounded-md text-gray-500 transition-colors hover:bg-gray-200 hover:text-primary-700 dark:text-dark-300 dark:hover:bg-dark-700 dark:hover:text-primary-300"
            :aria-label="alertSoundLabel"
            :title="alertSoundLabel"
            @click="previewAlertSound"
          >
            <Icon name="play" size="sm" />
          </button>
          <button
            v-if="notificationsSupported"
            type="button"
            role="switch"
            class="relative h-6 w-11 overflow-hidden rounded-full transition-colors"
            :class="browserNotificationsOn ? 'bg-primary-600' : 'bg-gray-300 dark:bg-dark-600'"
            :aria-checked="browserNotificationsOn"
            :aria-label="notificationLabel"
            :title="notificationLabel"
            @click="toggleNotifications"
          >
            <span
              class="absolute left-0.5 top-0.5 h-5 w-5 rounded-full bg-white shadow-sm transition-transform"
              :class="browserNotificationsOn ? 'translate-x-5' : 'translate-x-0'"
            />
          </button>
        </div>
      </div>
    </section>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore, useAuthStore } from '@/stores'
import { isFeatureFlagEnabled } from '@/utils/featureFlags'
import { getAdminSupportTicket, getSupportTicket, listAdminSupportTickets, listSupportTickets } from '../api'
import { playSupportTicketAlertSound } from '../alertSound'
import { supportTicketFeatureFlag } from '../navigation'
import { useSupportTicketBrowserNotifications } from '../notifications'

defineProps<{ open: boolean }>()
const emit = defineEmits<{
  close: []
  attentionChange: [count: number]
}>()

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const role = authStore.isAdmin ? 'admin' : 'user'
const featureEnabled = computed(() => isFeatureFlagEnabled(supportTicketFeatureFlag))
const ticketPath = role === 'admin' ? '/admin/support/tickets' : '/tickets'

const notifications = useSupportTicketBrowserNotifications({
  role,
  shouldPoll: () => featureEnabled.value,
  fetchTickets: () => role === 'admin'
    ? listAdminSupportTickets({ page: 1, page_size: 20 })
    : listSupportTickets({ page: 1, page_size: 20 }),
  fetchAttentionCount: async () => {
    const result = role === 'admin'
      ? await listAdminSupportTickets({ page: 1, page_size: 1, status: 'open' })
      : await listSupportTickets({ page: 1, page_size: 1, status: 'waiting_user' })
    return result.total
  },
  fetchTicket: role === 'admin' ? getAdminSupportTicket : getSupportTicket,
  getTicketPath: (item) => role === 'admin'
    ? `/admin/support/tickets/${item.public_id}`
    : `/tickets/${item.public_id}`,
  getCopy: (item, isNewTicket) => role === 'admin'
    ? {
        title: t(isNewTicket ? 'admin.supportTickets.browserNotificationTicketTitle' : 'admin.supportTickets.browserNotificationReplyTitle'),
        body: t(isNewTicket ? 'admin.supportTickets.browserNotificationTicketBody' : 'admin.supportTickets.browserNotificationReplyBody', { subject: item.subject }),
      }
    : {
        title: t('supportTickets.browserNotificationReplyTitle'),
        body: t('supportTickets.browserNotificationReplyBody', { subject: item.subject }),
      },
  getEnabledCopy: () => ({
    title: t('supportTickets.browserNotificationTestTitle'),
    body: t('supportTickets.browserNotificationTestBody'),
  }),
  playSound: playSupportTicketAlertSound,
})

const notificationsSupported = notifications.supported
const notificationPermission = notifications.permission
const browserNotificationsOn = notifications.canNotify
const attentionCount = notifications.attentionCount
const notificationLabel = computed(() => {
  if (notificationPermission.value === 'denied') return t('supportTickets.browserNotificationsDenied')
  return t(browserNotificationsOn.value ? 'supportTickets.disableBrowserNotifications' : 'supportTickets.enableBrowserNotifications')
})
const alertSoundLabel = computed(() => t('supportTickets.previewAlertSound'))
const notificationStatus = computed(() => {
  if (!notificationsSupported.value) return t('supportTickets.browserNotificationsUnsupported')
  if (notificationPermission.value === 'denied') return t('supportTickets.browserNotificationsDenied')
  if (notificationPermission.value === 'default') return t('supportTickets.browserNotificationsPending')
  return t(browserNotificationsOn.value ? 'supportTickets.browserNotificationsEnabled' : 'supportTickets.browserNotificationsDisabled')
})

watch(attentionCount, (count) => emit('attentionChange', count), { immediate: true })

async function toggleNotifications() {
  const permission = await notifications.toggle()
  if (permission === 'denied') {
    appStore.showError(t('supportTickets.browserNotificationsDenied'))
  } else if (permission === 'granted' && browserNotificationsOn.value) {
    appStore.showSuccess(t('supportTickets.browserNotificationsEnabled'))
  }
}

function previewAlertSound() {
  void playSupportTicketAlertSound()
}
</script>
