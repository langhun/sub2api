<template>
  <BaseDialog
    :show="show"
    :title="t('admin.proxies.poolMembersTitle')"
    width="full"
    @close="emit('close')"
  >
    <div class="space-y-4">
      <div class="rounded-lg border border-violet-200 bg-violet-50 px-4 py-3 text-sm text-violet-900 dark:border-violet-900/40 dark:bg-violet-950/30 dark:text-violet-100">
        <div class="font-medium">{{ t('admin.proxies.poolMembersSummary', { count: rows.length }) }}</div>
        <div v-if="hasActiveFilters" class="mt-1 text-xs text-violet-800 dark:text-violet-200">
          {{ t('admin.proxies.poolMembersFilteredSummary', { visible: filteredRows.length, total: rows.length }) }}
        </div>
        <div class="mt-1 text-xs text-violet-800 dark:text-violet-200">
          {{ t('admin.proxies.poolUsageHint') }}
        </div>
      </div>

      <div v-if="loading" class="flex items-center justify-center py-8 text-sm text-gray-500">
        <Icon name="refresh" size="md" class="mr-2 animate-spin" />
        {{ t('common.loading') }}
      </div>

      <template v-else>
        <div v-if="rows.length === 0" class="py-8 text-center text-sm text-gray-500">
          {{ t('admin.proxies.poolMembersEmpty') }}
        </div>

        <template v-else>
          <div class="rounded-lg border border-gray-200 bg-gray-50 p-3 dark:border-dark-600 dark:bg-dark-900/40">
            <div class="flex flex-wrap items-center gap-3">
              <div class="relative min-w-0 flex-1">
                <Icon
                  name="search"
                  size="md"
                  class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500"
                />
                <input
                  v-model="filters.query"
                  type="text"
                  :placeholder="t('admin.proxies.poolMembersSearchPlaceholder')"
                  class="input pl-10"
                />
              </div>
              <div class="w-full sm:w-40">
                <Select
                  v-model="filters.status"
                  :options="statusOptions"
                  :placeholder="t('admin.proxies.allStatus')"
                />
              </div>
              <div class="w-full sm:w-44">
                <Select
                  v-model="filters.health"
                  :options="healthOptions"
                  :placeholder="t('admin.proxies.allHealth')"
                />
              </div>
              <button
                v-if="hasActiveFilters"
                type="button"
                class="btn btn-secondary"
                @click="resetFilters"
              >
                {{ t('common.clear') }}
              </button>
            </div>
          </div>

          <div v-if="filteredRows.length === 0" class="py-8 text-center text-sm text-gray-500">
            {{ t('admin.proxies.poolMembersFilteredEmpty') }}
          </div>

          <div
            v-else
            class="max-h-[68vh] overflow-auto rounded-lg border border-gray-200 bg-white dark:border-dark-600 dark:bg-dark-900"
          >
            <table class="min-w-[1080px] divide-y divide-gray-200 text-sm dark:divide-dark-700">
              <thead class="sticky top-0 z-10 bg-gray-50 text-left text-xs uppercase text-gray-500 dark:bg-dark-800 dark:text-dark-400">
                <tr>
                  <th class="px-4 py-3 font-medium">{{ t('admin.proxies.columns.name') }}</th>
                  <th class="px-4 py-3 font-medium">{{ t('admin.proxies.columns.status') }}</th>
                  <th class="px-4 py-3 font-medium">{{ t('admin.proxies.poolMembersHealth') }}</th>
                  <th class="px-4 py-3 font-medium">{{ t('admin.proxies.columns.latency') }}</th>
                  <th class="px-4 py-3 font-medium">{{ t('admin.proxies.poolMembersSwitchCount') }}</th>
                  <th class="px-4 py-3 font-medium">{{ t('admin.proxies.poolMembersRecovered') }}</th>
                  <th class="px-4 py-3 font-medium">{{ t('admin.proxies.lastFailure') }}</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-200 bg-white dark:divide-dark-700 dark:bg-dark-900">
                <tr v-for="row in filteredRows" :key="row.id" class="align-top">
                  <td class="px-4 py-3">
                    <div class="font-medium text-gray-900 dark:text-white">
                      {{ row.name || t('admin.proxies.assignAccounts.unnamedProxy') }}
                    </div>
                    <div class="mt-1 flex flex-wrap items-center gap-2">
                      <code class="code text-xs">{{ row.host }}:{{ row.port }}</code>
                      <span class="text-xs text-gray-500 dark:text-dark-400">
                        {{ row.protocol.toUpperCase() }}
                      </span>
                    </div>
                  </td>
                  <td class="px-4 py-3">
                    <span :class="['badge', row.status === 'active' ? 'badge-success' : 'badge-danger']">
                      {{ t('admin.accounts.status.' + row.status) }}
                    </span>
                  </td>
                  <td class="px-4 py-3">
                    <div class="space-y-1">
                      <span :class="['badge', healthStatusClass(row.health_status)]">
                        {{ healthStatusLabel(row.health_status) }}
                      </span>
                      <div
                        v-if="row.health_status === 'cooldown' && row.cooldown_until_unix"
                        class="text-xs text-amber-600 dark:text-amber-400"
                      >
                        {{ t('admin.proxies.cooldownUntil', { time: formatCooldownCountdown(row.cooldown_until_unix) }) }}
                      </div>
                    </div>
                  </td>
                  <td class="px-4 py-3 text-gray-700 dark:text-gray-200">
                    {{ latencyText(row) }}
                  </td>
                  <td class="px-4 py-3 text-gray-700 dark:text-gray-200">
                    {{ row.failover_switch_count ?? 0 }}
                  </td>
                  <td class="px-4 py-3 text-gray-700 dark:text-gray-200">
                    {{ formatRuntimeTime(row.last_recovered_at_unix) }}
                  </td>
                  <td class="px-4 py-3">
                    <div class="max-w-xl break-all text-gray-700 dark:text-gray-200">
                      {{ row.last_fail_reason || '-' }}
                    </div>
                    <div v-if="row.last_fail_at_unix" class="mt-1 text-xs text-gray-500 dark:text-dark-400">
                      {{ formatRuntimeTime(row.last_fail_at_unix) }}
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </template>
      </template>
    </div>

    <template #footer>
      <div class="flex justify-end">
        <button @click="emit('close')" class="btn btn-secondary">
          {{ t('common.close') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Proxy } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'

interface Props {
  show: boolean
  loading: boolean
  rows: Proxy[]
}

interface Emits {
  (e: 'close'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()
const { t } = useI18n()

const filters = reactive({
  query: '',
  status: '' as '' | Proxy['status'],
  health: '' as '' | NonNullable<Proxy['health_status']>
})

const statusOptions = computed(() => [
  { value: '', label: t('admin.proxies.allStatus') },
  { value: 'active', label: t('admin.accounts.status.active') },
  { value: 'inactive', label: t('admin.accounts.status.inactive') }
])

const healthStatusClass = (status?: Proxy['health_status']) => {
  if (status === 'healthy') return 'badge-success'
  if (status === 'cooldown') return 'badge-warning'
  if (status === 'failed') return 'badge-danger'
  return 'badge-gray'
}

const healthStatusLabel = (status?: Proxy['health_status']) => {
  if (status === 'healthy') return t('admin.proxies.healthHealthy')
  if (status === 'cooldown') return t('admin.proxies.healthCooldown')
  if (status === 'failed') return t('admin.proxies.healthFailed')
  return t('admin.proxies.healthUnknown')
}

const healthOptions = computed(() => [
  { value: '', label: t('admin.proxies.allHealth') },
  { value: 'healthy', label: healthStatusLabel('healthy') },
  { value: 'cooldown', label: healthStatusLabel('cooldown') },
  { value: 'failed', label: healthStatusLabel('failed') }
])

const hasActiveFilters = computed(() => {
  return Boolean(filters.query.trim() || filters.status || filters.health)
})

const buildSearchText = (row: Proxy) =>
  [
    row.name,
    row.host,
    row.port,
    row.protocol,
    row.country,
    row.city,
    row.last_fail_reason
  ]
    .filter(Boolean)
    .join(' ')
    .toLowerCase()

const filteredRows = computed(() => {
  const query = filters.query.trim().toLowerCase()
  return props.rows.filter((row) => {
    if (filters.status && row.status !== filters.status) {
      return false
    }
    if (filters.health && row.health_status !== filters.health) {
      return false
    }
    if (query && !buildSearchText(row).includes(query)) {
      return false
    }
    return true
  })
})

const formatRuntimeTime = (unix?: number) => {
  if (!unix) return '-'
  return new Date(unix * 1000).toLocaleString()
}

const formatCooldownCountdown = (unix?: number) => {
  if (!unix) return ''
  const remaining = unix * 1000 - Date.now()
  if (remaining <= 0) return t('admin.proxies.cooldownExpired')
  const minutes = Math.floor(remaining / 60000)
  const seconds = Math.floor((remaining % 60000) / 1000)
  return minutes > 0 ? `${minutes}m ${seconds}s` : `${seconds}s`
}

const latencyText = (row: Proxy) => {
  if (typeof row.latency_ms === 'number') {
    return `${row.latency_ms}ms`
  }
  return row.latency_status === 'failed' ? t('admin.proxies.latencyFailed') : '-'
}

const resetFilters = () => {
  filters.query = ''
  filters.status = ''
  filters.health = ''
}

watch(
  () => props.show,
  (isOpen, wasOpen) => {
    if (isOpen && !wasOpen) {
      resetFilters()
    }
  }
)
</script>
