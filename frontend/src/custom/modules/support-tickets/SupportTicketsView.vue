<template>
  <AppLayout>
    <div class="mx-auto max-w-7xl space-y-5">
      <header class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div class="flex items-center gap-3">
          <span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-primary-50 text-primary-600 ring-1 ring-primary-100 dark:bg-primary-900/20 dark:text-primary-300 dark:ring-primary-900/60">
            <Icon name="inbox" size="lg" />
          </span>
          <h1 class="text-xl font-semibold text-gray-900 dark:text-white">{{ t('supportTickets.title') }}</h1>
        </div>
        <button v-if="!ticket" class="btn btn-primary self-start sm:self-auto" type="button" @click="showCreate = true">
          <Icon name="plus" size="sm" />
          {{ t('supportTickets.newTicket') }}
        </button>
        <button v-else class="btn btn-secondary self-start sm:self-auto" type="button" @click="goToList">
          <Icon name="arrowLeft" size="sm" />
          {{ t('supportTickets.backToList') }}
        </button>
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
        <div class="flex flex-col gap-3 px-5 py-5 sm:flex-row sm:items-start sm:justify-between sm:px-6">
          <div class="min-w-0">
            <p class="break-all font-mono text-xs text-gray-400 dark:text-dark-500">{{ ticket.public_id }}</p>
            <h2 class="mt-1 break-words text-lg font-semibold text-gray-900 dark:text-white">{{ ticket.subject }}</h2>
          </div>
          <span :class="['badge shrink-0', ticketPriorityClass(ticket.priority)]">{{ priorityLabel(ticket.priority) }}</span>
        </div>

        <div :class="['flex items-center gap-2 border-y px-5 py-3 text-sm sm:px-6', ticketStatusBannerClass(ticket.status, 'user')]">
          <Icon :name="statusIcon(ticket.status)" size="sm" class="shrink-0" />
          <span class="font-medium">{{ statusLabel(ticket.status) }}</span>
        </div>

        <div class="grid xl:grid-cols-[minmax(0,1fr)_280px]">
          <div class="min-w-0 xl:border-r xl:border-gray-200 dark:xl:border-dark-700">
            <TicketConversation
              :messages="ticket.messages ?? []"
              own-author="user"
              :user-label="senderLabel('user')"
              :admin-label="senderLabel('admin')"
              :system-label="senderLabel('system')"
              :internal-note-label="senderLabel('system')"
              :empty-label="t('supportTickets.noMessages')"
            />

            <form v-if="canReply" class="border-t border-gray-200 bg-white p-4 sm:p-5 dark:border-dark-700 dark:bg-dark-900" @submit.prevent="sendReply">
              <label class="input-label" for="ticket-reply">{{ t('supportTickets.reply') }}</label>
              <textarea id="ticket-reply" v-model.trim="replyBody" data-testid="ticket-reply" class="input min-h-28 resize-y" :placeholder="t('supportTickets.replyPlaceholder')" maxlength="5000" required />
              <div class="mt-3 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <p class="input-hint text-amber-700 dark:text-amber-300">{{ t('supportTickets.sensitiveNotice') }}</p>
                <button class="btn btn-primary shrink-0" type="submit" :disabled="submitting || !replyBody">
                  {{ submitting ? t('common.processing') : t('supportTickets.send') }}
                  <Icon name="arrowRight" size="sm" />
                </button>
              </div>
            </form>
          </div>

          <aside class="order-first border-b border-gray-200 p-5 xl:order-last xl:border-b-0 dark:border-dark-700">
            <dl class="grid gap-4 text-sm sm:grid-cols-3 xl:grid-cols-1">
              <div>
                <dt class="flex items-center gap-1.5 text-xs text-gray-500 dark:text-dark-400"><Icon name="inbox" size="xs" />{{ t('supportTickets.category') }}</dt>
                <dd class="mt-1.5 font-medium text-gray-900 dark:text-white">{{ categoryLabel(ticket.category) }}</dd>
              </div>
              <div>
                <dt class="flex items-center gap-1.5 text-xs text-gray-500 dark:text-dark-400"><Icon name="clock" size="xs" />{{ t('supportTickets.latestActivity') }}</dt>
                <dd class="mt-1.5 font-medium text-gray-900 dark:text-white"><time :datetime="ticket.last_activity_at">{{ formatDateTimeToMinute(ticket.last_activity_at) }}</time></dd>
              </div>
              <div>
                <dt class="flex items-center gap-1.5 text-xs text-gray-500 dark:text-dark-400"><Icon name="badge" size="xs" />{{ t('supportTickets.priority') }}</dt>
                <dd class="mt-1.5 font-medium text-gray-900 dark:text-white">{{ priorityLabel(ticket.priority) }}</dd>
              </div>
            </dl>
            <button v-if="ticket.status === 'resolved'" class="btn btn-secondary mt-5 w-full" type="button" :disabled="submitting" @click="reopen">
              <Icon name="refresh" size="sm" />
              {{ t('supportTickets.reopen') }}
            </button>
          </aside>
        </div>
      </section>

      <section v-else class="overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900">
        <div class="flex items-center justify-between border-b border-gray-200 px-5 py-3 dark:border-dark-700">
          <div class="flex items-center gap-2 text-sm font-medium text-gray-700 dark:text-dark-200">
            <Icon name="inbox" size="sm" />
            <span>{{ t('supportTickets.title') }}</span>
          </div>
          <span v-if="total > 0" class="text-xs tabular-nums text-gray-400 dark:text-dark-500">{{ tickets.length }} / {{ total }}</span>
        </div>

        <div v-if="tickets.length === 0" class="py-16 text-center">
          <span class="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-gray-100 text-gray-400 dark:bg-dark-800 dark:text-dark-500"><Icon name="chat" size="lg" /></span>
          <p class="mt-3 text-sm text-gray-500 dark:text-dark-400">{{ t('supportTickets.empty') }}</p>
          <button class="btn btn-primary mt-4" type="button" @click="showCreate = true">
            <Icon name="plus" size="sm" />
            {{ t('supportTickets.newTicket') }}
          </button>
        </div>

        <button
          v-for="item in tickets"
          v-else
          :key="item.public_id"
          type="button"
          class="group grid w-full grid-cols-[2.5rem_minmax(0,1fr)_1rem] items-center gap-3 border-b border-gray-100 px-4 py-4 text-left transition-colors last:border-b-0 hover:bg-gray-50 sm:grid-cols-[2.75rem_minmax(0,1fr)_auto_1.25rem] sm:gap-4 sm:px-5 dark:border-dark-800 dark:hover:bg-dark-800/70"
          @click="openTicket(item.public_id)"
        >
          <span :class="['flex h-10 w-10 items-center justify-center rounded-lg ring-1 sm:h-11 sm:w-11', ticketStatusIconClass(item.status, 'user')]"><Icon name="chat" size="sm" /></span>
          <span class="min-w-0">
            <span class="block break-words text-sm font-medium text-gray-900 dark:text-white">{{ item.subject }}</span>
            <span class="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-gray-500 dark:text-dark-400">
              <span class="break-all font-mono text-gray-400 dark:text-dark-500">{{ item.public_id }}</span>
              <span aria-hidden="true">·</span>
              <span>{{ categoryLabel(item.category) }}</span>
              <span :class="['badge sm:hidden', ticketStatusClass(item.status, 'user')]">{{ statusLabel(item.status) }}</span>
              <time :datetime="item.last_activity_at" class="sm:hidden">{{ formatDateTimeToMinute(item.last_activity_at) }}</time>
            </span>
          </span>
          <span class="hidden items-end text-right sm:flex sm:flex-col">
            <span :class="['badge', ticketStatusClass(item.status, 'user')]">{{ statusLabel(item.status) }}</span>
            <time :datetime="item.last_activity_at" class="mt-1.5 text-xs text-gray-400 dark:text-dark-500">{{ formatDateTimeToMinute(item.last_activity_at) }}</time>
          </span>
          <Icon name="chevronRight" size="sm" class="text-gray-300 transition-transform group-hover:translate-x-0.5 group-hover:text-gray-500 dark:text-dark-600 dark:group-hover:text-dark-300" />
        </button>

        <div v-if="total > tickets.length" class="border-t border-gray-200 p-4 text-center dark:border-dark-700">
          <button class="btn btn-secondary btn-sm" type="button" :disabled="loadingMore" @click="loadMore">{{ t('common.loadMore') }}</button>
          <p class="mt-2 text-xs text-gray-500 dark:text-dark-400">{{ t('supportTickets.pagination', { count: tickets.length, total }) }}</p>
        </div>
      </section>
    </div>

    <BaseDialog :show="showCreate" :title="t('supportTickets.newTicket')" width="wide" @close="showCreate = false">
      <form id="create-ticket-form" class="space-y-4" @submit.prevent="createTicket">
        <label class="block"><span class="input-label">{{ t('supportTickets.subject') }}</span><input v-model.trim="createForm.subject" class="input" maxlength="200" :placeholder="t('supportTickets.subjectPlaceholder')" required /></label>
        <label class="block"><span class="input-label">{{ t('supportTickets.category') }}</span><select v-model="createForm.category" class="input"><option v-for="category in ticketCategories" :key="category" :value="category">{{ categoryLabel(category) }}</option></select></label>
        <label class="block"><span class="input-label">{{ t('supportTickets.message') }}</span><textarea v-model.trim="createForm.body" class="input min-h-36 resize-y" maxlength="5000" :placeholder="t('supportTickets.messagePlaceholder')" required /></label>
        <p class="input-hint text-amber-700 dark:text-amber-300">{{ t('supportTickets.sensitiveNotice') }}</p>
      </form>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button class="btn btn-secondary" type="button" @click="showCreate = false">{{ t('common.cancel') }}</button>
          <button class="btn btn-primary" type="submit" form="create-ticket-form" :disabled="submitting || !createForm.subject || !createForm.body">{{ submitting ? t('common.processing') : t('supportTickets.create') }}</button>
        </div>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import { useAppStore } from '@/stores/app'
import { formatDateTimeToMinute } from '@/utils/format'
import { addSupportTicketMessage, createSupportTicket, getSupportTicket, listSupportTickets, reopenSupportTicket, type SupportTicket, type SupportTicketCategory, type SupportTicketPriority, type SupportTicketSenderType, type SupportTicketStatus } from './api'
import TicketConversation from './components/TicketConversation.vue'
import { markSupportTicketSeen } from './notifications'
import { ticketCategories, ticketPriorityClass, ticketStatusBannerClass, ticketStatusClass, ticketStatusIconClass, ticketStatusLocaleKey } from './presentation'

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
const showCreate = ref(false)
const replyBody = ref('')
const createForm = reactive({ subject: '', category: 'other' as SupportTicketCategory, body: '' })
const canReply = computed(() => ticket.value?.status === 'open' || ticket.value?.status === 'waiting_user')

function errorMessage(error: unknown) { return (error as { message?: string })?.message || t('supportTickets.loadFailed') }
function statusLabel(value: SupportTicketStatus) { return t(ticketStatusLocaleKey(value, 'user')) }
function categoryLabel(value: SupportTicketCategory) { return t(`supportTickets.categories.${value}`) }
function priorityLabel(value: SupportTicketPriority) { return t(`supportTickets.priorities.${value}`) }
function senderLabel(value: SupportTicketSenderType) { return t(`supportTickets.sender.${value}`) }
function statusIcon(value: SupportTicketStatus): 'clock' | 'checkCircle' | 'xCircle' { return value === 'resolved' ? 'checkCircle' : value === 'closed' ? 'xCircle' : 'clock' }
function openTicket(publicID: string) { router.push({ name: 'SupportTicketDetail', params: { id: publicID } }) }
function goToList() { router.push({ name: 'SupportTickets' }) }

async function loadList(nextPage = 1) {
  const isMore = nextPage > 1
  try {
    isMore ? loadingMore.value = true : loading.value = true
    const result = await listSupportTickets({ page: nextPage, page_size: 20 })
    tickets.value = isMore ? [...tickets.value, ...result.items] : result.items
    total.value = result.total
    page.value = result.page
    loadError.value = ''
  } catch (error) { loadError.value = errorMessage(error) } finally { loading.value = false; loadingMore.value = false }
}
async function loadTicket(id: string) {
  try { loading.value = true; ticket.value = await getSupportTicket(id); markSupportTicketSeen('user', ticket.value); loadError.value = '' } catch (error) { loadError.value = errorMessage(error) } finally { loading.value = false }
}
function loadCurrent() { const id = String(route.params.id || ''); return id ? loadTicket(id) : loadList() }
function loadMore() { return loadList(page.value + 1) }
async function createTicket() {
  try { submitting.value = true; const created = await createSupportTicket({ ...createForm }); appStore.showSuccess(t('supportTickets.createSuccess')); showCreate.value = false; createForm.subject = ''; createForm.category = 'other'; createForm.body = ''; openTicket(created.public_id) } catch (error) { appStore.showError(errorMessage(error)) } finally { submitting.value = false }
}
async function sendReply() {
  if (!ticket.value || !replyBody.value) return
  try { submitting.value = true; await addSupportTicketMessage(ticket.value.public_id, { body: replyBody.value }); replyBody.value = ''; await loadTicket(ticket.value.public_id); appStore.showSuccess(t('supportTickets.sendSuccess')) } catch (error) { appStore.showError(errorMessage(error)) } finally { submitting.value = false }
}
async function reopen() {
  if (!ticket.value) return
  try { submitting.value = true; await reopenSupportTicket(ticket.value.public_id); await loadTicket(ticket.value.public_id); appStore.showSuccess(t('supportTickets.reopenSuccess')) } catch (error) { appStore.showError(errorMessage(error)) } finally { submitting.value = false }
}

watch(() => route.params.id, () => { ticket.value = null; replyBody.value = ''; loadCurrent() })
onMounted(loadCurrent)
</script>
