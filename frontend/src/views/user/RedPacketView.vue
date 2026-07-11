<template>
  <AppLayout>
    <div class="mx-auto max-w-4xl space-y-5">
      <section
        class="relative overflow-hidden rounded-2xl bg-gradient-to-r from-rose-500 via-red-500 to-orange-500 px-6 py-7 text-white shadow-lg shadow-rose-500/15 sm:px-8"
      >
        <div class="relative z-10 flex flex-col gap-6 sm:flex-row sm:items-center sm:justify-between">
          <div class="flex items-center gap-4">
            <div class="flex h-16 w-16 shrink-0 items-center justify-center rounded-2xl bg-white/15 ring-1 ring-inset ring-white/20">
              <Icon name="gift" size="xl" />
            </div>
            <div>
              <h1 class="text-2xl font-bold">{{ t('redpacket.title') }}</h1>
              <p class="mt-1 text-sm text-white/85">{{ t('redpacket.description') }}</p>
            </div>
          </div>
          <div class="text-left sm:text-right">
            <p class="text-xs text-white/80">{{ t('transfer.currentBalance') }}</p>
            <p class="mt-1 max-w-full truncate text-2xl font-bold tabular-nums sm:text-3xl" :title="money(currentBalance)">
              {{ money(currentBalance) }}
            </p>
          </div>
        </div>
        <span class="absolute right-20 top-5 h-2 w-2 rounded-full bg-white/70" />
        <span class="absolute bottom-6 right-48 h-1.5 w-1.5 rounded-full bg-white/70" />
      </section>

      <section class="grid gap-3 sm:grid-cols-2">
        <button type="button" class="card flex items-center gap-4 p-4 text-left hover:border-rose-200 hover:shadow-card-hover dark:hover:border-rose-900/60" @click="openCreateDialog">
          <span class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-rose-50 text-rose-500 dark:bg-rose-900/20 dark:text-rose-400">
            <Icon name="sparkles" size="md" />
          </span>
          <span class="min-w-0 flex-1">
            <strong class="block text-base text-gray-900 dark:text-white">{{ t('redpacket.create') }}</strong>
            <span class="mt-0.5 block text-sm text-gray-500 dark:text-dark-400">{{ t('redpacket.createHint') }}</span>
          </span>
          <Icon name="chevronRight" size="sm" class="text-gray-400" />
        </button>

        <button type="button" class="card flex items-center gap-4 p-4 text-left hover:border-amber-200 hover:shadow-card-hover dark:hover:border-amber-900/60" @click="openClaimDialog">
          <span class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-amber-50 text-amber-500 dark:bg-amber-900/20 dark:text-amber-400">
            <Icon name="gift" size="md" />
          </span>
          <span class="min-w-0 flex-1">
            <strong class="block text-base text-gray-900 dark:text-white">{{ t('redpacket.claim') }}</strong>
            <span class="mt-0.5 block text-sm text-gray-500 dark:text-dark-400">{{ t('redpacket.claimHint') }}</span>
          </span>
          <Icon name="chevronRight" size="sm" class="text-gray-400" />
        </button>
      </section>

      <section class="card overflow-hidden p-0">
        <div class="flex flex-col gap-3 border-b border-gray-100 px-5 py-4 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
          <div class="flex items-center gap-3">
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('redpacket.myPackets') }}</h2>
            <div class="flex rounded-lg bg-gray-100 p-0.5 dark:bg-dark-800">
              <button
                v-for="role in historyRoles"
                :key="role.value"
                type="button"
                class="rounded-md px-2.5 py-1 text-xs font-medium transition-colors"
                :class="historyRole === role.value ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white' : 'text-gray-500 dark:text-dark-400'"
                @click="changeHistoryRole(role.value)"
              >
                {{ role.label }}
              </button>
            </div>
          </div>
          <div class="flex items-center gap-3 text-xs text-gray-400 dark:text-dark-500">
            <span v-if="total > 0">{{ pageRange }}</span>
            <button type="button" class="rounded-lg p-1.5 hover:bg-gray-100 hover:text-gray-700 dark:hover:bg-dark-800 dark:hover:text-white" :title="t('common.refresh')" @click="loadPackets">
              <Icon name="refresh" size="sm" />
            </button>
          </div>
        </div>

        <div v-if="listLoading" class="flex min-h-56 items-center justify-center"><LoadingSpinner /></div>
        <div v-else-if="listError" class="py-12 text-center">
          <p class="text-sm text-red-600 dark:text-red-400">{{ listError }}</p>
          <button type="button" class="btn btn-secondary btn-sm mt-3" @click="loadPackets">{{ t('common.retry') }}</button>
        </div>
        <div v-else-if="packets.length === 0" class="py-14 text-center text-sm text-gray-500 dark:text-dark-400">{{ t('redpacket.empty') }}</div>

        <div v-else class="space-y-3 p-4">
          <article v-for="packet in packets" :key="packet.id" class="overflow-hidden rounded-xl border border-gray-100 dark:border-dark-700">
            <button type="button" class="flex w-full items-center gap-3 px-4 py-3 text-left hover:bg-gray-50/70 dark:hover:bg-dark-800/35" @click="togglePacketDetail(packet)">
              <span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-rose-50 text-rose-500 dark:bg-rose-900/20 dark:text-rose-400">
                <Icon name="gift" size="sm" />
              </span>
              <span class="min-w-0 flex-1">
                <span class="flex flex-wrap items-center gap-2">
                  <strong class="truncate text-sm text-gray-900 dark:text-white">{{ packet.memo || typeLabel(packet.redpacket_type) }}</strong>
                  <span class="badge" :class="statusBadgeClass(packet.status)">{{ statusLabel(packet.status) }}</span>
                  <span class="badge bg-orange-50 text-orange-600 dark:bg-orange-900/20 dark:text-orange-400">{{ typeLabel(packet.redpacket_type) }}</span>
                </span>
                <span class="mt-1 block truncate text-xs text-gray-500 dark:text-dark-400">
                  {{ t('redpacket.remaining', { remaining: packet.remaining_count, total: packet.total_count }) }} · {{ t('redpacket.validUntil', { date: dateTime(packet.expire_at) }) }}
                </span>
              </span>
              <span class="shrink-0 text-right">
                <strong class="block text-base font-bold tabular-nums text-rose-600 dark:text-rose-400">{{ money(packet.total_amount) }}</strong>
                <span class="mt-0.5 block text-xs text-gray-400 dark:text-dark-500">{{ t('redpacket.remainingAmount') }} {{ money(packet.remaining_amount) }}</span>
              </span>
              <Icon name="chevronDown" size="sm" class="shrink-0 text-gray-400 transition-transform" :class="expandedPacketId === packet.id ? 'rotate-180' : ''" />
            </button>

            <div class="mx-4 mb-3 rounded-lg bg-gray-50 px-3 py-2.5 dark:bg-dark-900/50">
              <div class="flex items-center gap-2">
                <code class="min-w-0 flex-1 truncate text-xs text-gray-600 dark:text-dark-300">{{ packet.code }}</code>
                <button type="button" class="shrink-0 rounded-md p-1 text-gray-400 hover:bg-white hover:text-gray-700 dark:hover:bg-dark-700 dark:hover:text-white" :title="t('common.copy')" @click.stop="copyCode(packet.code)">
                  <Icon name="copy" size="xs" />
                </button>
              </div>
              <div class="mt-2 h-1.5 overflow-hidden rounded-full bg-white dark:bg-dark-800">
                <div class="h-full rounded-full bg-gradient-to-r from-rose-400 to-orange-400" :style="{ width: `${progressPercent(packet)}%` }" />
              </div>
            </div>

            <div v-if="expandedPacketId === packet.id" class="border-t border-gray-100 px-4 py-3 dark:border-dark-700">
              <div v-if="detailLoading" class="flex min-h-20 items-center justify-center"><LoadingSpinner size="sm" /></div>
              <div v-else-if="detailError" class="py-3 text-center text-sm text-red-600 dark:text-red-400">{{ detailError }}</div>
              <div v-else-if="detailClaims.length > 0">
                <h3 class="text-xs font-semibold text-gray-700 dark:text-dark-200">{{ t('redpacket.claimRecords') }}</h3>
                <div class="mt-2 max-h-44 divide-y divide-gray-100 overflow-y-auto dark:divide-dark-700">
                  <div v-for="claim in detailClaims" :key="claim.id" class="flex items-center justify-between py-2 text-xs">
                    <span class="text-gray-500 dark:text-dark-400">#{{ claim.user_id }} · {{ dateTime(claim.created_at) }}</span>
                    <strong class="text-gray-900 dark:text-white">{{ money(claim.amount) }}</strong>
                  </div>
                </div>
              </div>
              <p v-else class="py-2 text-center text-xs text-gray-500 dark:text-dark-400">{{ t('redpacket.empty') }}</p>
            </div>
          </article>
        </div>

        <Pagination v-if="total > pageSize" :page="page" :page-size="pageSize" :total="total" :show-page-size-selector="false" @update:page="changePage" />
      </section>

      <BaseDialog :show="showCreate" :title="t('redpacket.create')" width="narrow" @close="closeCreateDialog">
        <form class="space-y-5" @submit.prevent="prepareCreate">
          <div class="grid gap-4 sm:grid-cols-2">
            <label><span class="input-label">{{ t('redpacket.totalAmount') }}</span><input v-model.number="createForm.total_amount" class="input" type="number" min="0.01" step="0.01" required /></label>
            <label><span class="input-label">{{ t('redpacket.count') }}</span><input v-model.number="createForm.count" class="input" type="number" min="1" max="100" required /></label>
          </div>
          <div>
            <span class="input-label">{{ t('redpacket.type') }}</span>
            <div class="grid grid-cols-2 gap-3">
              <button v-for="type in packetTypes" :key="type.value" type="button" class="rounded-xl border p-3 text-left text-sm" :class="createForm.redpacket_type === type.value ? 'border-rose-400 bg-rose-50 dark:bg-rose-900/20' : 'border-gray-200 dark:border-dark-700'" @click="createForm.redpacket_type = type.value">
                <strong class="block">{{ type.label }}</strong><span class="mt-1 block text-xs text-gray-500">{{ type.hint }}</span>
              </button>
            </div>
          </div>
          <label class="block"><span class="input-label">{{ t('redpacket.memo') }}</span><input v-model.trim="createForm.memo" class="input" type="text" maxlength="100" :placeholder="t('redpacket.memoPlaceholder')" /></label>
          <div class="rounded-xl bg-gray-50 p-3 text-sm dark:bg-dark-800"><div class="flex justify-between"><span class="text-gray-500">{{ t('redpacket.perPacket') }}</span><strong>{{ money(createForm.count > 0 ? createForm.total_amount / createForm.count : 0) }}</strong></div></div>
          <p v-if="createError" class="text-sm text-red-600 dark:text-red-400">{{ createError }}</p>
          <button class="btn btn-danger w-full" type="submit" :disabled="createLoading || createForm.total_amount <= 0 || createForm.count < 1"><Icon name="gift" size="sm" />{{ createLoading ? t('common.processing') : t('redpacket.create') }}</button>
        </form>
      </BaseDialog>

      <BaseDialog :show="showClaim" :title="t('redpacket.claimTitle')" width="narrow" @close="closeClaimDialog">
        <form class="space-y-5" @submit.prevent="handleClaim">
          <div class="text-center"><div class="mx-auto flex h-14 w-14 items-center justify-center rounded-2xl bg-amber-50 text-amber-500 dark:bg-amber-900/20 dark:text-amber-400"><Icon name="gift" size="lg" /></div><p class="mt-3 text-sm text-gray-500 dark:text-dark-400">{{ t('redpacket.claimHint') }}</p></div>
          <label class="block"><span class="input-label">{{ t('redpacket.code') }}</span><input v-model="claimCode" class="input text-center font-mono" type="text" autocomplete="off" autocapitalize="none" spellcheck="false" required :placeholder="t('redpacket.codePlaceholder')" /></label>
          <div v-if="claimResult" class="rounded-xl bg-emerald-50 p-4 text-center dark:bg-emerald-900/20"><p class="text-sm text-emerald-700 dark:text-emerald-300">{{ t('redpacket.claimSuccess') }}</p><p class="mt-1 text-2xl font-bold text-emerald-700 dark:text-emerald-300">+{{ money(claimResult.amount) }}</p></div>
          <p v-if="claimError" class="text-center text-sm text-red-600 dark:text-red-400">{{ claimError }}</p>
          <button class="btn btn-warning w-full" type="submit" :disabled="claimLoading || !claimCode.trim()">{{ claimLoading ? t('common.processing') : t('redpacket.claim') }}</button>
        </form>
      </BaseDialog>

      <ConfirmDialog :show="confirmingCreate" :title="t('redpacket.confirmTitle')" :message="t('redpacket.confirmMessage', { amount: money(createForm.total_amount), count: createForm.count })" @confirm="handleCreate" @cancel="confirmingCreate = false" />
      <BaseDialog :show="!!createdPacket" :title="t('redpacket.createdTitle')" width="narrow" @close="createdPacket = null">
        <div v-if="createdPacket" class="space-y-4 text-center"><p class="text-sm text-gray-500">{{ t('redpacket.shareHint') }}</p><div class="rounded-xl bg-gray-50 p-4 font-mono text-lg font-semibold tracking-wider dark:bg-dark-800">{{ createdPacket.code }}</div><button class="btn btn-danger w-full" @click="copyCode(createdPacket.code)"><Icon name="copy" size="sm" />{{ t('common.copy') }}</button></div>
      </BaseDialog>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Pagination from '@/components/common/Pagination.vue'
import Icon from '@/components/icons/Icon.vue'
import {
  claimRedPacket,
  createActivityIdempotencyKey,
  createRedPacket,
  getMyRedPackets,
  getRedPacketDetail,
  type RedPacketClaimRecord,
  type RedPacketRecord,
} from '@/api/transfer'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { activityErrorMessage } from '@/utils/activityError'

const { t, locale } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const showCreate = ref(false)
const showClaim = ref(false)
const createForm = reactive({ total_amount: 0, count: 1, redpacket_type: 'equal' as 'equal' | 'random', memo: '' })
const createLoading = ref(false)
const createError = ref('')
const confirmingCreate = ref(false)
const createdPacket = ref<RedPacketRecord | null>(null)
const claimCode = ref('')
const claimLoading = ref(false)
const claimError = ref('')
const claimResult = ref<RedPacketClaimRecord | null>(null)
const packets = ref<RedPacketRecord[]>([])
const listLoading = ref(false)
const listError = ref('')
const total = ref(0)
const page = ref(1)
const pageSize = 10
const historyRole = ref<'sent' | 'received'>('sent')
const expandedPacketId = ref<number | null>(null)
const detailClaims = ref<RedPacketClaimRecord[]>([])
const detailLoading = ref(false)
const detailError = ref('')
const createAttempt = ref<{ signature: string; key: string } | null>(null)
const claimAttempt = ref<{ signature: string; key: string } | null>(null)

const currentBalance = computed(() => Number(authStore.user?.balance || 0))
const historyRoles = computed(() => [
  { value: 'sent' as const, label: t('redpacket.sent') },
  { value: 'received' as const, label: t('redpacket.received') },
])
const packetTypes = computed(() => [
  { value: 'equal' as const, label: t('redpacket.equal'), hint: t('redpacket.equalHint') },
  { value: 'random' as const, label: t('redpacket.random'), hint: t('redpacket.randomHint') },
])
const pageRange = computed(() => {
  const start = (page.value - 1) * pageSize + 1
  const end = Math.min(page.value * pageSize, total.value)
  return `${start}-${end} / ${total.value}`
})

function money(value: number) {
  return `$${Number(value || 0).toFixed(2)}`
}

function dateTime(value: string) {
  return new Intl.DateTimeFormat(locale.value, { dateStyle: 'short', timeStyle: 'short' }).format(new Date(value))
}

function errorMessage(error: unknown, fallback: string) {
  return activityErrorMessage(error, t, fallback)
}

function typeLabel(type: string) {
  return type === 'equal' ? t('redpacket.equal') : t('redpacket.random')
}

function statusLabel(status: string) {
  return t(`redpacket.status.${status}`, status)
}

function statusBadgeClass(status: string) {
  if (status === 'active') return 'bg-emerald-50 text-emerald-600 dark:bg-emerald-900/20 dark:text-emerald-400'
  if (status === 'expired') return 'bg-amber-50 text-amber-600 dark:bg-amber-900/20 dark:text-amber-400'
  return 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-dark-300'
}

function progressPercent(packet: RedPacketRecord) {
  if (packet.total_count <= 0) return 0
  return Math.min(100, Math.max(0, ((packet.total_count - packet.remaining_count) / packet.total_count) * 100))
}

function openCreateDialog() {
  createError.value = ''
  showCreate.value = true
}

function closeCreateDialog() {
  if (!createLoading.value) showCreate.value = false
}

function openClaimDialog() {
  claimError.value = ''
  claimResult.value = null
  showClaim.value = true
}

function closeClaimDialog() {
  if (!claimLoading.value) showClaim.value = false
}

function prepareCreate() {
  createError.value = ''
  confirmingCreate.value = true
}

async function handleCreate() {
  confirmingCreate.value = false
  createLoading.value = true
  const payload = { ...createForm, memo: createForm.memo || undefined }
  const signature = JSON.stringify(payload)
  if (createAttempt.value?.signature !== signature) {
    createAttempt.value = { signature, key: createActivityIdempotencyKey('redpacket-create') }
  }
  try {
    createdPacket.value = await createRedPacket(payload, createAttempt.value.key)
    createAttempt.value = null
    showCreate.value = false
    createForm.total_amount = 0
    createForm.count = 1
    createForm.memo = ''
    await Promise.all([loadPackets(), authStore.refreshUser()])
  } catch (error) {
    createError.value = errorMessage(error, t('redpacket.createFailed'))
  } finally {
    createLoading.value = false
  }
}

async function handleClaim() {
  claimError.value = ''
  claimResult.value = null
  claimLoading.value = true
  const code = claimCode.value.trim()
  if (claimAttempt.value?.signature !== code) {
    claimAttempt.value = { signature: code, key: createActivityIdempotencyKey('redpacket-claim') }
  }
  try {
    claimResult.value = await claimRedPacket(code, claimAttempt.value.key)
    claimAttempt.value = null
    claimCode.value = ''
    appStore.showSuccess(t('redpacket.claimSuccess'))
    await Promise.all([historyRole.value === 'received' ? loadPackets() : Promise.resolve(), authStore.refreshUser()])
  } catch (error) {
    claimError.value = errorMessage(error, t('redpacket.claimFailed'))
  } finally {
    claimLoading.value = false
  }
}

async function loadPackets() {
  listLoading.value = true
  listError.value = ''
  try {
    const result = await getMyRedPackets({ role: historyRole.value, page: page.value, page_size: pageSize })
    packets.value = result.items || []
    total.value = result.total || 0
  } catch (error) {
    listError.value = errorMessage(error, t('redpacket.historyFailed'))
  } finally {
    listLoading.value = false
  }
}

function changeHistoryRole(role: 'sent' | 'received') {
  historyRole.value = role
  page.value = 1
  expandedPacketId.value = null
  void loadPackets()
}

function changePage(value: number) {
  page.value = value
  expandedPacketId.value = null
  void loadPackets()
}

async function copyCode(code: string) {
  try {
    await navigator.clipboard.writeText(code)
    appStore.showSuccess(t('common.copied'))
  } catch {
    appStore.showError(t('common.copyFailed'))
  }
}

async function togglePacketDetail(packet: RedPacketRecord) {
  if (expandedPacketId.value === packet.id) {
    expandedPacketId.value = null
    return
  }
  expandedPacketId.value = packet.id
  detailClaims.value = []
  detailError.value = ''
  detailLoading.value = true
  try {
    const detail = await getRedPacketDetail(packet.id)
    detailClaims.value = detail.claims || []
  } catch (error) {
    detailError.value = errorMessage(error, t('redpacket.detailFailed'))
  } finally {
    detailLoading.value = false
  }
}

onMounted(loadPackets)
</script>
