<template>
  <AppLayout>
    <div class="mx-auto max-w-7xl space-y-5">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <div class="flex flex-wrap items-center gap-2">
            <h1 class="text-2xl font-semibold text-[var(--foreground)]">{{ t('bank.title') }}</h1>
            <span class="rounded-md border border-[var(--border)] px-2 py-1 text-xs font-medium text-[var(--muted-foreground)]">
              {{ t('bank.readonlyBadge') }}
            </span>
          </div>
          <p class="mt-1 text-sm text-[var(--muted-foreground)]">{{ t('bank.description') }}</p>
        </div>
        <button class="btn btn-secondary gap-2" type="button" :disabled="loading" @click="refreshAll">
          <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
          <span>{{ t('bank.refresh') }}</span>
        </button>
      </div>

      <section class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
        <div v-for="item in accountCards" :key="item.key" class="card p-4">
          <div class="flex items-start justify-between gap-3">
            <div>
              <p class="text-sm text-[var(--muted-foreground)]">{{ item.label }}</p>
              <p class="mt-2 break-words text-2xl font-semibold text-[var(--foreground)]" :title="item.full">
                {{ item.display }}
              </p>
            </div>
            <div :class="['rounded-lg p-2', item.tone]">
              <Icon :name="item.icon" size="md" />
            </div>
          </div>
        </div>
      </section>

      <section class="grid gap-3 md:grid-cols-3">
        <div class="card p-4">
          <p class="text-sm text-[var(--muted-foreground)]">{{ t('bank.status') }}</p>
          <p class="mt-2 text-lg font-semibold text-[var(--foreground)]">{{ statusLabel }}</p>
        </div>
        <div class="card p-4">
          <p class="text-sm text-[var(--muted-foreground)]">{{ t('bank.availableCapacity') }}</p>
          <p class="mt-2 text-lg font-semibold text-[var(--foreground)]" :title="moneyTitle(account?.available_capacity)">
            {{ formatDualMoney(account?.available_capacity) }}
          </p>
        </div>
        <div class="card p-4">
          <p class="text-sm text-[var(--muted-foreground)]">{{ t('bank.legacyMissing') }}</p>
          <p class="mt-2 text-lg font-semibold text-[var(--foreground)]">
            {{ account?.legacy_missing ? t('bank.legacyMissingYes') : t('bank.legacyMissingNo') }}
          </p>
        </div>
      </section>

      <section class="card overflow-hidden">
        <div class="border-b border-[var(--border)] p-4">
          <div class="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
            <div>
              <h2 class="text-lg font-semibold text-[var(--foreground)]">{{ t('bank.transactionLedger') }}</h2>
            </div>
            <div class="grid gap-3 sm:grid-cols-2 lg:w-[28rem]">
              <div>
                <label class="input-label">{{ t('bank.typeFilter') }}</label>
                <Select v-model="filters.type" :options="typeOptions" @change="applyFilters" />
              </div>
              <div>
                <label class="input-label">{{ t('bank.moduleFilter') }}</label>
                <Select v-model="filters.business_module" :options="moduleOptions" @change="applyFilters" />
              </div>
            </div>
          </div>
        </div>

        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-[var(--border)]">
            <thead class="bg-[var(--muted)]/40">
              <tr>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-[var(--muted-foreground)]">{{ t('bank.createdAt') }}</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-[var(--muted-foreground)]">{{ t('bank.type') }}</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-[var(--muted-foreground)]">{{ t('bank.module') }}</th>
                <th class="px-4 py-3 text-right text-xs font-medium uppercase text-[var(--muted-foreground)]">{{ t('bank.amount') }}</th>
                <th class="px-4 py-3 text-right text-xs font-medium uppercase text-[var(--muted-foreground)]">{{ t('bank.balanceAfter') }}</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-[var(--muted-foreground)]">{{ t('bank.descriptionColumn') }}</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-[var(--muted-foreground)]">{{ t('bank.reference') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-[var(--border)]">
              <tr v-if="transactionsLoading">
                <td colspan="7" class="px-4 py-10 text-center text-sm text-[var(--muted-foreground)]">{{ t('common.loading') }}</td>
              </tr>
              <tr v-else-if="transactions.length === 0">
                <td colspan="7" class="px-4 py-10 text-center text-sm text-[var(--muted-foreground)]">{{ t('bank.noTransactions') }}</td>
              </tr>
              <tr v-for="item in transactions" v-else :key="item.id" class="hover:bg-[var(--muted)]/30">
                <td class="whitespace-nowrap px-4 py-3 text-sm text-[var(--muted-foreground)]">{{ formatDateTime(item.created_at) }}</td>
                <td class="px-4 py-3 text-sm text-[var(--foreground)]">{{ txTypeLabel(item.tx_type) }}</td>
                <td class="px-4 py-3 text-sm text-[var(--muted-foreground)]">{{ moduleLabel(item.business_module) }}</td>
                <td class="whitespace-nowrap px-4 py-3 text-right text-sm font-semibold" :class="amountClass(item.amount)" :title="moneyTitle(item.amount)">
                  {{ signedMoney(item.amount) }}
                </td>
                <td class="whitespace-nowrap px-4 py-3 text-right text-sm text-[var(--foreground)]" :title="moneyTitle(item.balance_after)">
                  {{ formatMoney(item.balance_after) }}
                </td>
                <td class="max-w-sm px-4 py-3 text-sm text-[var(--foreground)]">
                  <span class="line-clamp-2" :title="item.description">{{ item.description || '-' }}</span>
                </td>
                <td class="px-4 py-3 text-sm text-[var(--muted-foreground)]">
                  <span class="font-mono text-xs">{{ referenceLabel(item) }}</span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <Pagination
          v-if="pagination.total > 0"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          @update:page="handlePageChange"
          @update:pageSize="handlePageSizeChange"
        />
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { financeAPI } from '@/api/finance'
import type { FinanceAccount, FinanceTransaction } from '@/api/finance'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'
import { formatDateTime, formatDualDisplayAmount } from '@/utils/format'

const { t, locale } = useI18n()
const appStore = useAppStore()

const account = ref<FinanceAccount | null>(null)
const transactions = ref<FinanceTransaction[]>([])
const loading = ref(false)
const transactionsLoading = ref(false)
const filters = reactive({ type: '', business_module: '' })
const pagination = reactive({ page: 1, page_size: 20, total: 0 })

const typeValues = [
  'CONSUME',
  'DEPOSIT',
  'WITHDRAW',
  'TRANSFER_OUT',
  'TRANSFER_IN',
  'SLOT_BET',
  'SLOT_WIN',
  'LOAN_BORROW',
  'LOAN_REPAY',
  'LOAN_INTEREST',
  'LEND_INVEST',
  'LEND_PROFIT',
  'REWARD',
  'REFUND',
  'FREEZE',
  'UNFREEZE',
]

const moduleValues = ['API_GATEWAY', 'PAYMENT', 'TRANSFER', 'GAME', 'LENDING', 'SYSTEM', 'FINANCIAL_HUB']

const typeOptions = computed(() => [
  { value: '', label: t('bank.allTypes') },
  ...typeValues.map((value) => ({ value, label: txTypeLabel(value) })),
])

const moduleOptions = computed(() => [
  { value: '', label: t('bank.allModules') },
  ...moduleValues.map((value) => ({ value, label: moduleLabel(value) })),
])

const totalAssetsLabel = computed(() =>
  locale.value.toLowerCase().startsWith('zh') ? '总资产' : 'Total Assets'
)

const accountCards = computed(() => [
  {
    key: 'total-assets',
    label: totalAssetsLabel.value,
    display: formatDualMoney(totalAssetsExactValue.value),
    full: moneyTitle(totalAssetsExactValue.value),
    icon: 'chartBar' as const,
    tone: 'text-[var(--foreground)] bg-[var(--muted)]',
  },
  {
    key: 'balance',
    label: t('bank.balance'),
    display: formatDualMoney(account.value?.balance),
    full: moneyTitle(account.value?.balance),
    icon: 'dollar' as const,
    tone: 'text-[var(--foreground)] bg-[var(--muted)]',
  },
  {
    key: 'frozen',
    label: t('bank.frozenAmount'),
    display: formatDualMoney(account.value?.frozen_amount),
    full: moneyTitle(account.value?.frozen_amount),
    icon: 'lock' as const,
    tone: 'text-[var(--muted-foreground)] bg-[var(--muted)]',
  },
  {
    key: 'credit',
    label: t('bank.creditLimit'),
    display: formatDualMoney(account.value?.credit_limit),
    full: moneyTitle(account.value?.credit_limit),
    icon: 'shield' as const,
    tone: 'text-sky-600 dark:text-sky-300 bg-sky-50 dark:bg-sky-950/30',
  },
  {
    key: 'debt-principal',
    label: t('bank.debtPrincipal'),
    display: formatDualMoney(account.value?.debt_principal),
    full: moneyTitle(account.value?.debt_principal),
    icon: 'document' as const,
    tone: 'text-rose-600 dark:text-rose-300 bg-rose-50 dark:bg-rose-950/30',
  },
  {
    key: 'debt-interest',
    label: t('bank.debtInterest'),
    display: formatDualMoney(account.value?.debt_interest),
    full: moneyTitle(account.value?.debt_interest),
    icon: 'chart' as const,
    tone: 'text-amber-600 dark:text-amber-300 bg-amber-50 dark:bg-amber-950/30',
  },
])

const statusLabel = computed(() => {
  const status = account.value?.status || 'ACTIVE'
  return t(`bank.statusMap.${status}`, status)
})

const totalAssetsExactValue = computed(() => addDecimalStrings(account.value?.balance, account.value?.frozen_amount))

async function refreshAll() {
  loading.value = true
  try {
    await Promise.all([loadAccount(), loadTransactions()])
  } catch {
    appStore.showError(t('bank.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function loadAccount() {
  account.value = await financeAPI.getFinanceAccount()
}

async function loadTransactions() {
  transactionsLoading.value = true
  try {
    const res = await financeAPI.getFinanceTransactions({
      page: pagination.page,
      page_size: pagination.page_size,
      type: filters.type || undefined,
      business_module: filters.business_module || undefined,
    })
    transactions.value = res.items || []
    pagination.total = res.total || 0
  } catch {
    appStore.showError(t('bank.transactionsLoadFailed'))
  } finally {
    transactionsLoading.value = false
  }
}

function applyFilters() {
  pagination.page = 1
  loadTransactions().catch(() => {})
}

function handlePageChange(page: number) {
  pagination.page = page
  loadTransactions().catch(() => {})
}

function handlePageSizeChange(size: number) {
  pagination.page_size = size
  pagination.page = 1
  loadTransactions().catch(() => {})
}

function formatMoney(value?: string | null): string {
  return `$${formatDecimalString(value)}`
}

function formatDualMoney(value?: string | number | null): string {
  return formatDualDisplayAmount(decimalToNumber(value), { currencySymbol: '$' }).display
}

function moneyTitle(value?: string | null): string {
  return formatMoney(value)
}

function signedMoney(value: string): string {
  if (isPositive(value)) {
    return `+${formatMoney(value)}`
  }
  return formatMoney(value)
}

function formatDecimalString(value?: string | null): string {
  const raw = (value || '0').trim()
  const sign = raw.startsWith('-') ? '-' : ''
  const unsigned = raw.replace(/^[+-]/, '')
  const [integer = '0', fraction = ''] = unsigned.split('.')
  const normalizedInteger = integer.replace(/^0+(?=\d)/, '') || '0'
  const groupedInteger = normalizedInteger.replace(/\B(?=(\d{3})+(?!\d))/g, ',')
  const normalizedFraction = fraction.replace(/0+$/, '')
  return `${sign}${groupedInteger}${normalizedFraction ? `.${normalizedFraction}` : ''}`
}

function decimalToNumber(value?: string | number | null): number {
  if (typeof value === 'number') {
    return Number.isFinite(value) ? value : 0
  }
  const parsed = Number((value || '0').trim())
  return Number.isFinite(parsed) ? parsed : 0
}

function addDecimalStrings(left?: string | null, right?: string | null): string {
  const scale = Math.max(decimalScale(left), decimalScale(right))
  const total = toScaledBigInt(left, scale) + toScaledBigInt(right, scale)
  return fromScaledBigInt(total, scale)
}

function decimalScale(value?: string | null): number {
  const raw = (value || '0').trim().replace(/^[+-]/, '')
  const [, fraction = ''] = raw.split('.')
  return fraction.length
}

function toScaledBigInt(value: string | null | undefined, scale: number): bigint {
  const raw = (value || '0').trim()
  const negative = raw.startsWith('-')
  const unsigned = raw.replace(/^[+-]/, '')
  const [integer = '0', fraction = ''] = unsigned.split('.')
  const normalizedInteger = integer.replace(/^0+(?=\d)/, '') || '0'
  const normalizedFraction = (fraction + '0'.repeat(scale)).slice(0, scale)
  const digits = `${normalizedInteger}${normalizedFraction}`.replace(/^0+(?=\d)/, '') || '0'
  const scaled = BigInt(digits)
  return negative ? -scaled : scaled
}

function fromScaledBigInt(value: bigint, scale: number): string {
  const negative = value < 0n
  const abs = negative ? -value : value
  if (scale === 0) {
    return `${negative ? '-' : ''}${abs.toString()}`
  }
  const raw = abs.toString().padStart(scale + 1, '0')
  const integer = raw.slice(0, -scale)
  const fraction = raw.slice(-scale).replace(/0+$/, '')
  return `${negative ? '-' : ''}${integer}${fraction ? `.${fraction}` : ''}`
}

function isPositive(value: string): boolean {
  return !value.trim().startsWith('-') && value !== '0' && value !== '0.000000000000000000'
}

function isNegative(value: string): boolean {
  return value.trim().startsWith('-')
}

function amountClass(value: string): string {
  if (isNegative(value)) return 'text-rose-600 dark:text-rose-300'
  if (isPositive(value)) return 'text-[var(--foreground)]'
  return 'text-[var(--muted-foreground)]'
}

function txTypeLabel(value: string): string {
  return t(`bank.txTypes.${value}`, value)
}

function moduleLabel(value: string): string {
  return t(`bank.modules.${value}`, value)
}

function referenceLabel(item: FinanceTransaction): string {
  if (item.reference_type && item.reference_id) {
    return `${item.reference_type}:${item.reference_id}`
  }
  return item.request_id || item.tx_id
}

onMounted(() => {
  refreshAll().catch(() => {})
})
</script>
