<template>
  <BaseDialog :show="show" :title="t('admin.users.balanceHistoryTitle')" width="wide" :close-on-click-outside="true" :z-index="40" @close="$emit('close')">
    <div v-if="displayUser" class="space-y-4">
      <!-- User header: two-row layout with full user info -->
      <div class="rounded-xl bg-gray-50 p-4 dark:bg-dark-700">
        <!-- Row 1: avatar + email/username/created_at (left) + current balance (right) -->
        <div class="flex items-center gap-3">
          <div class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-full bg-primary-100 dark:bg-primary-900/30">
            <span class="text-lg font-medium text-primary-700 dark:text-primary-300">
              {{ getUserDisplayInitial(displayUser) }}
            </span>
          </div>
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-2">
              <p class="truncate font-medium text-gray-900 dark:text-white">{{ getPreferredUserDisplayName(displayUser, '#' + displayUser.id) }}</p>
              <span v-if="displayUser.deleted_at" class="flex-shrink-0 inline-flex items-center rounded px-1 py-px text-[10px] font-medium leading-tight bg-rose-100 text-rose-600 ring-1 ring-inset ring-rose-200 dark:bg-rose-500/20 dark:text-rose-400 dark:ring-rose-500/30">
                {{ t('admin.usage.userDeletedBadge') }}
              </span>
            </div>
            <p v-if="getSecondaryUserEmail(displayUser)" class="truncate text-xs text-gray-400 dark:text-dark-500">
              {{ getSecondaryUserEmail(displayUser) }}
            </p>
            <p class="text-xs text-gray-400 dark:text-dark-500">
              {{ t('admin.users.createdAt') }}: {{ formatDateTime(displayUser.created_at) }}
            </p>
          </div>
          <!-- Current balance: prominent display on the right -->
          <div class="flex-shrink-0 text-right">
            <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.users.currentBalance') }}</p>
            <p class="text-xl font-bold text-gray-900 dark:text-white">
              ${{ displayUser.balance?.toFixed(2) || '0.00' }}
            </p>
          </div>
        </div>
        <div class="mt-2.5 space-y-2 border-t border-gray-200/60 pt-2.5 dark:border-dark-600/60">
          <div class="flex flex-col gap-1 md:flex-row md:items-center md:justify-between">
            <p class="min-w-0 flex-1 truncate text-xs text-gray-500 dark:text-dark-400" :title="displayUser.notes || ''">
              <template v-if="displayUser.notes">{{ t('admin.users.notes') }}: {{ displayUser.notes }}</template>
              <template v-else>&nbsp;</template>
            </p>
            <p class="flex-shrink-0 text-xs text-gray-500 dark:text-dark-400">
              {{ t('admin.users.totalRecharged') }}: <span class="font-semibold text-emerald-600 dark:text-emerald-400">{{ formatCurrencyAmount(totalRecharged) }}</span>
            </p>
          </div>
          <div class="flex flex-wrap gap-x-4 gap-y-1 text-xs text-gray-500 dark:text-dark-400">
            <span>{{ t('admin.users.signupSourceLabel') }}: {{ formatSignupSource(displayUser.signup_source) }}</span>
            <span>{{ t('admin.users.inviterLabel') }}: {{ inviterDisplayName || t('admin.users.noInviter') }}</span>
          </div>
          <div v-if="visibleAmountSources.length > 0" class="flex flex-wrap gap-2">
            <span
              v-for="source in visibleAmountSources"
              :key="source.key"
              class="rounded-md bg-white px-2 py-1 text-xs text-gray-600 dark:bg-dark-800 dark:text-gray-300"
            >
              {{ source.label }}: <span class="font-semibold">{{ formatAmountSourceValue(source.key, source.value) }}</span>
            </span>
          </div>
        </div>
      </div>

      <!-- Type filter + Action buttons -->
      <div class="flex items-center gap-3">
        <Select
          v-model="typeFilter"
          :options="typeOptions"
          class="w-56"
          @change="loadHistory(1)"
        />
        <!-- Deposit button - matches menu style -->
        <button
          v-if="!hideActions"
          @click="emit('deposit')"
          class="flex items-center gap-2 rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm text-gray-700 transition-colors hover:bg-gray-50 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-300 dark:hover:bg-dark-700"
        >
          <Icon name="plus" size="sm" class="text-emerald-500" :stroke-width="2" />
          {{ t('admin.users.deposit') }}
        </button>
        <!-- Withdraw button - matches menu style -->
        <button
          v-if="!hideActions"
          @click="emit('withdraw')"
          class="flex items-center gap-2 rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm text-gray-700 transition-colors hover:bg-gray-50 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-300 dark:hover:bg-dark-700"
        >
          <svg class="h-4 w-4 text-amber-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 12H4" />
          </svg>
          {{ t('admin.users.withdraw') }}
        </button>
      </div>

      <!-- Loading -->
      <div v-if="loading" class="flex justify-center py-8">
        <svg class="h-8 w-8 animate-spin text-primary-500" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
        </svg>
      </div>

      <!-- Empty state -->
      <div v-else-if="history.length === 0" class="py-8 text-center">
        <p class="text-sm text-gray-500">{{ t('admin.users.noBalanceHistory') }}</p>
      </div>

      <!-- History list -->
      <div v-else class="max-h-[28rem] space-y-3 overflow-y-auto">
        <div
          v-for="item in history"
          :key="item.id"
          :class="[
            'rounded-xl border p-4',
            getBlindboxRarityStyle(item)
              ? [getBlindboxRarityStyle(item)!.border, getBlindboxRarityStyle(item)!.bg, getBlindboxRarityStyle(item)!.darkBorder, getBlindboxRarityStyle(item)!.darkBg]
              : 'border-gray-200 bg-white dark:border-dark-600 dark:bg-dark-800'
          ]"
        >
          <div class="flex items-start justify-between">
            <div class="flex items-start gap-3">
              <div
                :class="[
                  'flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg',
                  getBlindboxRarityStyle(item)
                    ? [getBlindboxRarityStyle(item)!.badge, getBlindboxRarityStyle(item)!.badgeText, getBlindboxRarityStyle(item)!.darkBadge, getBlindboxRarityStyle(item)!.darkBadgeText]
                    : getIconBg(item)
                ]"
              >
                <Icon :name="getIconName(item)" size="sm" :class="getIconColor(item)" />
              </div>
              <div>
                <div class="flex items-center gap-2">
                  <p class="text-sm font-medium text-gray-900 dark:text-white">
                    {{ getItemTitle(item) }}
                  </p>
                  <span
                    v-if="getBlindboxRarity(item)"
                    class="rounded-full px-2 py-0.5 text-[10px] font-semibold"
                    :class="[
                      rarityColorMap[getBlindboxRarity(item)!].badge,
                      rarityColorMap[getBlindboxRarity(item)!].badgeText,
                      rarityColorMap[getBlindboxRarity(item)!].darkBadge,
                      rarityColorMap[getBlindboxRarity(item)!].darkBadgeText
                    ]"
                  >
                    {{ getRarityLabel(getBlindboxRarity(item)!) }}
                  </span>
                </div>
                <p
                  v-if="getItemDescription(item)"
                  class="mt-0.5 text-xs text-gray-500 dark:text-dark-400"
                  :title="getItemDescription(item) || ''"
                >
                  {{ getItemDescription(item) }}
                </p>
                <p class="mt-0.5 text-xs text-gray-400 dark:text-dark-500">
                  {{ formatDateTime(item.used_at || item.created_at) }}
                  <span v-if="item.type === 'checkin_luck' && item.multiplier" class="ml-1 text-amber-600 dark:text-amber-400">
                    · {{ t('checkin.multiplier') }} x{{ item.multiplier.toFixed(2) }}
                  </span>
                </p>
              </div>
            </div>
            <div class="text-right">
              <p :class="['text-sm font-semibold', getBlindboxRarityStyle(item) ? [getBlindboxRarityStyle(item)!.valueText, getBlindboxRarityStyle(item)!.darkValueText] : getValueColor(item)]">
                {{ formatValue(item) }}
              </p>
              <p
                v-if="isAdminType(item.type)"
                class="text-xs text-gray-400 dark:text-dark-500"
              >
                {{ t('redeem.adminAdjustment') }}
              </p>
              <p
                v-else
                class="font-mono text-xs text-gray-400 dark:text-dark-500"
              >
                {{ item.code.slice(0, 8) }}...
              </p>
            </div>
          </div>
        </div>
      </div>

      <!-- Pagination -->
      <div v-if="totalPages > 1" class="flex items-center justify-center gap-2 pt-2">
        <button
          :disabled="currentPage <= 1"
          class="btn btn-secondary px-3 py-1 text-sm"
          @click="loadHistory(currentPage - 1)"
        >
          {{ t('pagination.previous') }}
        </button>
        <span class="text-sm text-gray-500 dark:text-dark-400">
          {{ currentPage }} / {{ totalPages }}
        </span>
        <button
          :disabled="currentPage >= totalPages"
          class="btn btn-secondary px-3 py-1 text-sm"
          @click="loadHistory(currentPage + 1)"
        >
          {{ t('pagination.next') }}
        </button>
      </div>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI, type BalanceHistoryItem } from '@/api/admin'
import type { UserBalanceAmountSources } from '@/api/admin/users'
import { formatDateTime } from '@/utils/format'
import type { AdminUser } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import { getPreferredUserDisplayName, getSecondaryUserEmail, getUserDisplayInitial } from '@/utils/userDisplay'

const props = defineProps<{ show: boolean; user: AdminUser | null; hideActions?: boolean }>()
const emit = defineEmits(['close', 'deposit', 'withdraw'])
const { t } = useI18n()

const history = ref<BalanceHistoryItem[]>([])
const loading = ref(false)
const currentPage = ref(1)
const total = ref(0)
const totalRecharged = ref(0)
const userDetail = ref<AdminUser | null>(null)
const amountSources = ref<UserBalanceAmountSources | null>(null)
const pageSize = 15
const typeFilter = ref('')

const totalPages = computed(() => Math.ceil(total.value / pageSize) || 1)
const displayUser = computed(() => userDetail.value ?? props.user)
const inviterDisplayName = computed(() => formatHistoryUser(displayUser.value?.inviter_user))
const visibleAmountSources = computed(() => {
  if (!amountSources.value) return []

  const entries = [
    { key: 'total_credited', label: t('admin.users.amountSourceTotal'), value: amountSources.value.total_credited },
    { key: 'recharge', label: t('admin.users.amountSourceRecharge'), value: amountSources.value.recharge },
    { key: 'registration_bonus', label: t('admin.users.amountSourceRegistration'), value: amountSources.value.registration_bonus },
    { key: 'invitation_bonus', label: t('admin.users.amountSourceInvitation'), value: amountSources.value.invitation_bonus },
    { key: 'checkin_bonus', label: t('admin.users.amountSourceCheckin'), value: amountSources.value.checkin_bonus },
    { key: 'affiliate_transfer', label: t('admin.users.amountSourceAffiliateTransfer'), value: amountSources.value.affiliate_transfer },
    { key: 'admin_adjustment', label: t('admin.users.amountSourceAdminAdjustment'), value: amountSources.value.admin_adjustment },
  ]

  return entries.filter((entry) => entry.key === 'total_credited' || Math.abs(entry.value) > 0.000001)
})

// Type filter options
const typeOptions = computed(() => [
  { value: '', label: t('admin.users.allTypes') },
  { value: 'balance', label: t('admin.users.typeBalance') },
  { value: 'affiliate_balance', label: t('admin.users.typeAffiliateBalance') },
  { value: 'admin_balance', label: t('admin.users.typeAdminBalance') },
  { value: 'registration', label: t('admin.users.typeRegistration') },
  { value: 'checkin', label: t('admin.users.typeCheckin') },
  { value: 'checkin_luck', label: t('checkin.luckTitle') },
  { value: 'checkin_blindbox', label: t('admin.users.typeCheckinBlindbox') },
  { value: 'invitation', label: t('admin.users.typeInvitation') },
  { value: 'concurrency', label: t('admin.users.typeConcurrency') },
  { value: 'admin_concurrency', label: t('admin.users.typeAdminConcurrency') },
  { value: 'subscription', label: t('admin.users.typeSubscription') }
])

// Watch modal open
watch(() => props.show, (v) => {
  if (v && props.user) {
    typeFilter.value = ''
    void Promise.all([loadUserDetail(), loadHistory(1)])
  } else if (!v) {
    userDetail.value = null
    amountSources.value = null
  }
})

const loadUserDetail = async () => {
  if (!props.user) return
  userDetail.value = await adminAPI.users.getById(props.user.id)
}

const loadHistory = async (page: number) => {
  if (!props.user) return
  loading.value = true
  currentPage.value = page
  try {
    const res = await adminAPI.users.getUserBalanceHistory(
      props.user.id,
      page,
      pageSize,
      typeFilter.value || undefined
    )
    history.value = res.items || []
    total.value = res.total || 0
    totalRecharged.value = res.total_recharged || 0
    amountSources.value = res.amount_sources || null
  } catch (error) {
    console.error('Failed to load balance history:', error)
  } finally {
    loading.value = false
  }
}

const formatCurrencyAmount = (value: number) => `$${value.toFixed(2)}`

const formatAmountSourceValue = (key: string, value: number) => {
  if (key === 'admin_adjustment' || value < 0) {
    const sign = value >= 0 ? '+' : '-'
    return `${sign}$${Math.abs(value).toFixed(2)}`
  }
  return formatCurrencyAmount(value)
}

const formatSignupSource = (source?: string | null) => {
  switch ((source || '').toLowerCase()) {
    case 'linuxdo':
      return t('admin.users.signupSourceLinuxdo')
    case 'wechat':
      return t('admin.users.signupSourceWechat')
    case 'oidc':
      return t('admin.users.signupSourceOidc')
    case 'github':
      return t('admin.users.signupSourceGithub')
    case 'google':
      return t('admin.users.signupSourceGoogle')
    case 'dingtalk':
      return t('admin.users.signupSourceDingtalk')
    case 'email':
      return t('admin.users.signupSourceEmail')
    default:
      return source || t('admin.users.signupSourceUnknown')
  }
}

// Helper: check if admin type
const isAdminType = (type: string) => type === 'admin_balance' || type === 'admin_concurrency'

// Helper: check if balance-like type (cash rewards, admin adjustments, and reward credits)
const isBalanceType = (type: string) =>
  type === 'balance' ||
  type === 'admin_balance' ||
  type === 'affiliate_balance' ||
  type === 'checkin' ||
  type === 'checkin_luck' ||
  type === 'checkin_blindbox' ||
  type === 'registration' ||
  type === 'invitation'

// Helper: check if subscription type
const isSubscriptionType = (type: string) => type === 'subscription'

// Icon name based on type
const getIconName = (item: BalanceHistoryItem) => {
  if (isBlindboxInvitationCode(item)) return 'link'
  if (item.type === 'checkin' || item.type === 'checkin_luck' || item.type === 'checkin_blindbox') return 'calendar'
  if (item.type === 'registration') return 'gift'
  if (item.type === 'invitation') return 'link'
  if (isBalanceType(item.type)) return 'dollar'
  if (isSubscriptionType(item.type)) return 'badge'
  return 'bolt'
}

// Icon background color
const getIconBg = (item: BalanceHistoryItem) => {
  if (isBlindboxInvitationCode(item)) return 'bg-indigo-100 dark:bg-indigo-900/30'
  if (item.type === 'checkin' || item.type === 'checkin_luck' || item.type === 'checkin_blindbox') return 'bg-amber-100 dark:bg-amber-900/30'
  if (item.type === 'registration') return 'bg-sky-100 dark:bg-sky-900/30'
  if (item.type === 'invitation') return 'bg-rose-100 dark:bg-rose-900/30'
  if (isBalanceType(item.type)) {
    return item.value >= 0
      ? 'bg-emerald-100 dark:bg-emerald-900/30'
      : 'bg-red-100 dark:bg-red-900/30'
  }
  if (isSubscriptionType(item.type)) return 'bg-purple-100 dark:bg-purple-900/30'
  return item.value >= 0
    ? 'bg-blue-100 dark:bg-blue-900/30'
    : 'bg-orange-100 dark:bg-orange-900/30'
}

// Icon text color
const getIconColor = (item: BalanceHistoryItem) => {
  if (isBlindboxInvitationCode(item)) return 'text-indigo-600 dark:text-indigo-400'
  if (item.type === 'checkin' || item.type === 'checkin_luck' || item.type === 'checkin_blindbox') return 'text-amber-600 dark:text-amber-400'
  if (item.type === 'registration') return 'text-sky-600 dark:text-sky-400'
  if (item.type === 'invitation') return 'text-rose-600 dark:text-rose-400'
  if (isBalanceType(item.type)) {
    return item.value >= 0
      ? 'text-emerald-600 dark:text-emerald-400'
      : 'text-red-600 dark:text-red-400'
  }
  if (isSubscriptionType(item.type)) return 'text-purple-600 dark:text-purple-400'
  return item.value >= 0
    ? 'text-blue-600 dark:text-blue-400'
    : 'text-orange-600 dark:text-orange-400'
}

// Value text color
const getValueColor = (item: BalanceHistoryItem) => {
  if (isBlindboxInvitationCode(item)) return 'text-indigo-600 dark:text-indigo-400'
  if (isBlindboxConcurrency(item)) return 'text-blue-600 dark:text-blue-400'
  if (isBlindboxSubscription(item)) return 'text-purple-600 dark:text-purple-400'
  if (item.type === 'checkin' || item.type === 'checkin_luck' || item.type === 'checkin_blindbox') return 'text-amber-600 dark:text-amber-400'
  if (item.type === 'registration') return 'text-sky-600 dark:text-sky-400'
  if (item.type === 'invitation') return 'text-rose-600 dark:text-rose-400'
  if (isBalanceType(item.type)) {
    return item.value >= 0
      ? 'text-emerald-600 dark:text-emerald-400'
      : 'text-red-600 dark:text-red-400'
  }
  if (isSubscriptionType(item.type)) return 'text-purple-600 dark:text-purple-400'
  return item.value >= 0
    ? 'text-blue-600 dark:text-blue-400'
    : 'text-orange-600 dark:text-orange-400'
}

// Item title
const getItemTitle = (item: BalanceHistoryItem) => {
  switch (item.type) {
    case 'balance':
      return t('redeem.balanceAddedRedeem')
    case 'affiliate_balance':
      return t('redeem.balanceAddedAffiliate')
    case 'admin_balance':
      return item.value >= 0 ? t('redeem.balanceAddedAdmin') : t('redeem.balanceDeductedAdmin')
    case 'checkin':
      return t('admin.users.typeCheckin')
    case 'checkin_luck':
      return t('checkin.luckCheckinReward')
    case 'checkin_blindbox':
      return t('admin.users.typeCheckinBlindbox')
    case 'registration':
      return t('admin.users.typeRegistration')
    case 'invitation':
      return t('admin.users.typeInvitation')
    case 'concurrency':
      return t('redeem.concurrencyAddedRedeem')
    case 'admin_concurrency':
      return item.value >= 0 ? t('redeem.concurrencyAddedAdmin') : t('redeem.concurrencyReducedAdmin')
    case 'subscription':
      return t('redeem.subscriptionAssigned')
    default:
      return t('common.unknown')
  }
}

const isBlindboxConcurrency = (item: BalanceHistoryItem) => item.type === 'checkin_blindbox' && (item.notes?.includes('Concurrency') || item.notes?.includes('并发'))

const isBlindboxSubscription = (item: BalanceHistoryItem) => item.type === 'checkin_blindbox' && (item.notes?.includes('Subscription') || item.notes?.includes('订阅'))

const isBlindboxInvitationCode = (item: BalanceHistoryItem) => item.type === 'checkin_blindbox' && (item.notes?.includes('Invitation Code') || item.notes?.includes('邀请码'))

const formatHistoryUser = (user?: { id: number; email?: string; username?: string } | null) => {
  if (!user) return null
  const displayName = getPreferredUserDisplayName(user)
  if (displayName) return displayName
  if (user.id) return `#${user.id}`
  return null
}

const getInvitationDescription = (item: BalanceHistoryItem) => {
  if (item.type !== 'invitation') return null

  const parts: string[] = []
  if (item.source_summary) {
    parts.push(`邀请码来源：${item.source_summary}`)
  }

  const inviter = formatHistoryUser(item.inviter_user)
  if (inviter) {
    parts.push(`邀请人：${inviter}`)
  }

  return parts.length > 0 ? parts.join(' · ') : null
}

const getItemDescription = (item: BalanceHistoryItem) => {
  if (item.type === 'invitation') {
    return getInvitationDescription(item)
  }
  if (!item.notes) return ''
  return item.notes.length > 60 ? item.notes.substring(0, 55) + '...' : item.notes
}

const getBlindboxRarity = (item: BalanceHistoryItem): string | null => {
  if (item.type !== 'checkin_blindbox' || !item.notes) return null
  const parts = item.notes.split(' · ')
  if (parts.length < 2) return null
  const rarityMap: Record<string, string> = {
    Common: 'common', Rare: 'rare', Epic: 'epic', Legendary: 'legendary',
  }
  return rarityMap[parts[1]] || null
}

const rarityColorMap: Record<string, { border: string; bg: string; badge: string; badgeText: string; darkBorder: string; darkBg: string; darkBadge: string; darkBadgeText: string; valueText: string; darkValueText: string }> = {
  common: {
    border: 'border-gray-300', bg: 'bg-gray-50', badge: 'bg-gray-200', badgeText: 'text-gray-600',
    darkBorder: 'dark:border-gray-600', darkBg: 'dark:bg-gray-800/50', darkBadge: 'dark:bg-gray-700', darkBadgeText: 'dark:text-gray-400',
    valueText: 'text-gray-700', darkValueText: 'dark:text-gray-300',
  },
  rare: {
    border: 'border-blue-300', bg: 'bg-blue-50', badge: 'bg-blue-100', badgeText: 'text-blue-700',
    darkBorder: 'dark:border-blue-800', darkBg: 'dark:bg-blue-950/30', darkBadge: 'dark:bg-blue-900/50', darkBadgeText: 'dark:text-blue-300',
    valueText: 'text-blue-700', darkValueText: 'dark:text-blue-300',
  },
  epic: {
    border: 'border-purple-300', bg: 'bg-purple-50', badge: 'bg-purple-100', badgeText: 'text-purple-700',
    darkBorder: 'dark:border-purple-800', darkBg: 'dark:bg-purple-950/30', darkBadge: 'dark:bg-purple-900/50', darkBadgeText: 'dark:text-purple-300',
    valueText: 'text-purple-700', darkValueText: 'dark:text-purple-300',
  },
  legendary: {
    border: 'border-amber-300', bg: 'bg-amber-50', badge: 'bg-amber-100', badgeText: 'text-amber-700',
    darkBorder: 'dark:border-amber-800', darkBg: 'dark:bg-amber-950/30', darkBadge: 'dark:bg-amber-900/50', darkBadgeText: 'dark:text-amber-300',
    valueText: 'text-amber-700', darkValueText: 'dark:text-amber-300',
  },
}

const getBlindboxRarityStyle = (item: BalanceHistoryItem) => {
  const rarity = getBlindboxRarity(item)
  if (!rarity) return null
  return rarityColorMap[rarity]
}
const formatValue = (item: BalanceHistoryItem) => {
  if (isBlindboxInvitationCode(item)) return 'x1'
  if (isBlindboxConcurrency(item)) {
    const sign = item.value >= 0 ? '+' : ''
    return `${sign}${Math.round(item.value)}`
  }
  if (isBlindboxSubscription(item)) {
    const days = item.validity_days || Math.round(item.value)
    const groupName = item.group?.name || ''
    return groupName ? `${days}d - ${groupName}` : `${days}d`
  }
  if (isBalanceType(item.type)) {
    const sign = item.value > 0 ? '+' : item.value < 0 ? '-' : ''
    return `${sign}$${Math.abs(item.value).toFixed(2)}`
  }
  if (isSubscriptionType(item.type)) {
    const days = item.validity_days || Math.round(item.value)
    const groupName = item.group?.name || ''
    return groupName ? `${days}d - ${groupName}` : `${days}d`
  }
  const sign = item.value >= 0 ? '+' : ''
  return `${sign}${item.value}`
}

const getRarityLabel = (rarity: string) => {
  switch (rarity) {
    case 'common':
      return t('checkin.blindboxCommon')
    case 'rare':
      return t('checkin.blindboxRare')
    case 'epic':
      return t('checkin.blindboxEpic')
    case 'legendary':
      return t('checkin.blindboxLegendary')
    default:
      return rarity
  }
}
</script>
