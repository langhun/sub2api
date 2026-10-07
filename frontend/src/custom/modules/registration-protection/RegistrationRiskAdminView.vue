<template>
  <AppLayout>
    <div class="min-w-0 space-y-6" data-glass-content>
      <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">{{ t('admin.registrationRisk.title') }}</h1>
          <p class="mt-1 max-w-3xl text-sm text-gray-500 dark:text-gray-400">{{ t('admin.registrationRisk.description') }}</p>
        </div>
        <button type="button" class="btn btn-secondary inline-flex items-center gap-2 self-start" :disabled="loading || loadingMore || saving || dialog.show" @click="load()">
          <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
          {{ t('admin.registrationRisk.refresh') }}
        </button>
      </div>

      <div v-if="errorMessage" class="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-900/50 dark:bg-red-900/20 dark:text-red-300">
        {{ errorMessage }}
      </div>

      <div class="card overflow-hidden">
        <div v-if="loading" class="flex items-center justify-center px-6 py-16 text-sm text-gray-500 dark:text-gray-400">
          <Icon name="refresh" size="sm" class="mr-2 animate-spin" />
          {{ t('admin.registrationRisk.loading') }}
        </div>
        <div v-else-if="items.length === 0" class="px-6 py-16 text-center text-sm text-gray-500 dark:text-gray-400">
          {{ t('admin.registrationRisk.empty') }}
        </div>
        <template v-else>
          <div class="hidden overflow-x-auto md:block">
            <table class="w-full min-w-[1280px] divide-y divide-gray-100 dark:divide-dark-700">
              <thead class="bg-gray-50 dark:bg-dark-800/60">
                <tr>
                  <th v-for="heading in headings" :key="heading" scope="col" class="whitespace-nowrap px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ heading }}</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 bg-white dark:divide-dark-700 dark:bg-dark-800">
                <tr v-for="item in items" :key="item.user_id" class="align-top">
                  <td class="whitespace-nowrap px-4 py-4 text-sm font-medium text-gray-900 dark:text-white">#{{ item.user_id }}</td>
                  <td class="max-w-[220px] truncate px-4 py-4 text-sm text-gray-600 dark:text-gray-300" :title="item.email">{{ item.email || '-' }}</td>
                  <td class="px-4 py-4"><span class="inline-flex rounded-full px-2 py-1 text-xs font-semibold" :class="scoreClass(item.score)">{{ item.score }}</span></td>
                  <td class="px-4 py-4"><span class="inline-flex whitespace-nowrap rounded-full px-2 py-1 text-xs font-medium" :class="actionClass(item.action)">{{ actionLabel(item.action) }}</span></td>
                  <td class="min-w-[160px] max-w-[240px] px-4 py-4 text-xs text-gray-600 dark:text-gray-300"><div class="flex flex-wrap gap-1"><span v-for="signal in item.signals" :key="signal" class="rounded bg-gray-100 px-2 py-1 dark:bg-dark-700">{{ signalLabel(signal) }}</span><span v-if="item.signals.length === 0">{{ t('admin.registrationRisk.noSignals') }}</span></div></td>
                  <td class="min-w-[240px] max-w-[320px] px-4 py-4 text-xs text-gray-600 dark:text-gray-300"><ul v-if="item.evidence?.length" class="space-y-1"><li v-for="entry in item.evidence" :key="`${entry.signal}-${entry.detail}`" class="leading-5">{{ evidenceDetail(entry.detail) }}</li></ul><span v-else>{{ t('admin.registrationRisk.noEvidence') }}</span></td>
                  <td class="min-w-[220px] max-w-[300px] px-4 py-4 text-xs text-gray-600 dark:text-gray-300"><ul v-if="item.related_accounts?.length" class="space-y-1"><li v-for="account in item.related_accounts" :key="account.user_id" class="leading-5">#{{ account.user_id }} {{ account.email || '-' }}</li></ul><span v-else>{{ t('admin.registrationRisk.noRelatedAccounts') }}</span></td>
                  <td class="whitespace-nowrap px-4 py-4 text-xs text-gray-500 dark:text-gray-400">{{ formatDate(item.expires_at) }}</td>
                  <td class="whitespace-nowrap px-4 py-4 text-xs" :class="item.payment_unlocked ? 'text-emerald-600 dark:text-emerald-400' : 'text-gray-500 dark:text-gray-400'">{{ item.payment_unlocked ? t('admin.registrationRisk.paymentUnlocked') : t('admin.registrationRisk.paymentLocked') }}</td>
                  <td class="whitespace-nowrap px-4 py-4 text-right">
                    <div class="flex justify-end gap-2">
                      <button v-if="item.action !== 'manual_lock'" type="button" class="btn btn-secondary px-3 py-1.5 text-xs" :disabled="loadingMore || saving" @click="openOverride(item, 'lock')"><Icon name="lock" size="sm" class="mr-1" />{{ t('admin.registrationRisk.lock') }}</button>
                      <button v-if="item.action !== 'manual_unlock'" type="button" class="btn btn-secondary px-3 py-1.5 text-xs" :disabled="loadingMore || saving" @click="openOverride(item, 'unlock')"><Icon name="check" size="sm" class="mr-1" />{{ t('admin.registrationRisk.unlock') }}</button>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>

          <div class="divide-y divide-gray-100 dark:divide-dark-700 md:hidden">
            <article v-for="item in items" :key="item.user_id" class="space-y-3 p-4">
              <div class="flex items-start justify-between gap-3"><div class="min-w-0"><p class="font-semibold text-gray-900 dark:text-white">#{{ item.user_id }}</p><p class="truncate text-sm text-gray-500 dark:text-gray-400">{{ item.email || '-' }}</p></div><span class="inline-flex shrink-0 rounded-full px-2 py-1 text-xs font-semibold" :class="scoreClass(item.score)">{{ t('admin.registrationRisk.score') }} {{ item.score }}</span></div>
              <div class="flex flex-wrap gap-2 text-xs"><span class="rounded-full px-2 py-1 font-medium" :class="actionClass(item.action)">{{ actionLabel(item.action) }}</span><span class="rounded-full bg-gray-100 px-2 py-1 text-gray-600 dark:bg-dark-700 dark:text-gray-300">{{ item.payment_unlocked ? t('admin.registrationRisk.paymentUnlocked') : t('admin.registrationRisk.paymentLocked') }}</span></div>
              <div class="text-xs text-gray-500 dark:text-gray-400"><span class="font-medium">{{ t('admin.registrationRisk.signals') }}:</span> {{ item.signals.map(signalLabel).join(', ') || t('admin.registrationRisk.noSignals') }}</div>
              <div class="text-xs text-gray-500 dark:text-gray-400"><span class="font-medium">{{ t('admin.registrationRisk.evidence') }}:</span> <span v-if="item.evidence?.length">{{ item.evidence.map((entry) => evidenceDetail(entry.detail)).join('；') }}</span><span v-else>{{ t('admin.registrationRisk.noEvidence') }}</span></div>
              <div class="text-xs text-gray-500 dark:text-gray-400"><span class="font-medium">{{ t('admin.registrationRisk.relatedAccounts') }}:</span> <span v-if="item.related_accounts?.length">{{ item.related_accounts.map(formatRelatedAccount).join('、') }}</span><span v-else>{{ t('admin.registrationRisk.noRelatedAccounts') }}</span></div>
              <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.registrationRisk.expires') }}: {{ formatDate(item.expires_at) }}</p>
              <div class="flex justify-end gap-2">
                <button v-if="item.action !== 'manual_lock'" type="button" class="btn btn-secondary px-3 py-1.5 text-xs" :disabled="loadingMore || saving" @click="openOverride(item, 'lock')"><Icon name="lock" size="sm" class="mr-1" />{{ t('admin.registrationRisk.lock') }}</button>
                <button v-if="item.action !== 'manual_unlock'" type="button" class="btn btn-secondary px-3 py-1.5 text-xs" :disabled="loadingMore || saving" @click="openOverride(item, 'unlock')"><Icon name="check" size="sm" class="mr-1" />{{ t('admin.registrationRisk.unlock') }}</button>
              </div>
            </article>
          </div>
        </template>
      </div>
      <div v-if="hasMore && !loading" class="flex justify-center">
        <button type="button" class="btn btn-secondary inline-flex items-center gap-2" :disabled="loadingMore || saving || dialog.show" @click="load(true)">
          <Icon :name="loadingMore ? 'refresh' : 'chevronDown'" size="sm" :class="loadingMore ? 'animate-spin' : ''" />
          {{ loadingMore ? t('admin.registrationRisk.loading') : t('admin.registrationRisk.loadMore') }}
        </button>
      </div>
    </div>

    <BaseDialog :show="dialog.show" :title="dialog.mode === 'lock' ? t('admin.registrationRisk.lockTitle') : t('admin.registrationRisk.unlockTitle')" width="normal" @close="closeDialog">
      <div class="space-y-4">
        <p class="text-sm text-gray-600 dark:text-gray-300">{{ dialog.mode === 'lock' ? t('admin.registrationRisk.lockHint') : t('admin.registrationRisk.unlockHint') }}</p>
        <div>
          <label class="input-label" for="registration-risk-reason">{{ t('admin.registrationRisk.reason') }}</label>
          <textarea id="registration-risk-reason" v-model="dialog.reason" class="input min-h-28" :placeholder="t('admin.registrationRisk.reasonPlaceholder')" maxlength="500" required />
          <p v-if="dialog.reasonTouched && !dialog.reason.trim()" class="mt-1 text-xs text-red-600 dark:text-red-400">{{ t('admin.registrationRisk.reasonRequired') }}</p>
        </div>
      </div>
      <template #footer>
        <div class="flex justify-end gap-3"><button type="button" class="btn btn-secondary" :disabled="saving" @click="closeDialog">{{ t('admin.registrationRisk.cancel') }}</button><button type="button" class="btn btn-primary" :disabled="saving" @click="submitOverride">{{ saving ? t('common.saving') : t('admin.registrationRisk.confirm') }}</button></div>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import registrationRiskAPI from './admin-api'
import type { RegistrationRiskItem } from './admin-api'
import { useAppStore } from '@/stores/app'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'

const { t, te } = useI18n()
const appStore = useAppStore()
const items = ref<RegistrationRiskItem[]>([])
const loading = ref(false)
const loadingMore = ref(false)
const hasMore = ref(false)
const nextOffset = ref(0)
const saving = ref(false)
const errorMessage = ref('')
const dialog = reactive<{ show: boolean; mode: 'lock' | 'unlock'; userID: number | null; reason: string; reasonTouched: boolean }>({ show: false, mode: 'unlock', userID: null, reason: '', reasonTouched: false })

const headings = computed(() => [t('admin.registrationRisk.userId'), t('admin.users.email'), t('admin.registrationRisk.score'), t('admin.registrationRisk.action'), t('admin.registrationRisk.signals'), t('admin.registrationRisk.evidence'), t('admin.registrationRisk.relatedAccounts'), t('admin.registrationRisk.expires'), t('admin.registrationRisk.paymentUnlocked'), t('common.actions')])

async function load(append = false) {
  if (loading.value || loadingMore.value) return
  if (append) loadingMore.value = true
  else loading.value = true
  errorMessage.value = ''
  try {
    const offset = append ? nextOffset.value : 0
    const response = await registrationRiskAPI.list(100, offset)
    const nextItems = response.items || []
    items.value = append
      ? [...new Map([...items.value, ...nextItems].map(item => [item.user_id, item])).values()]
      : nextItems
    nextOffset.value = offset + nextItems.length
    hasMore.value = response.has_more ?? false
  } catch (error: any) {
    errorMessage.value = error?.message || t('admin.registrationRisk.loadFailed')
  } finally {
    loading.value = false
    loadingMore.value = false
  }
}

function signalLabel(signal: string) { const key = `admin.registrationRisk.signalLabels.${signal}`; return te(key) ? t(key) : signal }
function evidenceDetail(detail: string) { const key = `admin.registrationRisk.evidenceDetails.${detail}`; return te(key) ? t(key) : detail }
function formatRelatedAccount(account: { user_id: number; email: string }) { return `#${account.user_id} ${account.email || '-'}` }
function actionLabel(action: string) { if (action === 'observe' || action === 'none') return t('admin.registrationRisk.observe'); if (action === 'high') return t('admin.registrationRisk.high'); if (action === 'medium') return t('admin.registrationRisk.medium'); if (action === 'manual_lock') return t('admin.registrationRisk.manualLock'); if (action === 'manual_unlock') return t('admin.registrationRisk.manualUnlock'); return action }
function actionClass(action: string) { if (action === 'manual_unlock') return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'; if (action === 'observe' || action === 'none') return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'; return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300' }
function scoreClass(score: number) { return score >= 5 ? 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300' : 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300' }
function formatDate(value: string | null) { if (!value) return t('admin.registrationRisk.noExpiry'); const date = new Date(value); return Number.isNaN(date.getTime()) ? value : date.toLocaleString() }
function openOverride(item: RegistrationRiskItem, mode: 'lock' | 'unlock') { dialog.show = true; dialog.mode = mode; dialog.userID = item.user_id; dialog.reason = ''; dialog.reasonTouched = false }
function closeDialog() { if (!saving.value) dialog.show = false }
async function submitOverride() {
  if (saving.value) return
  dialog.reasonTouched = true
  if (!dialog.userID || !dialog.reason.trim()) return
  saving.value = true
  try {
    if (dialog.mode === 'lock') await registrationRiskAPI.lock(dialog.userID, dialog.reason.trim())
    else await registrationRiskAPI.unlock(dialog.userID, dialog.reason.trim())
    appStore.showSuccess(t(dialog.mode === 'lock' ? 'admin.registrationRisk.lockSuccess' : 'admin.registrationRisk.unlockSuccess'))
    dialog.show = false
    await load()
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.registrationRisk.operationFailed'))
  } finally { saving.value = false }
}

onMounted(() => { void load() })
</script>
