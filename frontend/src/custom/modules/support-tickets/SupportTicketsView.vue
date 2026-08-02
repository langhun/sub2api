<template>
  <AppLayout>
    <div class="mx-auto max-w-6xl space-y-6">
      <header class="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('supportTickets.title') }}</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('supportTickets.description') }}</p>
        </div>
        <button v-if="!ticket" class="btn btn-primary self-start" type="button" @click="showCreate = true"><Icon name="plus" size="sm" />{{ t('supportTickets.newTicket') }}</button>
        <button v-else class="btn btn-secondary self-start" type="button" @click="goToList"><Icon name="arrowLeft" size="sm" />{{ t('supportTickets.backToList') }}</button>
      </header>

      <div v-if="loading" class="card flex min-h-56 items-center justify-center"><LoadingSpinner /></div>
      <div v-else-if="loadError" class="card py-12 text-center">
        <Icon name="exclamationCircle" size="xl" class="mx-auto text-red-500" />
        <p class="mt-3 text-sm text-gray-600 dark:text-dark-300">{{ loadError }}</p>
        <button class="btn btn-secondary mt-4" type="button" @click="loadCurrent">{{ t('common.retry') }}</button>
      </div>

      <section v-else-if="ticket" class="grid gap-6 lg:grid-cols-[minmax(0,1fr)_260px]">
        <div class="card overflow-hidden p-0">
          <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
            <div class="flex flex-wrap items-start justify-between gap-3">
              <div class="min-w-0">
                <p class="text-xs font-medium text-gray-500 dark:text-dark-400">{{ t('supportTickets.ticketNumber', { number: ticket.public_id }) }}</p>
                <h2 class="mt-1 break-words text-lg font-semibold text-gray-900 dark:text-white">{{ ticket.subject }}</h2>
              </div>
              <span :class="['badge', ticketStatusClass(ticket.status)]">{{ statusLabel(ticket.status) }}</span>
            </div>
          </div>
          <div class="space-y-4 p-5">
            <div v-if="!ticket.messages?.length" class="py-8 text-center text-sm text-gray-500 dark:text-dark-400">{{ t('supportTickets.noMessages') }}</div>
            <article v-for="message in ticket.messages" :key="message.id" class="flex gap-3" :class="message.author_type === 'user' ? 'justify-end' : 'justify-start'">
              <div class="max-w-[88%] rounded-lg px-4 py-3 text-sm" :class="message.author_type === 'user' ? 'bg-primary-600 text-white' : 'bg-gray-100 text-gray-800 dark:bg-dark-800 dark:text-dark-100'">
                <div class="mb-1 flex items-center justify-between gap-4 text-xs opacity-75"><span>{{ senderLabel(message.author_type) }}</span><time>{{ formatDateTimeToMinute(message.created_at) }}</time></div>
                <p class="whitespace-pre-wrap break-words">{{ message.body }}</p>
              </div>
            </article>
          </div>
          <form v-if="canReply" class="border-t border-gray-100 p-5 dark:border-dark-700" @submit.prevent="sendReply">
            <label class="input-label" for="ticket-reply">{{ t('supportTickets.reply') }}</label>
            <textarea id="ticket-reply" v-model.trim="replyBody" data-testid="ticket-reply" class="input min-h-28 resize-y" :placeholder="t('supportTickets.replyPlaceholder')" maxlength="5000" required />
            <p class="input-hint mt-2 text-amber-700 dark:text-amber-300">{{ t('supportTickets.sensitiveNotice') }}</p>
            <div class="mt-3 flex justify-end"><button class="btn btn-primary" type="submit" :disabled="submitting || !replyBody">{{ submitting ? t('common.processing') : t('supportTickets.send') }}</button></div>
          </form>
        </div>
        <aside class="card h-fit p-5">
          <dl class="space-y-4 text-sm">
            <div><dt class="text-gray-500 dark:text-dark-400">{{ t('supportTickets.category') }}</dt><dd class="mt-1 font-medium text-gray-900 dark:text-white">{{ categoryLabel(ticket.category) }}</dd></div>
            <div><dt class="text-gray-500 dark:text-dark-400">{{ t('supportTickets.priority') }}</dt><dd class="mt-1"><span :class="['badge', ticketPriorityClass(ticket.priority)]">{{ priorityLabel(ticket.priority) }}</span></dd></div>
            <div><dt class="text-gray-500 dark:text-dark-400">{{ t('supportTickets.latestActivity') }}</dt><dd class="mt-1 font-medium text-gray-900 dark:text-white">{{ formatDateTimeToMinute(ticket.last_activity_at) }}</dd></div>
          </dl>
          <button v-if="ticket.status === 'resolved'" class="btn btn-secondary mt-6 w-full" type="button" :disabled="submitting" @click="reopen">{{ t('supportTickets.reopen') }}</button>
        </aside>
      </section>

      <section v-else class="card overflow-hidden p-0">
        <div v-if="tickets.length === 0" class="py-16 text-center">
          <Icon name="chatBubble" size="xl" class="mx-auto text-gray-400" />
          <p class="mt-3 text-sm text-gray-500 dark:text-dark-400">{{ t('supportTickets.empty') }}</p>
          <button class="btn btn-primary mt-4" type="button" @click="showCreate = true">{{ t('supportTickets.newTicket') }}</button>
        </div>
        <button v-for="item in tickets" v-else :key="item.public_id" type="button" class="block w-full border-b border-gray-100 px-5 py-4 text-left transition-colors last:border-b-0 hover:bg-gray-50 dark:border-dark-700 dark:hover:bg-dark-800" @click="openTicket(item.public_id)">
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div class="min-w-0"><div class="flex flex-wrap items-center gap-2"><span class="text-xs text-gray-500 dark:text-dark-400">{{ item.public_id }}</span><span :class="['badge', ticketStatusClass(item.status)]">{{ statusLabel(item.status) }}</span></div><p class="mt-1 break-words font-medium text-gray-900 dark:text-white">{{ item.subject }}</p><p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ categoryLabel(item.category) }}</p></div>
            <time class="shrink-0 text-xs text-gray-500 dark:text-dark-400">{{ formatDateTimeToMinute(item.last_activity_at) }}</time>
          </div>
        </button>
        <div v-if="total > tickets.length" class="border-t border-gray-100 p-4 text-center dark:border-dark-700"><button class="btn btn-secondary btn-sm" type="button" :disabled="loadingMore" @click="loadMore">{{ t('common.loadMore') }}</button><p class="mt-2 text-xs text-gray-500 dark:text-dark-400">{{ t('supportTickets.pagination', { count: tickets.length, total }) }}</p></div>
      </section>
    </div>

    <BaseDialog :show="showCreate" :title="t('supportTickets.newTicket')" width="wide" @close="showCreate = false">
      <form id="create-ticket-form" class="space-y-4" @submit.prevent="createTicket">
        <label class="block"><span class="input-label">{{ t('supportTickets.subject') }}</span><input v-model.trim="createForm.subject" class="input" maxlength="200" :placeholder="t('supportTickets.subjectPlaceholder')" required /></label>
        <label class="block"><span class="input-label">{{ t('supportTickets.category') }}</span><select v-model="createForm.category" class="input"><option v-for="category in ticketCategories" :key="category" :value="category">{{ categoryLabel(category) }}</option></select></label>
        <label class="block"><span class="input-label">{{ t('supportTickets.message') }}</span><textarea v-model.trim="createForm.body" class="input min-h-36 resize-y" maxlength="5000" :placeholder="t('supportTickets.messagePlaceholder')" required /></label>
        <p class="input-hint text-amber-700 dark:text-amber-300">{{ t('supportTickets.sensitiveNotice') }}</p>
      </form>
      <template #footer><div class="flex justify-end gap-3"><button class="btn btn-secondary" type="button" @click="showCreate = false">{{ t('common.cancel') }}</button><button class="btn btn-primary" type="submit" form="create-ticket-form" :disabled="submitting || !createForm.subject || !createForm.body">{{ submitting ? t('common.processing') : t('supportTickets.create') }}</button></div></template>
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
import { ticketCategories, ticketPriorityClass, ticketStatusClass } from './presentation'

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
function statusLabel(value: SupportTicketStatus) { return t(`supportTickets.statuses.${value}`) }
function categoryLabel(value: SupportTicketCategory) { return t(`supportTickets.categories.${value}`) }
function priorityLabel(value: SupportTicketPriority) { return t(`supportTickets.priorities.${value}`) }
function senderLabel(value: SupportTicketSenderType) { return t(`supportTickets.sender.${value}`) }
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
  try { loading.value = true; ticket.value = await getSupportTicket(id); loadError.value = '' } catch (error) { loadError.value = errorMessage(error) } finally { loading.value = false }
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
