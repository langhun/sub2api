<template>
  <button
    v-if="featureEnabled && notificationsSupported"
    type="button"
    class="relative flex h-9 w-9 items-center justify-center rounded-lg text-gray-600 transition-colors hover:bg-gray-100 hover:text-gray-900 dark:text-gray-400 dark:hover:bg-dark-800 dark:hover:text-white"
    :class="{ 'text-primary-600 dark:text-primary-400': browserNotificationsOn }"
    :aria-label="notificationLabel"
    :title="notificationLabel"
    :aria-pressed="browserNotificationsOn"
    @click="toggleNotifications"
  >
    <Icon name="bell" size="md" />
    <span v-if="browserNotificationsOn" class="absolute right-1 top-1 h-1.5 w-1.5 rounded-full bg-primary-500" />
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore, useAuthStore } from '@/stores'
import { isFeatureFlagEnabled } from '@/utils/featureFlags'
import { supportTicketFeatureFlag } from '../navigation'
import { getAdminSupportTicket, getSupportTicket, listAdminSupportTickets, listSupportTickets } from '../api'
import { useSupportTicketBrowserNotifications } from '../notifications'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const role = authStore.isAdmin ? 'admin' : 'user'
const featureEnabled = computed(() => isFeatureFlagEnabled(supportTicketFeatureFlag))

const notifications = useSupportTicketBrowserNotifications({
  role,
  shouldPoll: () => featureEnabled.value,
  fetchTickets: () => role === 'admin'
    ? listAdminSupportTickets({ page: 1, page_size: 20 })
    : listSupportTickets({ page: 1, page_size: 20 }),
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
})

const notificationsSupported = notifications.supported
const notificationPermission = notifications.permission
const browserNotificationsOn = notifications.canNotify
const notificationLabel = computed(() => {
  if (notificationPermission.value === 'denied') return t('supportTickets.browserNotificationsDenied')
  return t(browserNotificationsOn.value ? 'supportTickets.disableBrowserNotifications' : 'supportTickets.enableBrowserNotifications')
})

async function toggleNotifications() {
  const permission = await notifications.toggle()
  if (permission === 'denied') {
    appStore.showError(t('supportTickets.browserNotificationsDenied'))
  } else if (permission === 'granted' && browserNotificationsOn.value) {
    appStore.showSuccess(t('supportTickets.browserNotificationsEnabled'))
  }
}
</script>
