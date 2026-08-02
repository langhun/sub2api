<template>
  <AppLayout>
    <div class="mx-auto max-w-7xl space-y-6">
      <header class="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div><h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('admin.supportTickets.title') }}</h1><p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('admin.supportTickets.description') }}</p></div>
        <button v-if="ticket" class="btn btn-secondary self-start" type="button" @click="goToList"><Icon name="arrowLeft" size="sm" />{{ t('supportTickets.backToList') }}</button>
        <button v-else class="btn btn-secondary self-start" type="button" :disabled="loading" @click="() => loadList()"><Icon name="refresh" size="sm" />{{ t('common.refresh') }}</button>
      </header>

      <div v-if="loading" class="card flex min-h-56 items-center justify-center"><LoadingSpinner /></div>
      <div v-else-if="loadError" class="card py-12 text-center"><Icon name="exclamationCircle" size="xl" class="mx-auto text-red-500" /><p class="mt-3 text-sm text-gray-600 dark:text-dark-300">{{ loadError }}</p><button class="btn btn-secondary mt-4" type="button" @click="loadCurrent">{{ t('common.retry') }}</button></div>

      <section v-else-if="ticket" class="grid gap-6 xl:grid-cols-[minmax(0,1fr)_300px]">
        <div class="card overflow-hidden p-0">
          <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700"><div class="flex flex-wrap items-start justify-between gap-3"><div class="min-w-0"><p class="text-xs font-medium text-gray-500 dark:text-dark-400">{{ t('supportTickets.ticketNumber', { number: ticket.public_id }) }}</p><h2 class="mt-1 break-words text-lg font-semibold text-gray-900 dark:text-white">{{ ticket.subject }}</h2><p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('admin.supportTickets.requester') }}: #{{ ticket.user_id }}</p></div><span :class="['badge', ticketStatusClass(ticket.status)]">{{ statusLabel(ticket.status) }}</span></div></div>
          <div class="space-y-4 p-5">
            <div v-if="!ticket.messages?.length" class="py-8 text-center text-sm text-gray-500 dark:text-dark-400">{{ t('supportTickets.noMessages') }}</div>
            <article v-for="message in ticket.messages" :key="message.id" class="flex" :class="message.message_type === 'internal_note' ? 'justify-center' : message.author_type === 'admin' ? 'justify-end' : 'justify-start'">
              <div class="max-w-[88%] rounded-lg px-4 py-3 text-sm" :class="message.message_type === 'internal_note' ? 'border border-amber-200 bg-amber-50 text-amber-950 dark:border-amber-900/60 dark:bg-amber-900/20 dark:text-amber-100' : message.author_type === 'admin' ? 'bg-primary-600 text-white' : 'bg-gray-100 text-gray-800 dark:bg-dark-800 dark:text-dark-100'">
                <div class="mb-1 flex items-center justify-between gap-4 text-xs opacity-75"><span>{{ message.message_type === 'internal_note' ? t('admin.supportTickets.internalNote') : senderLabel(message.author_type) }}</span><time>{{ formatDateTimeToMinute(message.created_at) }}</time></div><p class="whitespace-pre-wrap break-words">{{ message.body }}</p>
              </div>
            </article>
          </div>
          <form class="border-t border-gray-100 p-5 dark:border-dark-700" @submit.prevent="sendMessage">
            <div class="flex flex-wrap items-center justify-between gap-3"><label class="input-label mb-0" for="admin-ticket-reply">{{ internalNote ? t('admin.supportTickets.internalNote') : t('admin.supportTickets.replyToUser') }}</label><label class="flex cursor-pointer items-center gap-2 text-sm text-gray-600 dark:text-dark-300"><input v-model="internalNote" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />{{ t('admin.supportTickets.internalNote') }}</label></div>
            <textarea id="admin-ticket-reply" v-model.trim="messageBody" data-testid="admin-ticket-message" class="input mt-2 min-h-28 resize-y" :placeholder="internalNote ? t('admin.supportTickets.internalNotePlaceholder') : t('supportTickets.replyPlaceholder')" maxlength="5000" required />
            <p v-if="!internalNote" class="input-hint mt-2 text-amber-700 dark:text-amber-300">{{ t('supportTickets.sensitiveNotice') }}</p>
            <div class="mt-3 flex justify-end"><button class="btn btn-primary" type="submit" :disabled="submitting || !messageBody">{{ submitting ? t('common.processing') : t('supportTickets.send') }}</button></div>
          </form>
        </div>
        <aside class="card h-fit p-5"><h2 class="font-semibold text-gray-900 dark:text-white">{{ t('admin.supportTickets.details') }}</h2><form class="mt-4 space-y-4" @submit.prevent="updateTicket"><label class="block"><span class="input-label">{{ t('supportTickets.status') }}</span><select v-model="editForm.status" class="input"><option v-for="status in ticketStatuses" :key="status" :value="status">{{ statusLabel(status) }}</option></select></label><label class="block"><span class="input-label">{{ t('supportTickets.priority') }}</span><select v-model="editForm.priority" class="input"><option v-for="priority in ticketPriorities" :key="priority" :value="priority">{{ priorityLabel(priority) }}</option></select></label><dl class="space-y-3 border-t border-gray-100 pt-4 text-sm dark:border-dark-700"><div><dt class="text-gray-500 dark:text-dark-400">{{ t('supportTickets.category') }}</dt><dd class="mt-1 font-medium text-gray-900 dark:text-white">{{ categoryLabel(ticket.category) }}</dd></div><div><dt class="text-gray-500 dark:text-dark-400">{{ t('supportTickets.latestActivity') }}</dt><dd class="mt-1 font-medium text-gray-900 dark:text-white">{{ formatDateTimeToMinute(ticket.last_activity_at) }}</dd></div></dl><button class="btn btn-secondary w-full" type="submit" :disabled="submitting">{{ t('admin.supportTickets.update') }}</button></form></aside>
      </section>

      <section v-else class="card overflow-hidden p-0">
        <div class="grid gap-3 border-b border-gray-100 p-4 sm:grid-cols-2 dark:border-dark-700"><select v-model="filters.status" class="input" @change="applyFilters"><option value="">{{ t('admin.supportTickets.allStatuses') }}</option><option v-for="status in ticketStatuses" :key="status" :value="status">{{ statusLabel(status) }}</option></select><select v-model="filters.priority" class="input" @change="applyFilters"><option value="">{{ t('admin.supportTickets.allPriorities') }}</option><option v-for="priority in ticketPriorities" :key="priority" :value="priority">{{ priorityLabel(priority) }}</option></select></div>
        <div v-if="tickets.length === 0" class="py-16 text-center text-sm text-gray-500 dark:text-dark-400">{{ t('admin.supportTickets.noTickets') }}</div>
        <button v-for="item in tickets" v-else :key="item.public_id" type="button" class="block w-full border-b border-gray-100 px-5 py-4 text-left transition-colors last:border-b-0 hover:bg-gray-50 dark:border-dark-700 dark:hover:bg-dark-800" @click="openTicket(item.public_id)"><div class="flex flex-wrap items-start justify-between gap-3"><div class="min-w-0"><div class="flex flex-wrap items-center gap-2"><span class="text-xs text-gray-500 dark:text-dark-400">{{ item.public_id }}</span><span :class="['badge', ticketStatusClass(item.status)]">{{ statusLabel(item.status) }}</span><span :class="['badge', ticketPriorityClass(item.priority)]">{{ priorityLabel(item.priority) }}</span></div><p class="mt-1 break-words font-medium text-gray-900 dark:text-white">{{ item.subject }}</p><p class="mt-1 text-xs text-gray-500 dark:text-dark-400">#{{ item.user_id }} · {{ categoryLabel(item.category) }}</p></div><time class="shrink-0 text-xs text-gray-500 dark:text-dark-400">{{ formatDateTimeToMinute(item.last_activity_at) }}</time></div></button>
        <div v-if="total > tickets.length" class="border-t border-gray-100 p-4 text-center dark:border-dark-700"><button class="btn btn-secondary btn-sm" type="button" :disabled="loadingMore" @click="loadMore">{{ t('common.loadMore') }}</button><p class="mt-2 text-xs text-gray-500 dark:text-dark-400">{{ t('supportTickets.pagination', { count: tickets.length, total }) }}</p></div>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import { useAppStore } from '@/stores/app'
import { formatDateTimeToMinute } from '@/utils/format'
import { addAdminSupportTicketMessage, getAdminSupportTicket, listAdminSupportTickets, updateAdminSupportTicket, type SupportTicket, type SupportTicketCategory, type SupportTicketPriority, type SupportTicketSenderType, type SupportTicketStatus } from '../api'
import { ticketPriorities, ticketPriorityClass, ticketStatuses, ticketStatusClass } from '../presentation'

const { t } = useI18n()
const appStore = useAppStore()
const route = useRoute()
const router = useRouter()
const tickets = ref<SupportTicket[]>([])
const ticket = ref<SupportTicket | null>(null)
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const loadingMore = ref(false)
const submitting = ref(false)
const loadError = ref('')
const messageBody = ref('')
const internalNote = ref(false)
const filters = reactive({ status: '' as SupportTicketStatus | '', priority: '' as SupportTicketPriority | '' })
const editForm = reactive({ status: 'open' as SupportTicketStatus, priority: 'normal' as SupportTicketPriority })

function errorMessage(error: unknown) { return (error as { message?: string })?.message || t('supportTickets.loadFailed') }
function statusLabel(value: SupportTicketStatus) { return t(`supportTickets.statuses.${value}`) }
function categoryLabel(value: SupportTicketCategory) { return t(`supportTickets.categories.${value}`) }
function priorityLabel(value: SupportTicketPriority) { return t(`supportTickets.priorities.${value}`) }
function senderLabel(value: SupportTicketSenderType) { return t(`supportTickets.sender.${value}`) }
function openTicket(publicID: string) { router.push({ name: 'AdminSupportTicketDetail', params: { id: publicID } }) }
function goToList() { router.push({ name: 'AdminSupportTickets' }) }

async function loadList(nextPage = 1) {
  const isMore = nextPage > 1
  try { isMore ? loadingMore.value = true : loading.value = true; const result = await listAdminSupportTickets({ page: nextPage, page_size: 20, ...(filters.status ? { status: filters.status } : {}), ...(filters.priority ? { priority: filters.priority } : {}) }); tickets.value = isMore ? [...tickets.value, ...result.items] : result.items; total.value = result.total; page.value = result.page; loadError.value = '' } catch (error) { loadError.value = errorMessage(error) } finally { loading.value = false; loadingMore.value = false }
}
async function loadTicket(id: string) {
  try { loading.value = true; ticket.value = await getAdminSupportTicket(id); editForm.status = ticket.value.status; editForm.priority = ticket.value.priority; loadError.value = '' } catch (error) { loadError.value = errorMessage(error) } finally { loading.value = false }
}
function loadCurrent() { const id = String(route.params.id || ''); return id ? loadTicket(id) : loadList() }
function loadMore() { return loadList(page.value + 1) }
function applyFilters() { return loadList(1) }
async function sendMessage() {
  if (!ticket.value || !messageBody.value) return
  try { submitting.value = true; await addAdminSupportTicketMessage(ticket.value.public_id, { body: messageBody.value, internal: internalNote.value }); messageBody.value = ''; internalNote.value = false; await loadTicket(ticket.value.public_id); appStore.showSuccess(t('supportTickets.sendSuccess')) } catch (error) { appStore.showError(errorMessage(error)) } finally { submitting.value = false }
}
async function updateTicket() {
  if (!ticket.value) return
  try {
    submitting.value = true
    if (editForm.status !== ticket.value.status) await updateAdminSupportTicket(ticket.value.public_id, { status: editForm.status })
    if (editForm.priority !== ticket.value.priority) await updateAdminSupportTicket(ticket.value.public_id, { priority: editForm.priority })
    await loadTicket(ticket.value.public_id)
    appStore.showSuccess(t('admin.supportTickets.updateSuccess'))
  } catch (error) { appStore.showError(errorMessage(error)) } finally { submitting.value = false }
}

watch(() => route.params.id, () => { ticket.value = null; messageBody.value = ''; loadCurrent() })
onMounted(loadCurrent)
</script>
