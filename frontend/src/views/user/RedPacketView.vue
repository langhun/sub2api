<template>
  <AppLayout>
    <div class="mx-auto max-w-2xl space-y-5">
      <section class="rounded-xl border border-[var(--border)] bg-[var(--card)] px-6 py-7 shadow-sm">
        <div class="flex items-center gap-5">
          <div class="flex h-16 w-16 flex-shrink-0 items-center justify-center rounded-xl border border-[var(--border)] bg-[var(--muted)] text-[var(--foreground)] shadow-sm">
            <Icon name="gift" size="xl" class="text-current" />
          </div>
          <div class="min-w-0 flex-1">
            <h1 class="text-2xl font-bold tracking-tight text-[var(--foreground)]">
              {{ t('redpacket.title') }}
            </h1>
            <p class="mt-1 text-sm text-[var(--muted-foreground)]">
              {{ t('redpacket.subtitle') }}
            </p>
          </div>
          <div class="rounded-xl border border-[var(--border)] bg-[var(--muted)] px-4 py-2.5 text-right">
            <p class="text-xs text-[var(--muted-foreground)]">{{ t('redpacket.currentBalance') }}</p>
            <p class="mt-1 text-2xl font-bold text-[var(--foreground)]" :title="redPacketBalanceDisplay.full">{{ redPacketBalanceDisplay.display }}</p>
          </div>
        </div>
      </section>

      <section class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <button type="button" @click="openCreate" :class="actionCardClass('create')">
          <div class="flex items-center gap-3">
            <div class="action-illustration action-illustration-create">
              <Icon name="sparkles" size="lg" class="text-current" />
            </div>
            <div class="min-w-0 flex-1 text-left">
              <p class="text-base font-semibold text-slate-900 dark:text-white">
                {{ t('redpacket.create') }}
              </p>
              <p class="mt-0.5 text-xs text-slate-500 dark:text-dark-400">
                {{ t('redpacket.createDesc') }}
              </p>
            </div>
          </div>
          <Icon name="chevronRight" size="sm" class="text-slate-400 transition-transform group-hover:translate-x-0.5" />
        </button>

        <button type="button" @click="openClaim" :class="actionCardClass('claim')">
          <div class="flex items-center gap-3">
            <div class="action-illustration action-illustration-claim">
              <Icon name="gift" size="lg" class="text-current" />
            </div>
            <div class="min-w-0 flex-1 text-left">
              <p class="text-base font-semibold text-slate-900 dark:text-white">
                {{ t('redpacket.claim') }}
              </p>
              <p class="mt-0.5 text-xs text-slate-500 dark:text-dark-400">
                {{ t('redpacket.claimDesc') }}
              </p>
            </div>
          </div>
          <Icon name="chevronRight" size="sm" class="text-slate-400 transition-transform group-hover:translate-x-0.5" />
        </button>
      </section>

      <transition name="fade">
        <section v-if="showCreate" class="card overflow-hidden">
          <div class="border-b border-rose-100 bg-rose-50/80 px-5 py-3.5 dark:border-rose-500/20 dark:bg-rose-500/10">
            <div class="flex items-center gap-2.5">
              <div class="flex h-8 w-8 items-center justify-center rounded-xl bg-rose-100 text-rose-600 dark:bg-rose-500/15 dark:text-rose-300">
                <Icon name="sparkles" size="sm" class="text-current" />
              </div>
              <p class="text-sm font-semibold text-slate-900 dark:text-white">
                {{ t('redpacket.create') }}
              </p>
            </div>
          </div>

          <div class="space-y-4 p-5">
            <form @submit.prevent="handleCreate" class="space-y-4">
              <div class="grid grid-cols-2 gap-3">
                <div>
                  <label class="input-label">{{ t('redpacket.totalAmount') }}</label>
                  <div class="relative mt-1">
                    <span class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3 text-sm text-slate-400">$</span>
                    <input v-model.number="createForm.total_amount" type="number" step="0.01" min="0.01" required :disabled="createLoading" class="input pl-7" />
                  </div>
                </div>
                <div>
                  <label class="input-label">{{ t('redpacket.count') }}</label>
                  <input v-model.number="createForm.count" type="number" min="1" max="100" required :disabled="createLoading" class="input mt-1" />
                </div>
              </div>

              <div>
                <label class="input-label">{{ t('redpacket.type') }}</label>
                <div class="mt-1 grid grid-cols-2 gap-2">
                  <button type="button" @click="createForm.redpacket_type = 'equal'" :class="typeToggleClass(createForm.redpacket_type === 'equal')">
                    {{ t('redpacket.equal') }}
                  </button>
                  <button type="button" @click="createForm.redpacket_type = 'random'" :class="typeToggleClass(createForm.redpacket_type === 'random')">
                    {{ t('redpacket.random') }}
                  </button>
                </div>
              </div>

              <div>
                <label class="input-label">{{ t('redpacket.memo') }}</label>
                <input v-model="createForm.memo" type="text" maxlength="100" :placeholder="t('redpacket.memoPlaceholder')" :disabled="createLoading" class="input mt-1 w-full" />
              </div>

              <p v-if="createError" class="rounded-xl border border-rose-100 bg-rose-50 px-3 py-2 text-sm text-rose-700 dark:border-rose-500/30 dark:bg-rose-500/10 dark:text-rose-300">
                {{ createError }}
              </p>

              <div class="flex gap-3">
                <button type="submit" :disabled="createLoading" class="btn btn-primary flex-1">
                  <svg v-if="createLoading" class="-ml-1 mr-2 h-4 w-4 animate-spin" fill="none" viewBox="0 0 24 24">
                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                  </svg>
                  {{ t('redpacket.create') }}
                </button>
                <button type="button" @click="showCreate = false" class="btn btn-secondary">
                  {{ t('common.cancel', '取消') }}
                </button>
              </div>
            </form>

            <div v-if="createdRp" class="rounded-xl border border-emerald-100 bg-emerald-50/80 p-4 dark:border-emerald-500/30 dark:bg-emerald-500/10">
              <div class="flex items-start gap-3">
                <div class="flex h-9 w-9 items-center justify-center rounded-xl bg-white text-emerald-600 shadow-sm dark:bg-dark-800 dark:text-emerald-300">
                  <Icon name="checkCircle" size="md" />
                </div>
                <div class="min-w-0 flex-1">
                  <p class="text-sm font-semibold text-emerald-700 dark:text-emerald-300">
                    {{ t('redpacket.created') }}
                  </p>
                  <div class="mt-2 flex items-center gap-2 rounded-xl bg-white px-3 py-2.5 shadow-sm dark:bg-dark-800">
                    <code class="min-w-0 flex-1 truncate font-mono text-sm font-semibold text-slate-900 dark:text-white">
                      {{ createdRp.code }}
                    </code>
                    <button type="button" @click="copyCode(createdRp.code)" class="flex h-8 w-8 items-center justify-center rounded-lg text-slate-400 transition-colors hover:bg-emerald-50 hover:text-emerald-600 dark:hover:bg-emerald-500/10">
                      <Icon :name="copiedCode === createdRp.code ? 'checkCircle' : 'copy'" size="xs" />
                    </button>
                  </div>
                  <p class="mt-1.5 text-xs text-emerald-600/80 dark:text-emerald-400/70">
                    {{ t('redpacket.shareCode') }}
                  </p>
                </div>
              </div>
            </div>
          </div>
        </section>
      </transition>

      <transition name="fade">
        <section v-if="showClaim" class="card overflow-hidden">
          <div class="border-b border-amber-100 bg-amber-50/80 px-5 py-3.5 dark:border-amber-500/20 dark:bg-amber-500/10">
            <div class="flex items-center gap-2.5">
              <div class="flex h-8 w-8 items-center justify-center rounded-xl bg-amber-100 text-amber-600 dark:bg-amber-500/15 dark:text-amber-300">
                <Icon name="gift" size="sm" class="text-current" />
              </div>
              <p class="text-sm font-semibold text-slate-900 dark:text-white">
                {{ t('redpacket.claim') }}
              </p>
            </div>
          </div>

          <div class="space-y-4 p-5">
            <form @submit.prevent="handleClaim" class="space-y-4">
              <div>
                <label class="input-label">{{ t('redpacket.code') }}</label>
                <input v-model="claimCode" type="text" required :placeholder="t('redpacket.codePlaceholder')" :disabled="claimLoading" class="input mt-1 w-full text-center font-mono text-lg tracking-[0.22em]" />
              </div>

              <div v-if="claimResult" class="rounded-xl border border-emerald-100 bg-emerald-50 p-5 text-center dark:border-emerald-500/30 dark:bg-emerald-500/10">
                <p class="text-xs uppercase tracking-wider text-emerald-600 dark:text-emerald-300">
                  {{ t('redpacket.congrats') }}
                </p>
                <p class="mt-2 text-3xl font-bold text-emerald-700 dark:text-emerald-300">
                  +{{ formatAmount(claimResult.amount) }}
                </p>
              </div>

              <p v-if="claimError" :class="claimMessageClass">
                {{ claimError }}
              </p>

              <div class="flex gap-3">
                <button type="submit" :disabled="claimLoading || !claimCode" class="btn btn-primary flex-1">
                  <svg v-if="claimLoading" class="-ml-1 mr-2 h-4 w-4 animate-spin" fill="none" viewBox="0 0 24 24">
                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                  </svg>
                  {{ t('redpacket.claim') }}
                </button>
                <button type="button" @click="showClaim = false; claimResult = null; claimError = ''" class="btn btn-secondary">
                  {{ t('common.cancel', '取消') }}
                </button>
              </div>
            </form>
          </div>
        </section>
      </transition>

      <section class="card overflow-hidden">
        <div class="flex items-center justify-between border-b border-gray-100 px-5 py-4 dark:border-dark-700">
          <h2 class="text-base font-semibold text-slate-900 dark:text-white">
            {{ t('redpacket.myPackets') }}
          </h2>
          <span v-if="totalPackets > 0" class="text-xs text-slate-400">
            {{ packetsPage * pageSize + 1 }}-{{ Math.min((packetsPage + 1) * pageSize, totalPackets) }} / {{ totalPackets }}
          </span>
        </div>

        <div class="p-4">
          <div v-if="loadingPackets" class="flex items-center justify-center py-12">
            <svg class="h-6 w-6 animate-spin text-[var(--foreground)]" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
          </div>

          <template v-else-if="myPackets.length > 0">
            <div class="space-y-3">
              <article v-for="rp in myPackets" :key="rp.id" class="overflow-hidden rounded-xl border border-gray-100 bg-white transition-all hover:border-gray-200 dark:border-dark-700 dark:bg-dark-800 dark:hover:border-dark-600">
                <div class="flex items-center gap-3.5 px-4 py-3.5">
                  <div class="packet-avatar-shell">
                    <Icon name="gift" size="md" class="text-[var(--foreground)]" />
                  </div>

                  <div class="min-w-0 flex-1">
                    <div class="flex flex-wrap items-center gap-1.5">
                      <span class="text-sm font-semibold text-slate-900 dark:text-white">
                        {{ packetTypeLabel(rp.redpacket_type) }}
                      </span>
                      <span :class="statusBadgeClass(getDisplayStatus(rp))">{{ rpStatusLabel(getDisplayStatus(rp)) }}</span>
                      <span :class="packetTypeBadgeClass(rp.redpacket_type)">{{ packetTypeBadgeLabel(rp.redpacket_type) }}</span>
                    </div>
                    <p class="mt-1 text-xs text-slate-500 dark:text-dark-400">
                      {{ packetProgressText(rp) }}
                      <template v-if="rp.expire_at"> · {{ t('redpacket.expireAt') }} {{ formatDate(rp.expire_at) }}</template>
                    </p>
                    <p v-if="rp.memo" class="mt-0.5 truncate text-xs text-slate-400 dark:text-dark-500">
                      「{{ rp.memo }}」
                    </p>
                  </div>

                  <div class="flex items-center gap-2">
                    <div class="text-right">
                      <p class="text-lg font-bold leading-none text-[var(--foreground)]">
                        {{ formatAmount(rp.total_amount) }}
                      </p>
                      <p class="mt-1 text-xs text-slate-400 dark:text-dark-500">
                        {{ t('redpacket.remainingAmount') }} {{ formatAmount(rp.remaining_amount) }}
                      </p>
                    </div>
                    <button type="button" @click="toggleDetail(rp)" :class="['flex h-7 w-7 items-center justify-center rounded-lg text-slate-400 transition-all hover:bg-slate-100 hover:text-slate-600 dark:hover:bg-dark-700 dark:hover:text-dark-300', detailExpandedId === rp.id ? 'rotate-180' : '']">
                      <Icon name="chevronDown" size="xs" />
                    </button>
                  </div>
                </div>

                <div class="px-4 pb-3">
                  <div class="rounded-lg bg-amber-50/70 px-3 py-2.5 dark:bg-amber-500/10">
                    <div class="flex items-center gap-2">
                      <code class="min-w-0 flex-1 truncate text-xs text-slate-500 dark:text-dark-400">{{ rp.code }}</code>
                      <button type="button" @click="copyCode(rp.code)" class="flex h-7 w-7 items-center justify-center rounded-md text-slate-400 transition-colors hover:bg-white hover:text-[var(--foreground)] dark:hover:bg-dark-800">
                        <Icon :name="copiedCode === rp.code ? 'checkCircle' : 'copy'" size="xs" />
                      </button>
                    </div>
                    <div class="mt-2 h-1.5 overflow-hidden rounded-full bg-white dark:bg-dark-800">
                      <div class="h-full rounded-full bg-emerald-500 transition-all duration-300" :style="{ width: `${packetClaimedPercent(rp)}%` }"></div>
                    </div>
                  </div>
                </div>

                <transition name="fade">
                  <div v-if="detailExpandedId === rp.id" class="border-t border-dashed border-amber-100 bg-amber-50/50 px-4 py-3 dark:border-amber-500/20 dark:bg-amber-500/5">
                    <div v-if="loadingDetail" class="flex items-center justify-center py-4">
                      <svg class="h-4 w-4 animate-spin text-slate-400" fill="none" viewBox="0 0 24 24">
                        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                      </svg>
                    </div>

                    <div v-else-if="detailClaims.length > 0" class="space-y-2">
                      <p class="text-xs font-medium uppercase tracking-wider text-slate-400 dark:text-dark-500">
                        {{ t('redpacket.claimDetail') }} ({{ detailClaims.length }}/{{ rp.total_count }})
                      </p>
                      <div class="space-y-1.5">
                        <div v-for="claim in detailClaims" :key="claim.id" class="flex items-center justify-between rounded-lg bg-white px-3 py-2.5 shadow-sm dark:bg-dark-800">
                          <div class="flex min-w-0 items-center gap-2.5">
                            <div class="flex h-7 w-7 items-center justify-center rounded-full bg-sky-50 text-sky-600 ring-1 ring-inset ring-sky-100 dark:bg-sky-500/10 dark:text-sky-300 dark:ring-sky-500/30">
                              <Icon name="user" size="xs" />
                            </div>
                            <div class="min-w-0">
                              <p class="truncate text-sm font-medium text-slate-700 dark:text-dark-200">
                                {{ claim.user_email || '#' + claim.user_id }}
                              </p>
                              <p class="text-xs text-slate-400 dark:text-dark-500">
                                {{ formatDateTime(claim.created_at) }}
                              </p>
                            </div>
                          </div>
                          <div class="flex flex-shrink-0 flex-col items-end gap-1">
                            <span class="text-sm font-semibold text-emerald-700 dark:text-emerald-300">
                              +{{ formatAmount(claim.amount) }}
                            </span>
                            <span v-if="isBestLuckClaim(claim, rp)" class="inline-flex items-center rounded-full bg-amber-50 px-2 py-0.5 text-[11px] font-semibold text-amber-700 ring-1 ring-inset ring-amber-200 dark:bg-amber-500/10 dark:text-amber-300 dark:ring-amber-500/30">
                              {{ t('redpacket.bestLuck') }}
                            </span>
                          </div>
                        </div>
                      </div>
                      <div v-if="getDisplayStatus(rp) === 'active' && rp.remaining_count > 0" class="rounded-lg border border-dashed border-amber-200 bg-white/70 px-3 py-2 text-center text-xs text-amber-700 dark:border-amber-500/30 dark:bg-dark-800/70 dark:text-amber-300">
                        {{ t('redpacket.waitingClaim', { n: rp.remaining_count }) }}
                      </div>
                    </div>

                    <div v-else class="px-3 py-4 text-center text-xs text-slate-400 dark:text-dark-500">
                      {{ t('redpacket.noClaimsYet') }}
                    </div>
                  </div>
                </transition>
              </article>
            </div>

            <div v-if="totalPackets > pageSize" class="mt-4 flex items-center justify-center gap-2">
              <button type="button" :disabled="packetsPage <= 0" @click="packetsPage--; loadMyPackets()" class="btn btn-secondary px-3 py-1.5 text-xs">
                <Icon name="chevronLeft" size="xs" />
              </button>
              <span class="text-xs text-slate-400">{{ packetsPage + 1 }} / {{ totalPages }}</span>
              <button type="button" :disabled="packetsPage >= totalPages - 1" @click="packetsPage++; loadMyPackets()" class="btn btn-secondary px-3 py-1.5 text-xs">
                <Icon name="chevronRight" size="xs" />
              </button>
            </div>
          </template>

          <div v-else class="py-10 text-center">
            <div class="mx-auto mb-3 flex h-14 w-14 items-center justify-center rounded-xl bg-rose-50 text-rose-500 ring-1 ring-inset ring-rose-100 dark:bg-rose-500/10 dark:text-rose-300 dark:ring-rose-500/30">
              <Icon name="gift" size="xl" class="text-current" />
            </div>
            <p class="text-sm text-slate-400 dark:text-dark-500">
              {{ t('redpacket.noPackets') }}
            </p>
          </div>
        </div>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { createRedPacket, claimRedPacket, getMyRedPackets, getRedPacketDetail } from '@/api/transfer'
import type { RedPacketRecord, RedPacketClaimRecord } from '@/api/transfer'
import { extractApiErrorCode, extractApiErrorMessage } from '@/utils/apiError'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { formatDateTime, formatDualDisplayAmount } from '@/utils/format'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const user = computed(() => authStore.user)
const redPacketBalanceDisplay = computed(() =>
  formatDualDisplayAmount(user.value?.balance || 0, { currencySymbol: '$' })
)

const showCreate = ref(false)
const showClaim = ref(false)
const claimCode = ref('')
const claimResult = ref<RedPacketClaimRecord | null>(null)
const claimError = ref('')
const claimMessageType = ref<'error' | 'info'>('error')
const claimLoading = ref(false)
const createError = ref('')
const createLoading = ref(false)
const createdRp = ref<RedPacketRecord | null>(null)
const myPackets = ref<RedPacketRecord[]>([])
const loadingPackets = ref(false)
const copiedCode = ref('')
const totalPackets = ref(0)
const packetsPage = ref(0)
const pageSize = 10

const detailExpandedId = ref<number | null>(null)
const detailClaims = ref<RedPacketClaimRecord[]>([])
const loadingDetail = ref(false)

const totalPages = computed(() => Math.max(1, Math.ceil(totalPackets.value / pageSize)))

const claimMessageClass = computed(() => [
  'rounded-xl border px-3 py-2 text-sm',
  claimMessageType.value === 'info'
    ? 'border-amber-100 bg-amber-50 text-amber-700 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-300'
    : 'border-red-100 bg-red-50 text-red-600 dark:border-red-900/30 dark:bg-red-900/20 dark:text-red-300',
])

const createForm = reactive({
  total_amount: 0,
  count: 1,
  redpacket_type: 'equal' as 'equal' | 'random',
  memo: '',
})

function formatDate(dateStr: string) {
  const d = new Date(dateStr)
  if (isNaN(d.getTime())) return ''
  const month = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  const hours = String(d.getHours()).padStart(2, '0')
  const minutes = String(d.getMinutes()).padStart(2, '0')
  return `${month}-${day} ${hours}:${minutes}`
}

function openCreate() {
  const next = !showCreate.value
  showCreate.value = next
  showClaim.value = false
  createError.value = ''
  if (!next) {
    createdRp.value = null
  }
}

function openClaim() {
  const next = !showClaim.value
  showClaim.value = next
  showCreate.value = false
  claimError.value = ''
  if (!next) {
    claimResult.value = null
    claimCode.value = ''
  }
}

function actionCardClass(kind: 'create' | 'claim') {
  const active = kind === 'create' ? showCreate.value : showClaim.value
  const inactiveTone = kind === 'create'
    ? 'border-gray-200 hover:border-gray-300 hover:bg-gray-50 dark:border-dark-700 dark:hover:border-dark-600 dark:hover:bg-dark-800'
    : 'border-gray-200 hover:border-gray-300 hover:bg-gray-50 dark:border-dark-700 dark:hover:border-dark-600 dark:hover:bg-dark-800'
  const activeTone = kind === 'create'
    ? 'border-gray-300 bg-gray-50 ring-1 ring-gray-200 dark:border-dark-500 dark:bg-dark-800 dark:ring-white/10'
    : 'border-gray-300 bg-gray-50 ring-1 ring-gray-200 dark:border-dark-500 dark:bg-dark-800 dark:ring-white/10'
  return [
    'group flex items-center justify-between rounded-xl border bg-white px-4 py-3.5 text-left transition-all dark:bg-dark-900',
    active ? activeTone : inactiveTone,
  ]
}

function typeToggleClass(active: boolean) {
  return [
    'flex items-center justify-center rounded-xl border px-3 py-2.5 text-sm font-medium transition-all',
    active
      ? 'border-[var(--foreground)] bg-[var(--foreground)] text-[var(--primary-foreground)] shadow-sm'
      : 'border-gray-200 bg-white text-slate-600 hover:border-gray-300 hover:bg-gray-50 dark:border-dark-600 dark:bg-dark-800 dark:text-dark-300 dark:hover:border-dark-500 dark:hover:bg-dark-700',
  ]
}

function getDisplayStatus(rp: RedPacketRecord): string {
  if (rp.status === 'exhausted') return 'exhausted'
  if (new Date(rp.expire_at) < new Date()) return 'expired'
  return 'active'
}

function rpStatusLabel(status: string) {
  switch (status) {
    case 'active': return t('redpacket.statusActive')
    case 'exhausted': return t('redpacket.statusExhausted')
    case 'expired': return t('redpacket.statusExpired')
    default: return status
  }
}

function statusBadgeClass(status: string) {
  const base = 'inline-flex items-center rounded-full px-2 py-0.5 text-xs font-semibold'
  switch (status) {
    case 'active': return [base, 'bg-emerald-50 text-emerald-700 ring-1 ring-inset ring-emerald-200 dark:bg-emerald-500/10 dark:text-emerald-300 dark:ring-emerald-500/30']
    case 'exhausted': return [base, 'bg-slate-100 text-slate-600 dark:bg-dark-700 dark:text-dark-300']
    case 'expired': return [base, 'bg-rose-50 text-rose-700 ring-1 ring-inset ring-rose-200 dark:bg-rose-500/10 dark:text-rose-300 dark:ring-rose-500/30']
    default: return [base, 'bg-slate-100 text-slate-600 dark:bg-dark-700 dark:text-dark-300']
  }
}

function packetTypeLabel(type: 'equal' | 'random') {
  return type === 'equal' ? t('redpacket.equal') : t('redpacket.random')
}

function packetTypeBadgeClass(type: 'equal' | 'random') {
  const base = 'inline-flex items-center rounded-full px-2 py-0.5 text-xs font-semibold'
  return [
    base,
    type === 'equal'
      ? 'bg-sky-50 text-sky-700 ring-1 ring-inset ring-sky-200 dark:bg-sky-500/10 dark:text-sky-300 dark:ring-sky-500/30'
      : 'bg-violet-50 text-violet-700 ring-1 ring-inset ring-violet-200 dark:bg-violet-500/10 dark:text-violet-300 dark:ring-violet-500/30',
  ]
}

function packetTypeBadgeLabel(type: 'equal' | 'random') {
  return type === 'equal' ? t('redpacket.equalBadge') : t('redpacket.randomBadge')
}

function packetClaimedPercent(rp: RedPacketRecord) {
  if (rp.total_count <= 0) return 0
  return Math.max(0, Math.min(100, ((rp.total_count - rp.remaining_count) / rp.total_count) * 100))
}

function packetProgressText(rp: RedPacketRecord) {
  const claimed = rp.total_count - rp.remaining_count
  return `${claimed}/${rp.total_count} ${t('redpacket.copies')}`
}

function isBestLuckClaim(claim: RedPacketClaimRecord, rp: RedPacketRecord) {
  if (rp.redpacket_type !== 'random' || detailClaims.value.length !== rp.total_count) return false
  const maxAmount = Math.max(...detailClaims.value.map(item => item.amount))
  return claim.amount === maxAmount
}

async function copyCode(code: string) {
  try {
    await navigator.clipboard.writeText(code)
    copiedCode.value = code
    setTimeout(() => { copiedCode.value = '' }, 2000)
  } catch {
    // Clipboard writes can be denied by the browser; leave the copied state unchanged.
  }
}

async function toggleDetail(rp: RedPacketRecord) {
  if (detailExpandedId.value === rp.id) {
    detailExpandedId.value = null
    detailClaims.value = []
    return
  }
  detailExpandedId.value = rp.id
  loadingDetail.value = true
  detailClaims.value = []
  try {
    const res = await getRedPacketDetail(rp.id)
    detailClaims.value = res.claims || []
  } catch {
    detailClaims.value = []
  } finally {
    loadingDetail.value = false
  }
}

async function loadMyPackets() {
  loadingPackets.value = true
  try {
    const res = await getMyRedPackets({ page: packetsPage.value + 1, page_size: pageSize })
    myPackets.value = res.items || []
    totalPackets.value = res.total || 0
  } catch {
    // Keep the previous packet list if a background refresh fails.
  } finally {
    loadingPackets.value = false
  }
}

async function handleCreate() {
  createError.value = ''
  createLoading.value = true
  try {
    createdRp.value = await createRedPacket({
      total_amount: createForm.total_amount,
      count: createForm.count,
      redpacket_type: createForm.redpacket_type,
      memo: createForm.memo || undefined,
    })
    appStore.showSuccess(t('redpacket.createdSuccess'))
    createForm.total_amount = 0
    createForm.count = 1
    createForm.redpacket_type = 'equal'
    createForm.memo = ''
    loadMyPackets().catch(() => {})
    authStore.refreshUser().catch(() => {})
  } catch (e: any) {
    createError.value = e?.response?.data?.error || t('redpacket.createFailed')
  } finally {
    createLoading.value = false
  }
}

async function handleClaim() {
  claimError.value = ''
  claimMessageType.value = 'error'
  claimResult.value = null
  claimLoading.value = true
  try {
    claimResult.value = await claimRedPacket(claimCode.value)
    appStore.showSuccess(t('redpacket.claimSuccess'))
    loadMyPackets().catch(() => {})
    authStore.refreshUser().catch(() => {})
  } catch (e: any) {
    if (extractApiErrorCode(e) === 'REDPACKET_EXHAUSTED') {
      claimMessageType.value = 'info'
      claimError.value = t('redpacket.claimExhausted')
      appStore.showInfo(t('redpacket.claimExhausted'))
      loadMyPackets().catch(() => {})
    } else {
      claimError.value = extractApiErrorMessage(e, t('redpacket.claimFailed'))
    }
  } finally {
    claimLoading.value = false
  }
}

onMounted(loadMyPackets)

function formatAmount(value: number): string {
  return formatDualDisplayAmount(value, { currencySymbol: '$' }).display
}
</script>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: all 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
  transform: translateY(-6px);
}

.action-illustration {
  display: flex;
  height: 3rem;
  width: 3rem;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  border-radius: 0.75rem;
}

.action-illustration-create {
  background: var(--muted);
  border: 1px solid var(--border);
  color: var(--foreground);
}

.action-illustration-claim {
  background: var(--muted);
  border: 1px solid var(--border);
  color: var(--foreground);
}

.packet-avatar-shell {
  display: flex;
  height: 2.5rem;
  width: 2.5rem;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  border-radius: 0.75rem;
  background: var(--muted);
  border: 1px solid var(--border);
  color: var(--foreground);
}

:global(.dark) .action-illustration-create,
:global(.dark) .packet-avatar-shell {
  background: var(--muted);
  border-color: var(--border);
  color: var(--foreground);
}

:global(.dark) .action-illustration-claim {
  background: var(--muted);
  border-color: var(--border);
  color: var(--foreground);
}
</style>
