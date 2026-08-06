<template>
  <AppLayout>
    <div class="mx-auto max-w-7xl space-y-6">
      <header class="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div><h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('admin.supportTickets.title') }}</h1><p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('admin.supportTickets.description') }}</p></div>
        <button v-if="ticket" class="btn btn-secondary self-start" type="button" @click="goToList"><Icon name="arrowLeft" size="sm" />{{ t('supportTickets.backToList') }}</button>
        <div v-else class="flex flex-wrap gap-2 self-start">
          <button class="btn btn-secondary" type="button" :disabled="loading" @click="() => loadList()"><Icon name="refresh" size="sm" />{{ t('common.refresh') }}</button>
          <button class="btn btn-primary" type="button" @click="openCreateDialog"><Icon name="plus" size="sm" />{{ t('admin.supportTickets.createOutbound') }}</button>
        </div>
      </header>

      <div v-if="loading" class="card flex min-h-56 items-center justify-center"><LoadingSpinner /></div>
      <div v-else-if="loadError" class="card py-12 text-center"><Icon name="exclamationCircle" size="xl" class="mx-auto text-red-500" /><p class="mt-3 text-sm text-gray-600 dark:text-dark-300">{{ loadError }}</p><button class="btn btn-secondary mt-4" type="button" @click="loadCurrent">{{ t('common.retry') }}</button></div>

      <section v-else-if="ticket" class="grid gap-6 xl:grid-cols-[minmax(0,1fr)_300px]">
        <div class="card overflow-hidden p-0">
          <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700"><div class="flex flex-wrap items-start justify-between gap-3"><div class="min-w-0"><p class="font-mono text-xs font-medium text-gray-500 dark:text-dark-400">{{ t('supportTickets.ticketNumber', { number: ticket.public_id }) }}</p><h2 class="mt-1 break-words text-lg font-semibold text-gray-900 dark:text-white">{{ ticket.subject }}</h2><p class="mt-1 break-words text-sm text-gray-500 dark:text-dark-400">{{ t('admin.supportTickets.requester') }}: {{ requesterLabel(ticket) }}</p></div><span :class="['badge', ticketStatusClass(ticket.status)]" :aria-label="statusLabel(ticket.status)">{{ statusLabel(ticket.status) }}</span></div></div>
          <div class="space-y-4 p-5">
            <div v-if="!ticket.messages?.length" class="py-8 text-center text-sm text-gray-500 dark:text-dark-400">{{ t('supportTickets.noMessages') }}</div>
            <article v-for="message in ticket.messages" :key="message.id" class="flex" :class="message.message_type === 'internal_note' ? 'justify-center' : message.author_type === 'admin' ? 'justify-end' : 'justify-start'">
              <div class="max-w-[88%] rounded-lg px-4 py-3 text-sm" :class="message.message_type === 'internal_note' ? 'border border-amber-200 bg-amber-50 text-amber-950 dark:border-amber-900/60 dark:bg-amber-900/20 dark:text-amber-100' : message.author_type === 'admin' ? 'bg-primary-600 text-white' : 'bg-gray-100 text-gray-800 dark:bg-dark-800 dark:text-dark-100'">
                <div class="mb-1 flex items-center justify-between gap-4 text-xs opacity-75"><span>{{ message.message_type === 'internal_note' ? t('admin.supportTickets.internalNote') : message.author_type === 'user' ? requesterLabel(ticket) : senderLabel(message.author_type) }}</span><time>{{ formatDateTimeToMinute(message.created_at) }}</time></div><p class="whitespace-pre-wrap break-words">{{ message.body }}</p>
              </div>
            </article>
          </div>
          <form class="border-t border-gray-100 p-5 dark:border-dark-700" @submit.prevent="sendMessage">
            <div class="flex flex-wrap items-center justify-between gap-3"><label class="input-label mb-0" for="admin-ticket-reply">{{ internalNote ? t('admin.supportTickets.internalNote') : t('admin.supportTickets.replyToUser') }}</label><label class="flex cursor-pointer items-center gap-2 text-sm text-gray-600 dark:text-dark-300"><input v-model="internalNote" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />{{ t('admin.supportTickets.internalNote') }}</label></div>
            <textarea id="admin-ticket-reply" v-model.trim="messageBody" data-testid="admin-ticket-message" class="input mt-2 min-h-28 resize-y" :disabled="!canSendMessage" :placeholder="internalNote ? t('admin.supportTickets.internalNotePlaceholder') : t('supportTickets.replyPlaceholder')" maxlength="5000" required />
            <p v-if="!internalNote && !canSendMessage" class="input-hint mt-2 text-amber-700 dark:text-amber-300">{{ replyUnavailableHint }}</p>
            <p v-if="!internalNote" class="input-hint mt-2 text-amber-700 dark:text-amber-300">{{ t('supportTickets.sensitiveNotice') }}</p>
            <div class="mt-3 flex justify-end"><button class="btn btn-primary" type="submit" :disabled="submitting || !messageBody || !canSendMessage">{{ submitting ? t('common.processing') : t('supportTickets.send') }}</button></div>
          </form>
        </div>
        <aside class="card h-fit p-5"><h2 class="font-semibold text-gray-900 dark:text-white">{{ t('admin.supportTickets.details') }}</h2><form class="mt-4 space-y-4" @submit.prevent="updateTicket"><label class="block"><span class="input-label">{{ t('supportTickets.status') }}</span><select v-model="editForm.status" class="input"><option v-for="status in ticketStatuses" :key="status" :value="status">{{ statusLabel(status) }}</option></select></label><label class="block"><span class="input-label">{{ t('supportTickets.priority') }}</span><select v-model="editForm.priority" class="input"><option v-for="priority in ticketPriorities" :key="priority" :value="priority">{{ priorityLabel(priority) }}</option></select></label><dl class="space-y-3 border-t border-gray-100 pt-4 text-sm dark:border-dark-700"><div><dt class="text-gray-500 dark:text-dark-400">{{ t('supportTickets.category') }}</dt><dd class="mt-1 font-medium text-gray-900 dark:text-white">{{ categoryLabel(ticket.category) }}</dd></div><div><dt class="text-gray-500 dark:text-dark-400">{{ t('supportTickets.latestActivity') }}</dt><dd class="mt-1 font-medium text-gray-900 dark:text-white">{{ formatDateTimeToMinute(ticket.last_activity_at) }}</dd></div></dl><button class="btn btn-secondary w-full" type="submit" :disabled="submitting">{{ t('admin.supportTickets.update') }}</button></form></aside>
      </section>

      <section v-else class="card overflow-hidden p-0">
        <div class="grid gap-3 border-b border-gray-100 p-4 sm:grid-cols-2 dark:border-dark-700"><select v-model="filters.status" class="input" @change="applyFilters"><option value="">{{ t('admin.supportTickets.allStatuses') }}</option><option v-for="status in ticketStatuses" :key="status" :value="status">{{ statusLabel(status) }}</option></select><select v-model="filters.priority" class="input" @change="applyFilters"><option value="">{{ t('admin.supportTickets.allPriorities') }}</option><option v-for="priority in ticketPriorities" :key="priority" :value="priority">{{ priorityLabel(priority) }}</option></select></div>
        <div v-if="tickets.length === 0" class="py-16 text-center text-sm text-gray-500 dark:text-dark-400">{{ t('admin.supportTickets.noTickets') }}</div>
        <button v-for="item in tickets" v-else :key="item.public_id" type="button" class="block w-full border-b border-l-2 border-b-gray-100 px-5 py-4 text-left transition-colors last:border-b-0 hover:bg-gray-50 dark:border-b-dark-700 dark:hover:bg-dark-800" :class="item.status === 'open' ? 'border-l-amber-500 bg-amber-50/40 dark:bg-amber-900/10' : 'border-l-transparent'" @click="openTicket(item.public_id)"><div class="flex flex-wrap items-start justify-between gap-3"><div class="min-w-0"><div class="flex flex-wrap items-center gap-2"><span class="font-mono text-xs text-gray-500 dark:text-dark-400">{{ item.public_id }}</span><span :class="['badge', ticketStatusClass(item.status)]" :aria-label="statusLabel(item.status)">{{ statusLabel(item.status) }}</span><span :class="['badge', ticketPriorityClass(item.priority)]">{{ priorityLabel(item.priority) }}</span></div><p class="mt-1 break-words font-medium text-gray-900 dark:text-white">{{ item.subject }}</p><p class="mt-1 break-words text-xs text-gray-500 dark:text-dark-400">{{ requesterLabel(item) }} · {{ categoryLabel(item.category) }}</p></div><time class="shrink-0 text-xs text-gray-500 dark:text-dark-400">{{ formatDateTimeToMinute(item.last_activity_at) }}</time></div></button>
        <div v-if="total > tickets.length" class="border-t border-gray-100 p-4 text-center dark:border-dark-700"><button class="btn btn-secondary btn-sm" type="button" :disabled="loadingMore" @click="loadMore">{{ t('common.loadMore') }}</button><p class="mt-2 text-xs text-gray-500 dark:text-dark-400">{{ t('supportTickets.pagination', { count: tickets.length, total }) }}</p></div>
      </section>

      <BaseDialog :show="showCreateDialog" :title="t('admin.supportTickets.createOutboundTitle')" width="normal" @close="closeCreateDialog">
        <form id="admin-create-support-ticket-form" class="space-y-4" @submit.prevent="createOutboundTicket">
          <div>
            <label class="input-label" for="admin-ticket-user-search">{{ t('admin.supportTickets.recipient') }}</label>
            <div class="relative">
              <input id="admin-ticket-user-search" v-model="userSearchKeyword" type="search" class="input pr-8" :placeholder="t('admin.supportTickets.recipientPlaceholder')" autocomplete="off" @input="onUserSearchInput" @focus="showUserResults = true" />
              <button v-if="selectedUser" class="absolute right-2 top-1/2 -translate-y-1/2 text-gray-400 transition-colors hover:text-gray-600 dark:hover:text-dark-300" type="button" :aria-label="t('admin.supportTickets.clearRecipient')" @click="clearSelectedUser"><Icon name="x" size="sm" /></button>
              <div v-if="showUserResults && (userSearchLoading || userSearchResults.length || userSearchKeyword)" class="absolute z-50 mt-1 max-h-60 w-full overflow-auto rounded-lg border border-gray-200 bg-white shadow-lg dark:border-dark-700 dark:bg-dark-800">
                <div v-if="userSearchLoading" class="px-4 py-3 text-sm text-gray-500 dark:text-dark-400">{{ t('common.loading') }}</div>
                <div v-else-if="!userSearchResults.length" class="px-4 py-3 text-sm text-gray-500 dark:text-dark-400">{{ t('common.noOptionsFound') }}</div>
                <button v-for="user in userSearchResults" v-else :key="user.id" class="block w-full px-4 py-2 text-left text-sm transition-colors hover:bg-gray-100 dark:hover:bg-dark-700" type="button" @click="selectUser(user)"><span class="font-medium text-gray-900 dark:text-white">{{ user.email }}</span><span class="ml-2 text-gray-500 dark:text-dark-400">#{{ user.id }}</span></button>
              </div>
            </div>
            <p v-if="selectedUser" class="input-hint mt-2">{{ t('admin.supportTickets.recipientSelected', { email: selectedUser.email, id: selectedUser.id }) }}</p>
          </div>
          <label class="block"><span class="input-label">{{ t('supportTickets.category') }}</span><select v-model="createForm.category" class="input"><option v-for="category in ticketCategories" :key="category" :value="category">{{ categoryLabel(category) }}</option></select></label>
          <label class="block"><span class="input-label">{{ t('supportTickets.subject') }}</span><input v-model.trim="createForm.subject" class="input" type="text" maxlength="200" required :placeholder="t('supportTickets.subjectPlaceholder')" /></label>
          <label class="block"><span class="input-label">{{ t('supportTickets.message') }}</span><textarea v-model.trim="createForm.body" class="input min-h-32 resize-y" maxlength="5000" required :placeholder="t('admin.supportTickets.outboundMessagePlaceholder')" /></label>
          <p class="input-hint text-amber-700 dark:text-amber-300">{{ t('supportTickets.sensitiveNotice') }}</p>
        </form>
        <template #footer><div class="flex justify-end gap-3"><button class="btn btn-secondary" type="button" :disabled="creating" @click="closeCreateDialog">{{ t('common.cancel') }}</button><button class="btn btn-primary" type="submit" form="admin-create-support-ticket-form" :disabled="creating || !selectedUser || !createForm.subject || !createForm.body">{{ creating ? t('common.processing') : t('admin.supportTickets.createOutbound') }}</button></div></template>
      </BaseDialog>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import { searchUsers, type SimpleUser } from '@/api/admin/usage'
import { useAppStore } from '@/stores/app'
import { formatDateTimeToMinute } from '@/utils/format'
import { addAdminSupportTicketMessage, createAdminSupportTicket, getAdminSupportTicket, listAdminSupportTickets, updateAdminSupportTicket, type SupportTicket, type SupportTicketCategory, type SupportTicketPriority, type SupportTicketSenderType, type SupportTicketStatus } from '../api'
import { ticketPriorities, ticketPriorityClass, ticketStatuses, ticketStatusClass, ticketStatusLocaleKey } from '../presentation'

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
const showCreateDialog = ref(false)
const creating = ref(false)
const userSearchKeyword = ref('')
const userSearchResults = ref<SimpleUser[]>([])
const userSearchLoading = ref(false)
const showUserResults = ref(false)
const selectedUser = ref<SimpleUser | null>(null)
const createForm = reactive({ category: 'account' as SupportTicketCategory, subject: '', body: '' })
const filters = reactive({ status: '' as SupportTicketStatus | '', priority: '' as SupportTicketPriority | '' })
const editForm = reactive({ status: 'open' as SupportTicketStatus, priority: 'normal' as SupportTicketPriority })
const ticketCategories: SupportTicketCategory[] = ['account', 'billing', 'api', 'model', 'feature', 'other']
const canSendMessage = computed(() => internalNote.value || ticket.value?.status === 'open' || ticket.value?.status === 'waiting_user')
const replyUnavailableHint = computed(() => ticket.value?.status === 'resolved' ? t('admin.supportTickets.replyRequiresReopen') : t('admin.supportTickets.replyClosed'))
let userSearchTimeout: ReturnType<typeof setTimeout> | null = null
let userSearchRequest = 0

function errorMessage(error: unknown) { return (error as { message?: string })?.message || t('supportTickets.loadFailed') }
function statusLabel(value: SupportTicketStatus) { return t(ticketStatusLocaleKey(value, 'admin')) }
function categoryLabel(value: SupportTicketCategory) { return t(`supportTickets.categories.${value}`) }
function priorityLabel(value: SupportTicketPriority) { return t(`supportTickets.priorities.${value}`) }
function senderLabel(value: SupportTicketSenderType) { return t(`supportTickets.sender.${value}`) }
function requesterLabel(value: Pick<SupportTicket, 'user_id' | 'user_name' | 'user_email'>) { const userDisplay = value.user_name?.trim() || value.user_email?.trim(); return userDisplay ? `#${value.user_id} · ${userDisplay}` : `#${value.user_id}` }
function openTicket(publicID: string) { router.push({ name: 'AdminSupportTicketDetail', params: { id: publicID } }) }
function goToList() { router.push({ name: 'AdminSupportTickets' }) }
function resetCreateForm() { createForm.category = 'account'; createForm.subject = ''; createForm.body = ''; userSearchKeyword.value = ''; userSearchResults.value = []; userSearchLoading.value = false; showUserResults.value = false; selectedUser.value = null; userSearchRequest += 1 }
function openCreateDialog() { resetCreateForm(); showCreateDialog.value = true }
function closeCreateDialog() { if (!creating.value) { showCreateDialog.value = false; resetCreateForm() } }
function clearSelectedUser() { selectedUser.value = null; userSearchKeyword.value = ''; userSearchResults.value = []; showUserResults.value = false; userSearchRequest += 1 }
function selectUser(user: SimpleUser) { if (user.deleted) return; selectedUser.value = user; userSearchKeyword.value = user.email; userSearchResults.value = []; showUserResults.value = false }
function onUserSearchInput() { if (selectedUser.value && userSearchKeyword.value !== selectedUser.value.email) selectedUser.value = null; showUserResults.value = true; debounceUserSearch() }
function debounceUserSearch() { if (userSearchTimeout) clearTimeout(userSearchTimeout); userSearchTimeout = setTimeout(searchTicketUsers, 300) }
async function searchTicketUsers() {
  const keyword = userSearchKeyword.value.trim()
  if (selectedUser.value && keyword !== selectedUser.value.email) selectedUser.value = null
  if (!keyword) { userSearchResults.value = []; userSearchLoading.value = false; return }
  const request = ++userSearchRequest
  try { userSearchLoading.value = true; const users = await searchUsers(keyword); if (request === userSearchRequest) userSearchResults.value = users.filter((user) => !user.deleted) } catch { if (request === userSearchRequest) userSearchResults.value = [] } finally { if (request === userSearchRequest) userSearchLoading.value = false }
}
async function createOutboundTicket() {
  if (!selectedUser.value) { appStore.showError(t('admin.supportTickets.recipientRequired')); return }
  try { creating.value = true; const created = await createAdminSupportTicket({ user_id: selectedUser.value.id, category: createForm.category, subject: createForm.subject, body: createForm.body }); showCreateDialog.value = false; resetCreateForm(); appStore.showSuccess(t('admin.supportTickets.createOutboundSuccess')); await router.push({ name: 'AdminSupportTicketDetail', params: { id: created.public_id } }) } catch (error) { appStore.showError(errorMessage(error)) } finally { creating.value = false }
}

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
  if (!ticket.value || !messageBody.value || !canSendMessage.value) return
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
onBeforeUnmount(() => { if (userSearchTimeout) clearTimeout(userSearchTimeout) })
</script>
