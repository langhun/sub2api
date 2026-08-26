<template>
  <AppLayout>
    <div class="mx-auto -mt-2 max-w-7xl space-y-3 md:-mt-3 lg:-mt-4">
      <header v-if="!ticket" class="flex min-h-0 justify-end">
        <button v-if="ticket" class="btn btn-secondary self-start sm:self-auto" type="button" @click="goToList">
          <Icon name="arrowLeft" size="sm" />
          {{ t('supportTickets.backToList') }}
        </button>
        <div v-else class="flex flex-wrap gap-2 self-start sm:self-auto">
          <button class="btn btn-secondary" type="button" :disabled="loading" :title="t('common.refresh')" @click="() => loadList()">
            <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
            <span class="sr-only sm:not-sr-only">{{ t('common.refresh') }}</span>
          </button>
          <button class="btn btn-primary" type="button" @click="openCreateDialog">
            <Icon name="userPlus" size="sm" />
            {{ t('admin.supportTickets.createOutbound') }}
          </button>
        </div>
      </header>

      <div v-if="loading" class="flex min-h-64 items-center justify-center rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-900">
        <LoadingSpinner />
      </div>
      <div v-else-if="loadError" class="rounded-lg border border-red-200 bg-white py-12 text-center dark:border-red-900/60 dark:bg-dark-900">
        <Icon name="exclamationCircle" size="xl" class="mx-auto text-red-500" />
        <p class="mt-3 text-sm text-gray-600 dark:text-dark-300">{{ loadError }}</p>
        <button class="btn btn-secondary mt-4" type="button" @click="loadCurrent">{{ t('common.retry') }}</button>
      </div>

      <section v-else-if="ticket" class="overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900">
        <div class="flex items-center gap-3 px-4 py-2.5 sm:px-5">
          <button class="btn btn-secondary btn-sm shrink-0" type="button" @click="goToList"><Icon name="arrowLeft" size="sm" />{{ t('supportTickets.backToList') }}</button>
          <div class="min-w-0 flex-1 sm:flex sm:items-baseline sm:gap-2">
            <span class="shrink-0 font-mono text-[11px] text-gray-400 dark:text-dark-500">{{ ticket.public_id }}</span>
            <h2 class="truncate text-base font-semibold text-gray-900 dark:text-white">{{ ticket.subject }}</h2>
            <span class="hidden truncate text-xs text-gray-500 dark:text-dark-400 sm:inline">{{ requesterLabel(ticket) }}</span>
          </div>
          <span :class="['badge shrink-0', ticketPriorityClass(ticket.priority)]">{{ priorityLabel(ticket.priority) }}</span>
        </div>

        <div :class="['flex items-center gap-2 border-y px-5 py-3 text-sm sm:px-6', ticketStatusBannerClass(ticket.status, 'admin')]">
          <Icon :name="statusIcon(ticket.status)" size="sm" class="shrink-0" />
          <span class="font-medium">{{ statusLabel(ticket.status) }}</span>
        </div>

        <div class="grid xl:grid-cols-[minmax(0,1fr)_300px]">
          <div class="min-w-0 xl:border-r xl:border-gray-200 dark:xl:border-dark-700">
            <TicketConversation
              ref="conversation"
              :messages="ticket.messages ?? []"
              own-author="admin"
              :user-label="requesterLabel(ticket)"
              :admin-label="senderLabel('admin')"
              :system-label="senderLabel('system')"
              :internal-note-label="t('admin.supportTickets.internalNote')"
              :empty-label="t('supportTickets.noMessages')"
              :new-messages-label="t('supportTickets.newMessages')"
            />

            <form class="sticky bottom-0 z-10 border-t border-gray-200 bg-white/95 p-4 shadow-[0_-8px_20px_-18px_rgba(15,23,42,0.45)] backdrop-blur sm:p-5 dark:border-dark-700 dark:bg-dark-900/95" @submit.prevent="sendMessage">
              <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <label class="input-label mb-0" for="admin-ticket-reply">{{ internalNote ? t('admin.supportTickets.internalNote') : t('admin.supportTickets.replyToUser') }}</label>
                <div class="inline-flex w-fit rounded-lg bg-gray-100 p-1 text-sm dark:bg-dark-800" role="group" :aria-label="t('admin.supportTickets.messageMode')">
                  <button
                    type="button"
                    class="rounded-md px-3 py-1.5 font-medium transition-colors"
                    :class="!internalNote ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white' : 'text-gray-500 hover:text-gray-800 dark:text-dark-400 dark:hover:text-dark-200'"
                    :aria-pressed="!internalNote"
                    @click="internalNote = false"
                  >
                    {{ t('admin.supportTickets.replyToUser') }}
                  </button>
                  <button
                    type="button"
                    class="flex items-center gap-1.5 rounded-md px-3 py-1.5 font-medium transition-colors"
                    :class="internalNote ? 'bg-amber-50 text-amber-800 shadow-sm ring-1 ring-amber-200 dark:bg-amber-900/30 dark:text-amber-200 dark:ring-amber-900/70' : 'text-gray-500 hover:text-gray-800 dark:text-dark-400 dark:hover:text-dark-200'"
                    :aria-pressed="internalNote"
                    @click="internalNote = true"
                  >
                    <Icon name="lock" size="xs" />
                    {{ t('admin.supportTickets.internalNote') }}
                  </button>
                </div>
              </div>
              <textarea
                id="admin-ticket-reply"
                v-model.trim="messageBody"
                data-testid="admin-ticket-message"
                class="input mt-3 min-h-28 resize-y"
                :class="internalNote ? 'border-amber-300 focus:border-amber-500 focus:ring-amber-500 dark:border-amber-900/80' : ''"
                :disabled="!canSendMessage"
                :placeholder="internalNote ? t('admin.supportTickets.internalNotePlaceholder') : t('supportTickets.replyPlaceholder')"
                maxlength="5000"
                required
              />
              <p v-if="!internalNote && !canSendMessage" class="mt-2 flex items-start gap-1.5 text-xs text-amber-700 dark:text-amber-300"><Icon name="exclamationTriangle" size="xs" class="mt-0.5 shrink-0" />{{ replyUnavailableHint }}</p>
              <div class="mt-3 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <p v-if="!internalNote" class="input-hint text-amber-700 dark:text-amber-300">{{ t('supportTickets.sensitiveNotice') }}</p>
                <p v-else class="input-hint flex items-center gap-1.5 text-amber-700 dark:text-amber-300"><Icon name="lock" size="xs" />{{ t('admin.supportTickets.internalNotePlaceholder') }}</p>
                <button class="btn btn-primary shrink-0" type="submit" :disabled="submitting || !messageBody || !canSendMessage">
                  {{ submitting ? t('common.processing') : internalNote ? t('admin.supportTickets.addInternalNote') : t('supportTickets.send') }}
                  <Icon name="arrowRight" size="sm" />
                </button>
              </div>
            </form>
          </div>

          <aside class="order-first border-b border-gray-200 p-5 xl:order-last xl:border-b-0 dark:border-dark-700">
            <form class="space-y-4 xl:sticky xl:top-20" @submit.prevent="updateTicket">
              <div class="flex items-center gap-2 text-sm font-semibold text-gray-900 dark:text-white"><Icon name="cog" size="sm" />{{ t('admin.supportTickets.details') }}</div>
              <label class="block"><span class="input-label">{{ t('supportTickets.status') }}</span><select v-model="editForm.status" class="input"><option v-for="status in ticketStatuses" :key="status" :value="status">{{ statusLabel(status) }}</option></select></label>
              <label class="block"><span class="input-label">{{ t('supportTickets.priority') }}</span><select v-model="editForm.priority" class="input"><option v-for="priority in ticketPriorities" :key="priority" :value="priority">{{ priorityLabel(priority) }}</option></select></label>
              <dl class="grid gap-4 border-t border-gray-200 pt-4 text-sm sm:grid-cols-2 xl:grid-cols-1 dark:border-dark-700">
                <div><dt class="flex items-center gap-1.5 text-xs text-gray-500 dark:text-dark-400"><Icon name="inbox" size="xs" />{{ t('supportTickets.category') }}</dt><dd class="mt-1.5 font-medium text-gray-900 dark:text-white">{{ categoryLabel(ticket.category) }}</dd></div>
                <div><dt class="flex items-center gap-1.5 text-xs text-gray-500 dark:text-dark-400"><Icon name="clock" size="xs" />{{ t('supportTickets.latestActivity') }}</dt><dd class="mt-1.5 font-medium text-gray-900 dark:text-white"><time :datetime="ticket.last_activity_at">{{ formatDateTimeToMinute(ticket.last_activity_at) }}</time></dd></div>
              </dl>
              <button class="btn btn-secondary w-full" type="submit" :disabled="submitting"><Icon name="check" size="sm" />{{ t('admin.supportTickets.update') }}</button>
            </form>
          </aside>
        </div>
      </section>

      <section v-else class="overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900">
        <div class="flex flex-col gap-3 border-b border-gray-200 p-4 sm:flex-row sm:items-center dark:border-dark-700">
          <div class="flex items-center gap-2 text-sm font-medium text-gray-700 dark:text-dark-200"><Icon name="filter" size="sm" />{{ t('admin.supportTickets.title') }}</div>
          <div class="grid flex-1 gap-2 sm:grid-cols-2 sm:justify-end xl:ml-auto xl:max-w-xl">
            <label><span class="sr-only">{{ t('admin.supportTickets.allStatuses') }}</span><select v-model="filters.status" class="input" :aria-label="t('supportTickets.status')" @change="applyFilters"><option value="">{{ t('admin.supportTickets.allStatuses') }}</option><option v-for="status in ticketStatuses" :key="status" :value="status">{{ statusLabel(status) }}</option></select></label>
            <label><span class="sr-only">{{ t('admin.supportTickets.allPriorities') }}</span><select v-model="filters.priority" class="input" :aria-label="t('supportTickets.priority')" @change="applyFilters"><option value="">{{ t('admin.supportTickets.allPriorities') }}</option><option v-for="priority in ticketPriorities" :key="priority" :value="priority">{{ priorityLabel(priority) }}</option></select></label>
          </div>
        </div>

        <div v-if="tickets.length === 0" class="py-16 text-center">
          <span class="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-gray-100 text-gray-400 dark:bg-dark-800 dark:text-dark-500"><Icon name="inbox" size="lg" /></span>
          <p class="mt-3 text-sm text-gray-500 dark:text-dark-400">{{ t('admin.supportTickets.noTickets') }}</p>
        </div>

        <button
          v-for="item in tickets"
          v-else
          :key="item.public_id"
          type="button"
          class="group grid w-full grid-cols-[2.5rem_minmax(0,1fr)_1rem] items-center gap-3 border-b border-gray-100 px-4 py-4 text-left transition-colors last:border-b-0 hover:bg-gray-50 sm:grid-cols-[2.75rem_minmax(0,1fr)_auto_1.25rem] sm:gap-4 sm:px-5 xl:grid-cols-[2.75rem_minmax(0,1fr)_minmax(12rem,16rem)_auto_1.25rem] dark:border-dark-800 dark:hover:bg-dark-800/70"
          :class="item.status === 'open' ? 'bg-amber-50/40 dark:bg-amber-900/10' : ''"
          @click="openTicket(item.public_id)"
        >
          <span :class="['flex h-10 w-10 items-center justify-center rounded-lg ring-1 sm:h-11 sm:w-11', ticketStatusIconClass(item.status)]"><Icon name="chat" size="sm" /></span>
          <span class="min-w-0">
            <span class="block break-words text-sm font-medium text-gray-900 dark:text-white">{{ item.subject }}</span>
            <span class="mt-1 flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1 text-xs text-gray-500 dark:text-dark-400">
              <span class="break-all font-mono text-gray-400 dark:text-dark-500">{{ item.public_id }}</span>
              <span aria-hidden="true">·</span>
              <span>{{ categoryLabel(item.category) }}</span>
              <span class="min-w-0 max-w-full [overflow-wrap:anywhere] xl:hidden">{{ requesterLabel(item) }}</span>
              <span :class="['badge sm:hidden', ticketStatusClass(item.status)]">{{ statusLabel(item.status) }}</span>
              <span :class="['badge sm:hidden', ticketPriorityClass(item.priority)]">{{ priorityLabel(item.priority) }}</span>
              <time :datetime="item.last_activity_at" class="sm:hidden">{{ formatDateTimeToMinute(item.last_activity_at) }}</time>
            </span>
          </span>
          <span class="hidden min-w-0 xl:block"><span class="block truncate text-sm font-medium text-gray-700 dark:text-dark-200">{{ requesterLabel(item) }}</span><span class="mt-1 block text-xs text-gray-400 dark:text-dark-500">{{ t('admin.supportTickets.requester') }}</span></span>
          <span class="hidden items-end text-right sm:flex sm:flex-col">
            <span class="flex items-center gap-1.5"><span :class="['badge', ticketStatusClass(item.status)]">{{ statusLabel(item.status) }}</span><span :class="['badge', ticketPriorityClass(item.priority)]">{{ priorityLabel(item.priority) }}</span></span>
            <time :datetime="item.last_activity_at" class="mt-1.5 text-xs text-gray-400 dark:text-dark-500">{{ formatDateTimeToMinute(item.last_activity_at) }}</time>
          </span>
          <Icon name="chevronRight" size="sm" class="text-gray-300 transition-transform group-hover:translate-x-0.5 group-hover:text-gray-500 dark:text-dark-600 dark:group-hover:text-dark-300" />
        </button>

        <div v-if="total > tickets.length" class="border-t border-gray-200 p-4 text-center dark:border-dark-700">
          <button class="btn btn-secondary btn-sm" type="button" :disabled="loadingMore" @click="loadMore">{{ t('common.loadMore') }}</button>
          <p class="mt-2 text-xs text-gray-500 dark:text-dark-400">{{ t('supportTickets.pagination', { count: tickets.length, total }) }}</p>
        </div>
      </section>

      <BaseDialog :show="showCreateDialog" :title="t('admin.supportTickets.createOutboundTitle')" width="normal" @close="closeCreateDialog">
        <form id="admin-create-support-ticket-form" class="space-y-4" @submit.prevent="createOutboundTicket">
          <div>
            <label class="input-label" for="admin-ticket-user-search">{{ t('admin.supportTickets.recipient') }}</label>
            <div class="relative">
              <input
                id="admin-ticket-user-search"
                v-model="userSearchKeyword"
                type="search"
                role="combobox"
                class="input pr-11"
                :placeholder="t('admin.supportTickets.recipientPlaceholder')"
                autocomplete="off"
                aria-autocomplete="list"
                aria-controls="admin-ticket-user-results"
                :aria-expanded="showUserResults"
                :aria-activedescendant="highlightedUserIndex >= 0 ? `admin-ticket-user-option-${highlightedUserIndex}` : undefined"
                @input="onUserSearchInput"
                @focus="showUserResults = true"
                @keydown.down.prevent="moveUserHighlight(1)"
                @keydown.up.prevent="moveUserHighlight(-1)"
                @keydown.enter.prevent="selectHighlightedUser"
                @keydown.esc="closeUserResults"
              />
              <button v-if="selectedUser" class="absolute right-1.5 top-1/2 flex h-8 w-8 -translate-y-1/2 items-center justify-center rounded-md text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-dark-700 dark:hover:text-dark-300" type="button" :aria-label="t('admin.supportTickets.clearRecipient')" @click="clearSelectedUser"><Icon name="x" size="sm" /></button>
              <div v-if="showUserResults && (userSearchLoading || userSearchResults.length || userSearchKeyword)" id="admin-ticket-user-results" role="listbox" class="absolute z-50 mt-1 max-h-60 w-full overflow-auto rounded-lg border border-gray-200 bg-white py-1 shadow-lg dark:border-dark-700 dark:bg-dark-800">
                <div v-if="userSearchLoading" class="px-4 py-3 text-sm text-gray-500 dark:text-dark-400">{{ t('common.loading') }}</div>
                <div v-else-if="!userSearchResults.length" class="px-4 py-3 text-sm text-gray-500 dark:text-dark-400">{{ t('common.noOptionsFound') }}</div>
                <button
                  v-for="(user, index) in userSearchResults"
                  v-else
                  :id="`admin-ticket-user-option-${index}`"
                  :key="user.id"
                  role="option"
                  :aria-selected="highlightedUserIndex === index"
                  class="block w-full px-4 py-2.5 text-left text-sm transition-colors"
                  :class="highlightedUserIndex === index ? 'bg-primary-50 dark:bg-primary-900/20' : 'hover:bg-gray-100 dark:hover:bg-dark-700'"
                  type="button"
                  @mouseenter="highlightedUserIndex = index"
                  @click="selectUser(user)"
                >
                  <span class="font-medium text-gray-900 dark:text-white">#{{ user.id }} · {{ user.email }}</span>
                </button>
              </div>
            </div>
            <p v-if="selectedUser" class="input-hint mt-2">#{{ selectedUser.id }} · {{ selectedUser.email }}</p>
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
import TicketConversation from '../components/TicketConversation.vue'
import { markSupportTicketSeen } from '../notifications'
import { ticketPriorities, ticketPriorityClass, ticketStatusBannerClass, ticketStatuses, ticketStatusClass, ticketStatusIconClass, ticketStatusLocaleKey } from '../presentation'
import { mergeSupportTicket, useSupportTicketDetailPolling } from '../ticketPolling'

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
const highlightedUserIndex = ref(-1)
const selectedUser = ref<SimpleUser | null>(null)
const createForm = reactive({ category: 'account' as SupportTicketCategory, subject: '', body: '' })
const filters = reactive({ status: '' as SupportTicketStatus | '', priority: '' as SupportTicketPriority | '' })
const editForm = reactive({ status: 'open' as SupportTicketStatus, priority: 'normal' as SupportTicketPriority })
const conversation = ref<{ scrollToLatest: (behavior?: 'auto' | 'smooth') => void } | null>(null)
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
function statusIcon(value: SupportTicketStatus): 'clock' | 'checkCircle' | 'xCircle' { return value === 'resolved' ? 'checkCircle' : value === 'closed' ? 'xCircle' : 'clock' }
function requesterLabel(value: Pick<SupportTicket, 'user_id' | 'user_name' | 'user_email'>) { const userDisplay = value.user_name?.trim() || value.user_email?.trim(); return userDisplay ? `#${value.user_id} · ${userDisplay}` : `#${value.user_id}` }
function openTicket(publicID: string) { router.push({ name: 'AdminSupportTicketDetail', params: { id: publicID } }) }
function goToList() { router.push({ name: 'AdminSupportTickets' }) }
function resetCreateForm() { createForm.category = 'account'; createForm.subject = ''; createForm.body = ''; userSearchKeyword.value = ''; userSearchResults.value = []; userSearchLoading.value = false; showUserResults.value = false; highlightedUserIndex.value = -1; selectedUser.value = null; userSearchRequest += 1 }
function openCreateDialog() { resetCreateForm(); showCreateDialog.value = true }
function closeCreateDialog() { if (!creating.value) { showCreateDialog.value = false; resetCreateForm() } }
function closeUserResults() { showUserResults.value = false; highlightedUserIndex.value = -1 }
function clearSelectedUser() { selectedUser.value = null; userSearchKeyword.value = ''; userSearchResults.value = []; closeUserResults(); userSearchRequest += 1 }
function selectUser(user: SimpleUser) { if (user.deleted) return; selectedUser.value = user; userSearchKeyword.value = user.email; userSearchResults.value = []; closeUserResults() }
function moveUserHighlight(direction: number) { if (!userSearchResults.value.length) return; showUserResults.value = true; highlightedUserIndex.value = (highlightedUserIndex.value + direction + userSearchResults.value.length) % userSearchResults.value.length }
function selectHighlightedUser() { const user = userSearchResults.value[highlightedUserIndex.value]; if (user) selectUser(user) }
function onUserSearchInput() { if (selectedUser.value && userSearchKeyword.value !== selectedUser.value.email) selectedUser.value = null; highlightedUserIndex.value = -1; showUserResults.value = true; debounceUserSearch() }
function debounceUserSearch() { if (userSearchTimeout) clearTimeout(userSearchTimeout); userSearchTimeout = setTimeout(searchTicketUsers, 300) }
async function searchTicketUsers() {
  const keyword = userSearchKeyword.value.trim()
  if (selectedUser.value && keyword !== selectedUser.value.email) selectedUser.value = null
  if (!keyword) { userSearchResults.value = []; userSearchLoading.value = false; highlightedUserIndex.value = -1; return }
  const request = ++userSearchRequest
  try { userSearchLoading.value = true; const users = await searchUsers(keyword); if (request === userSearchRequest) { userSearchResults.value = users.filter((user) => !user.deleted); highlightedUserIndex.value = userSearchResults.value.length ? 0 : -1 } } catch { if (request === userSearchRequest) { userSearchResults.value = []; highlightedUserIndex.value = -1 } } finally { if (request === userSearchRequest) userSearchLoading.value = false }
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
  try { loading.value = true; ticket.value = await getAdminSupportTicket(id); markSupportTicketSeen('admin', ticket.value); editForm.status = ticket.value.status; editForm.priority = ticket.value.priority; loadError.value = '' } catch (error) { loadError.value = errorMessage(error) } finally { loading.value = false }
}
function applyPolledTicket(next: SupportTicket) {
  const current = ticket.value
  if (!current) return
  if (editForm.status === current.status) editForm.status = next.status
  if (editForm.priority === current.priority) editForm.priority = next.priority
  mergeSupportTicket(current, next)
  markSupportTicketSeen('admin', current)
}
function loadCurrent() { const id = String(route.params.id || ''); return id ? loadTicket(id) : loadList() }
function loadMore() { return loadList(page.value + 1) }
function applyFilters() { return loadList(1) }
async function sendMessage() {
  if (!ticket.value || !messageBody.value || !canSendMessage.value) return
  try { submitting.value = true; const next = await addAdminSupportTicketMessage(ticket.value.public_id, { body: messageBody.value, internal: internalNote.value }); messageBody.value = ''; internalNote.value = false; applyPolledTicket(next); conversation.value?.scrollToLatest(); appStore.showSuccess(t('supportTickets.sendSuccess')) } catch (error) { appStore.showError(errorMessage(error)) } finally { submitting.value = false }
}
async function updateTicket() {
  if (!ticket.value) return
  try {
    submitting.value = true
    if (editForm.status !== ticket.value.status) applyPolledTicket(await updateAdminSupportTicket(ticket.value.public_id, { status: editForm.status }))
    if (editForm.priority !== ticket.value.priority) applyPolledTicket(await updateAdminSupportTicket(ticket.value.public_id, { priority: editForm.priority }))
    appStore.showSuccess(t('admin.supportTickets.updateSuccess'))
  } catch (error) { appStore.showError(errorMessage(error)) } finally { submitting.value = false }
}

watch(() => route.params.id, () => { ticket.value = null; messageBody.value = ''; internalNote.value = false; loadCurrent() })
useSupportTicketDetailPolling({
  ticketID: () => String(route.params.id || ''),
  currentTicket: ticket,
  fetchTicket: getAdminSupportTicket,
  onTicket: applyPolledTicket,
  shouldPause: () => loading.value || submitting.value,
})
onMounted(loadCurrent)
onBeforeUnmount(() => { if (userSearchTimeout) clearTimeout(userSearchTimeout) })
</script>
