<template>
  <BaseDialog
    :show="show"
    :title="t('admin.proxies.assignAccounts.title')"
    width="wide"
    @close="handleClose"
  >
    <div class="space-y-5">
      <div class="rounded-lg border border-blue-200 bg-blue-50 p-3 text-sm text-blue-800 dark:border-blue-700/40 dark:bg-blue-900/20 dark:text-blue-200">
        {{ t('admin.proxies.assignAccounts.scopeHint', { count: proxyIds.length }) }}
      </div>

      <div class="grid gap-4 lg:grid-cols-3">
        <section class="rounded-lg border border-gray-200 p-3 dark:border-dark-600">
          <h4 class="mb-3 text-sm font-semibold text-gray-900 dark:text-white">
            {{ t('admin.proxies.assignAccounts.platforms') }}
          </h4>
          <label v-for="option in platformOptions" :key="option.value" class="mb-2 flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              :checked="selectedPlatforms.includes(option.value)"
              @change="toggleValue(selectedPlatforms, option.value)"
            />
            <span>{{ option.label }}</span>
          </label>
          <p class="text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.proxies.assignAccounts.emptyMeansAll') }}
          </p>
        </section>

        <section class="rounded-lg border border-gray-200 p-3 dark:border-dark-600">
          <h4 class="mb-3 text-sm font-semibold text-gray-900 dark:text-white">
            {{ t('admin.proxies.assignAccounts.statuses') }}
          </h4>
          <label v-for="option in statusOptions" :key="option.value" class="mb-2 flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              :checked="selectedStatuses.includes(option.value)"
              @change="toggleValue(selectedStatuses, option.value)"
            />
            <span>{{ option.label }}</span>
          </label>
          <p class="text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.proxies.assignAccounts.emptyMeansAll') }}
          </p>
        </section>

        <section class="rounded-lg border border-gray-200 p-3 dark:border-dark-600">
          <div class="mb-3 flex items-center justify-between gap-2">
            <h4 class="text-sm font-semibold text-gray-900 dark:text-white">
              {{ t('admin.proxies.assignAccounts.groups') }}
            </h4>
            <button
              v-if="selectedGroupIds.length"
              type="button"
              class="text-xs text-primary-600 hover:text-primary-700 dark:text-primary-400"
              @click="selectedGroupIds = []"
            >
              {{ t('common.clear') }}
            </button>
          </div>
          <div class="max-h-40 overflow-auto pr-1">
            <label v-for="group in groups" :key="group.id" class="mb-2 flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                :checked="selectedGroupIds.includes(group.id)"
                @change="toggleValue(selectedGroupIds, group.id)"
              />
              <span class="truncate">{{ group.name }}</span>
            </label>
            <p v-if="groups.length === 0" class="text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.proxies.assignAccounts.noGroups') }}
            </p>
          </div>
          <p class="mt-2 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.proxies.assignAccounts.groupOrHint') }}
          </p>
        </section>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <button
          type="button"
          class="btn btn-secondary"
          :disabled="previewing || executing || proxyIds.length === 0"
          @click="previewAssignment"
        >
          <Icon name="search" size="sm" class="mr-1.5" />
          {{ previewing ? t('admin.proxies.assignAccounts.previewing') : t('admin.proxies.assignAccounts.preview') }}
        </button>
        <button
          type="button"
          class="btn btn-primary"
          :disabled="executing || previewing || proxyIds.length === 0"
          @click="executeAssignment"
        >
          <Icon name="check" size="sm" class="mr-1.5" />
          {{ executing ? t('admin.proxies.assignAccounts.assigning') : t('admin.proxies.assignAccounts.execute') }}
        </button>
      </div>

      <div v-if="errorMessage" class="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-700/40 dark:bg-red-900/20 dark:text-red-200">
        {{ errorMessage }}
      </div>

      <section v-if="result" class="space-y-4">
        <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
          <div
            v-for="metric in resultMetrics"
            :key="metric.label"
            class="rounded-lg border border-gray-200 bg-white p-3 dark:border-dark-600 dark:bg-dark-800"
          >
            <div class="text-xs text-gray-500 dark:text-gray-400">{{ metric.label }}</div>
            <div class="mt-1 text-xl font-semibold text-gray-900 dark:text-white">{{ metric.value }}</div>
          </div>
        </div>

        <div v-if="result.unique_account_count === 0" class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800 dark:border-amber-700/40 dark:bg-amber-900/20 dark:text-amber-200">
          {{ t('admin.proxies.assignAccounts.emptyTargets') }}
        </div>

        <div class="max-h-96 space-y-3 overflow-auto pr-1">
          <article
            v-for="proxy in result.proxies"
            :key="proxy.proxy_id"
            class="rounded-lg border border-gray-200 bg-white p-3 dark:border-dark-600 dark:bg-dark-800"
          >
            <div class="flex flex-wrap items-center justify-between gap-2">
              <div>
                <h4 class="font-semibold text-gray-900 dark:text-white">
                  #{{ proxy.proxy_id }} {{ proxy.proxy_name || t('admin.proxies.assignAccounts.unnamedProxy') }}
                </h4>
                <p class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.proxies.assignAccounts.proxyCountChange', {
                    before: proxy.before_account_count,
                    after: proxy.after_account_count,
                    planned: proxy.planned_count,
                    assigned: proxy.assigned_count
                  }) }}
                </p>
              </div>
            </div>
            <div v-if="proxy.accounts.length" class="mt-3 flex flex-wrap gap-2">
              <span
                v-for="account in proxy.accounts.slice(0, 8)"
                :key="account.account_id"
                class="inline-flex items-center gap-1 rounded-full bg-gray-100 px-2.5 py-1 text-xs text-gray-700 dark:bg-dark-700 dark:text-gray-200"
                :title="account.skipped_reason || undefined"
              >
                #{{ account.account_id }} {{ account.account_name }}
                <span v-if="account.assigned" class="text-green-600 dark:text-green-400">✓</span>
                <span v-else-if="account.skipped_reason" class="text-amber-600 dark:text-amber-400">!</span>
              </span>
              <span v-if="proxy.accounts.length > 8" class="text-xs text-gray-500">
                {{ t('admin.proxies.assignAccounts.moreAccounts', { count: proxy.accounts.length - 8 }) }}
              </span>
            </div>
          </article>
        </div>
      </section>
    </div>

    <template #footer>
      <div class="flex justify-end">
        <button type="button" class="btn btn-secondary" @click="handleClose">
          {{ t('common.close') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import type { AdminGroup, ProxyAccountAssignmentResult } from '@/types'

const props = defineProps<{
  show: boolean
  proxyIds: number[]
  groups: AdminGroup[]
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'assigned'): void
}>()

const { t } = useI18n()
const appStore = useAppStore()

const selectedPlatforms = ref<string[]>([])
const selectedStatuses = ref<string[]>([])
const selectedGroupIds = ref<number[]>([])
const previewing = ref(false)
const executing = ref(false)
const errorMessage = ref('')
const result = ref<ProxyAccountAssignmentResult | null>(null)

const platformOptions = [
  { value: 'anthropic', label: 'Anthropic' },
  { value: 'openai', label: 'OpenAI' },
  { value: 'gemini', label: 'Gemini' },
  { value: 'antigravity', label: 'Antigravity' }
]

const statusOptions = [
  { value: 'active', label: t('admin.accounts.status.active') },
  { value: 'inactive', label: t('admin.accounts.status.inactive') },
  { value: 'error', label: t('admin.accounts.status.error') }
]

const resultMetrics = computed(() => {
  if (!result.value) return []
  return [
    { label: t('admin.proxies.assignAccounts.matched'), value: result.value.matched_account_count },
    { label: t('admin.proxies.assignAccounts.unique'), value: result.value.unique_account_count },
    { label: t('admin.proxies.assignAccounts.duplicateHits'), value: result.value.duplicate_hit_count },
    { label: t('admin.proxies.assignAccounts.planned'), value: result.value.planned_assignment_count },
    { label: t('admin.proxies.assignAccounts.actual'), value: result.value.actual_assignment_count }
  ]
})

watch(() => props.show, (visible) => {
  if (!visible) return
  errorMessage.value = ''
  result.value = null
})

function toggleValue<T>(target: T[], value: T) {
  const index = target.indexOf(value)
  if (index >= 0) {
    target.splice(index, 1)
    return
  }
  target.push(value)
}

function buildPayload(dryRun: boolean) {
  return {
    proxy_ids: props.proxyIds,
    dry_run: dryRun,
    filters: {
      platforms: selectedPlatforms.value,
      group_ids: selectedGroupIds.value,
      statuses: selectedStatuses.value
    }
  }
}

async function previewAssignment() {
  previewing.value = true
  errorMessage.value = ''
  try {
    result.value = await adminAPI.proxies.assignAccounts(buildPayload(true))
  } catch (error: any) {
    errorMessage.value = error.response?.data?.detail || t('admin.proxies.assignAccounts.previewFailed')
  } finally {
    previewing.value = false
  }
}

async function executeAssignment() {
  executing.value = true
  errorMessage.value = ''
  try {
    result.value = await adminAPI.proxies.assignAccounts(buildPayload(false))
    appStore.showSuccess(t('admin.proxies.assignAccounts.assignSuccess', {
      count: result.value.actual_assignment_count
    }))
    emit('assigned')
  } catch (error: any) {
    errorMessage.value = error.response?.data?.detail || t('admin.proxies.assignAccounts.assignFailed')
  } finally {
    executing.value = false
  }
}

function handleClose() {
  emit('close')
}
</script>
