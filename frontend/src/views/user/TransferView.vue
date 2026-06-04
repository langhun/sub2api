<template>
  <AppLayout>
    <div class="mx-auto max-w-2xl space-y-6">
      <!-- Current Balance Card -->
      <div class="card overflow-hidden">
        <div class="px-6 py-8 text-center">
          <div class="feature-icon feature-icon-success mb-4 inline-flex h-16 w-16 items-center justify-center rounded-xl">
            <Icon name="dollar" size="xl" class="text-current" />
          </div>
          <p class="text-sm font-medium text-[var(--muted-foreground)]">{{ t('transfer.currentBalance', '当前余额') }}</p>
          <p class="mt-2 text-4xl font-bold text-[var(--foreground)]" :title="currentBalanceDisplay.full">{{ currentBalanceDisplay.display }}</p>
        </div>
      </div>

      <!-- Stats -->
      <div v-if="stats" class="grid grid-cols-3 gap-4">
        <div class="card p-4 text-center">
          <div class="feature-icon feature-icon-warning mb-1 mx-auto flex h-8 w-8 items-center justify-center rounded-lg">
            <Icon name="arrowUp" size="sm" class="text-current" />
          </div>
          <p class="text-lg font-bold text-gray-900 dark:text-white" :title="formatMoneyTitle(stats.total_sent)">{{ formatMoney(stats.total_sent) }}</p>
          <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('transfer.totalSent', '累计转出') }}</p>
        </div>
        <div class="card p-4 text-center">
          <div class="feature-icon feature-icon-success mb-1 mx-auto flex h-8 w-8 items-center justify-center rounded-lg">
            <Icon name="arrowDown" size="sm" class="text-current" />
          </div>
          <p class="text-lg font-bold text-gray-900 dark:text-white" :title="formatMoneyTitle(stats.total_received)">{{ formatMoney(stats.total_received) }}</p>
          <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('transfer.totalReceived', '累计转入') }}</p>
        </div>
        <div class="card p-4 text-center">
          <div class="feature-icon feature-icon-purple mb-1 mx-auto flex h-8 w-8 items-center justify-center rounded-lg">
            <Icon name="creditCard" size="sm" class="text-current" />
          </div>
          <p class="text-lg font-bold text-gray-900 dark:text-white" :title="formatMoneyTitle(stats.total_fee_paid)">{{ formatMoney(stats.total_fee_paid) }}</p>
          <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('transfer.totalFee', '手续费') }}</p>
        </div>
      </div>

      <!-- Transfer Form -->
      <div class="card">
        <div class="p-6">
          <form @submit.prevent="handleTransfer" class="space-y-5">
            <!-- Receiver Search -->
            <div>
              <label class="input-label">{{ t('transfer.receiver', '接收方') }}</label>
              <div class="relative mt-1">
                <div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-4">
                  <Icon name="search" size="md" class="text-gray-400 dark:text-dark-500" />
                </div>
                <input v-model="searchQuery" type="text"
                  :placeholder="t('transfer.searchPlaceholder', '输入邮箱或用户名搜索')"
                  :disabled="submitting" class="input py-3 pl-12" @input="onSearchInput" />
              </div>

              <!-- Search Results Dropdown -->
              <div v-if="searchResults.length > 0 && !selectedUser" class="mt-1 max-h-48 overflow-y-auto rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-600 dark:bg-dark-800">
                <button v-for="u in searchResults" :key="u.id" type="button"
                  @click="selectUser(u)"
                  class="flex w-full items-center gap-3 px-4 py-2.5 text-left transition-colors hover:bg-gray-50 dark:hover:bg-dark-700">
                  <div class="feature-icon feature-icon-info flex h-8 w-8 items-center justify-center rounded-full">
                    <Icon name="user" size="sm" class="text-current" />
                  </div>
                  <div class="min-w-0 flex-1">
                    <p class="truncate text-sm font-medium text-gray-900 dark:text-white">{{ u.email }}</p>
                    <p v-if="u.username" class="truncate text-xs text-gray-500 dark:text-dark-400">{{ u.username }}</p>
                  </div>
                  <span class="text-xs text-gray-400 dark:text-dark-500">#{{ u.id }}</span>
                </button>
              </div>

              <!-- Selected User Badge -->
              <div v-if="selectedUser" class="feature-panel-info mt-2 inline-flex items-center gap-2 rounded-lg border px-3 py-2">
                <div class="feature-icon feature-icon-info flex h-6 w-6 items-center justify-center rounded-full">
                  <Icon name="user" size="xs" class="text-current" />
                </div>
                <span class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ selectedUser.email }}</span>
                <button type="button" @click="clearSelection" class="ml-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200">
                  <Icon name="x" size="xs" />
                </button>
              </div>
            </div>

            <!-- Amount -->
            <div>
              <label for="amount" class="input-label">{{ t('transfer.amount', '金额') }}</label>
              <div class="relative mt-1">
                <div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-4">
                  <Icon name="dollar" size="md" class="text-gray-400 dark:text-dark-500" />
                </div>
                <input id="amount" v-model.number="amount" type="number" step="0.01" min="0.01"
                  :max="user?.balance || 0"
                  :placeholder="t('transfer.amountPlaceholder', '输入转账金额')"
                  :disabled="submitting" class="input py-3 pl-12" @input="calcFee" />
              </div>
              <div class="mt-1 flex items-center justify-between">
                <p v-if="feePreview !== null" class="input-hint">
                  {{ t('transfer.feePreview', '手续费') }}:
                  <span :title="formatMoneyTitle(feePreview)">{{ formatMoney(feePreview) }}</span>
                  · {{ t('transfer.total', '合计扣款') }}:
                  <span :title="formatMoneyTitle(amount + feePreview)">{{ formatMoney(amount + feePreview) }}</span>
                </p>
                <button v-else type="button" class="input-hint text-gray-500 hover:text-gray-600" @click="calcFee">
                  {{ t('transfer.calcFee', '计算手续费') }}
                </button>
                <p class="text-xs text-gray-400 dark:text-dark-500">
                  {{ t('transfer.available', '可用') }}: {{ currentBalanceDisplay.display }}
                </p>
              </div>
            </div>

            <!-- Memo -->
            <div>
              <label for="memo" class="input-label">{{ t('transfer.memo', '留言') }}</label>
              <div class="relative mt-1">
                <div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-4">
                  <Icon name="chatBubble" size="md" class="text-gray-400 dark:text-dark-500" />
                </div>
                <input id="memo" v-model="memo" type="text" maxlength="200"
                  :placeholder="t('transfer.memoPlaceholder', '可选留言')"
                  :disabled="submitting" class="input py-3 pl-12" />
              </div>
            </div>

            <button type="submit" :disabled="!selectedUser || !amount || submitting" class="btn btn-secondary w-full py-3">
              <svg v-if="submitting" class="-ml-1 mr-2 h-5 w-5 animate-spin" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
              </svg>
              <Icon v-else name="checkCircle" size="md" class="mr-2" />
              {{ submitting ? t('common.saving') : t('transfer.submit', '确认转账') }}
            </button>
          </form>
        </div>
      </div>

      <!-- Transfer History -->
      <div class="card">
        <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('transfer.history', '转账记录') }}</h2>
        </div>
        <div class="p-6">
          <div v-if="loadingHistory" class="flex items-center justify-center py-8">
            <svg class="h-6 w-6 animate-spin text-gray-500" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
          </div>

          <div v-else-if="history.length > 0" class="space-y-3">
            <div v-for="item in history" :key="item.id" class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-600 dark:bg-dark-800">
              <div class="flex items-start justify-between">
                <div class="flex items-start gap-3">
                  <div :class="['feature-icon flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg', item.sender_id === user?.id ? 'feature-icon-warning' : 'feature-icon-success']">
                    <Icon :name="item.sender_id === user?.id ? 'arrowUp' : 'arrowDown'" size="sm" class="text-current" />
                  </div>
                  <div>
                    <p class="text-sm font-medium text-gray-900 dark:text-white">
                      {{ item.sender_id === user?.id ? t('transfer.sentTo', '转出至') + ' #' + item.receiver_id : t('transfer.receivedFrom', '从 #') + item.sender_id + t('transfer.received', ' 转入') }}
                    </p>
                    <p v-if="item.memo" class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">{{ item.memo }}</p>
                    <p class="mt-0.5 text-xs text-gray-400 dark:text-dark-500">{{ formatDateTime(item.created_at) }}</p>
                  </div>
                </div>
                <div class="text-right">
                  <p :class="['text-sm font-semibold', item.sender_id === user?.id ? 'text-amber-600 dark:text-amber-300' : 'text-emerald-600 dark:text-emerald-300']">
                    <span :title="formatSignedMoneyTitle(item.sender_id === user?.id ? -item.amount : item.amount)">{{ formatSignedMoney(item.sender_id === user?.id ? -item.amount : item.amount) }}</span>
                  </p>
                  <p v-if="item.fee > 0" class="text-xs text-gray-400 dark:text-dark-500">{{ t('transfer.fee', '手续费') }}: <span :title="formatMoneyTitle(item.fee)">{{ formatMoney(item.fee) }}</span></p>
                </div>
              </div>
            </div>
          </div>

          <div v-else class="empty-state py-8">
            <div class="feature-icon feature-icon-info mb-4 flex h-16 w-16 items-center justify-center rounded-xl">
              <Icon name="clock" size="xl" class="text-current" />
            </div>
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('transfer.noHistory', '暂无转账记录') }}</p>
          </div>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { transferBalance, getTransferStats, validateTransfer, getTransferHistory, searchUsers } from '@/api'
import type { TransferStats, TransferRecord, UserSearchResult } from '@/api/transfer'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { formatDateTime, formatDualDisplayAmount } from '@/utils/format'

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()
const user = computed(() => authStore.user)
const currentBalanceDisplay = computed(() =>
  formatDualDisplayAmount(user.value?.balance || 0, { currencySymbol: '$' })
)

const stats = ref<TransferStats | null>(null)
const feePreview = ref<number | null>(null)
const submitting = ref(false)
const history = ref<TransferRecord[]>([])
const loadingHistory = ref(false)

const searchQuery = ref('')
const searchResults = ref<UserSearchResult[]>([])
const selectedUser = ref<UserSearchResult | null>(null)
const amount = ref(0)
const memo = ref('')

let searchTimer: ReturnType<typeof setTimeout> | null = null

function onSearchInput() {
  if (searchTimer) clearTimeout(searchTimer)
  if (!searchQuery.value || searchQuery.value.length < 1) {
    searchResults.value = []
    return
  }
  searchTimer = setTimeout(async () => {
    try {
      searchResults.value = await searchUsers(searchQuery.value)
    } catch {
      searchResults.value = []
    }
  }, 300)
}

function selectUser(u: UserSearchResult) {
  selectedUser.value = u
  searchQuery.value = u.email
  searchResults.value = []
  feePreview.value = null
}

function clearSelection() {
  selectedUser.value = null
  searchQuery.value = ''
  searchResults.value = []
  feePreview.value = null
}

async function loadStats() {
  try {
    stats.value = await getTransferStats()
  } catch {
    // Stats are supplemental; keep the current values when refresh fails.
  }
}

async function loadHistory() {
  loadingHistory.value = true
  try {
    const res = await getTransferHistory({ page: 1, page_size: 20 })
    history.value = res.items || []
  } catch {
    // Keep the previous history if a background refresh fails.
  } finally {
    loadingHistory.value = false
  }
}

async function calcFee() {
  if (selectedUser.value && amount.value > 0) {
    try {
      const result = await validateTransfer(selectedUser.value.id, amount.value)
      feePreview.value = result.fee
    } catch {
      feePreview.value = null
    }
  }
}

async function handleTransfer() {
  if (!selectedUser.value) {
    appStore.showError(t('transfer.selectReceiver', '请选择接收方'))
    return
  }
  const total = amount.value + (feePreview.value || 0)
  if (total > (user.value?.balance || 0)) {
    appStore.showError(t('transfer.insufficient', '余额不足'))
    return
  }
  submitting.value = true
  try {
    await transferBalance(selectedUser.value.id, amount.value, memo.value || undefined)
    appStore.showSuccess(t('transfer.success', '转账成功'))
    selectedUser.value = null
    searchQuery.value = ''
    amount.value = 0
    memo.value = ''
    feePreview.value = null
    loadStats().catch(() => {})
    loadHistory().catch(() => {})
    authStore.refreshUser().catch(() => {})
  } catch (e: any) {
    appStore.showError(e?.response?.data?.error || t('transfer.failed', '转账失败'))
  } finally {
    submitting.value = false
  }
}

onMounted(() => {
  loadStats()
  loadHistory()
})

function formatMoney(value: number): string {
  return formatDualDisplayAmount(value, { currencySymbol: '$' }).display
}

function formatMoneyTitle(value: number): string {
  return formatDualDisplayAmount(value, { currencySymbol: '$' }).full
}

function formatSignedMoney(value: number): string {
  return formatDualDisplayAmount(value, { currencySymbol: '$' }).display
}

function formatSignedMoneyTitle(value: number): string {
  return formatDualDisplayAmount(value, { currencySymbol: '$' }).full
}
</script>
