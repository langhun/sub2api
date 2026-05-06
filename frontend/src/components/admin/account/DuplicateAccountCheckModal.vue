<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.duplicateCheck.title')"
    width="extra-wide"
    @close="emit('close')"
  >
    <div class="space-y-5">
      <div class="rounded-lg border border-gray-200 bg-gray-50 p-3 text-sm text-gray-700 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-200">
        {{ t('admin.accounts.duplicateCheck.scopeHint') }}
      </div>

      <div class="grid gap-4 lg:grid-cols-3">
        <section class="rounded-lg border border-gray-200 p-3 dark:border-dark-600">
          <h4 class="mb-3 text-sm font-semibold text-gray-900 dark:text-white">
            {{ t('admin.accounts.duplicateCheck.platforms') }}
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
            {{ t('admin.accounts.duplicateCheck.emptyMeansAll') }}
          </p>
        </section>

        <section class="rounded-lg border border-gray-200 p-3 dark:border-dark-600">
          <h4 class="mb-3 text-sm font-semibold text-gray-900 dark:text-white">
            {{ t('admin.accounts.duplicateCheck.statuses') }}
          </h4>
          <label class="mb-3 flex items-center gap-2 text-sm">
            <input
              v-model="includeInactive"
              type="checkbox"
              class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
            />
            <span>{{ t('admin.accounts.duplicateCheck.includeInactive') }}</span>
          </label>
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
            {{ t('admin.accounts.duplicateCheck.statusHint') }}
          </p>
        </section>

        <section class="rounded-lg border border-gray-200 p-3 dark:border-dark-600">
          <div class="mb-3 flex items-center justify-between gap-2">
            <h4 class="text-sm font-semibold text-gray-900 dark:text-white">
              {{ t('admin.accounts.duplicateCheck.groups') }}
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
              {{ t('admin.accounts.duplicateCheck.noGroups') }}
            </p>
          </div>
        </section>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <button type="button" class="btn btn-primary" :disabled="checking" @click="runCheck">
          <Icon name="search" size="sm" class="mr-1.5" />
          {{ checking ? t('admin.accounts.duplicateCheck.checking') : t('admin.accounts.duplicateCheck.run') }}
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

        <div v-if="result.groups.length === 0" class="rounded-lg border border-green-200 bg-green-50 p-4 text-sm text-green-800 dark:border-green-700/40 dark:bg-green-900/20 dark:text-green-200">
          {{ t('admin.accounts.duplicateCheck.noDuplicates') }}
        </div>

        <div v-else class="max-h-[32rem] space-y-4 overflow-auto pr-1">
          <article
            v-for="group in result.groups"
            :key="`${group.severity}-${group.key_type}-${group.value_hash}`"
            class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-600 dark:bg-dark-800"
          >
            <div class="mb-3 flex flex-wrap items-start justify-between gap-3">
              <div>
                <div class="mb-1 flex flex-wrap items-center gap-2">
                  <span
                    class="badge"
                    :class="group.severity === 'strong' ? 'badge-danger' : 'badge-warning'"
                  >
                    {{ severityLabel(group.severity) }}
                  </span>
                  <span class="font-semibold text-gray-900 dark:text-white">
                    {{ group.key_type }}
                  </span>
                  <span class="rounded bg-gray-100 px-2 py-0.5 font-mono text-xs text-gray-700 dark:bg-dark-700 dark:text-gray-200">
                    {{ group.masked_value }}
                  </span>
                </div>
                <p class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.accounts.duplicateCheck.groupScope', {
                    platform: group.platform,
                    type: group.type || '-',
                    count: group.account_count
                  }) }}
                </p>
              </div>
            </div>

            <div class="overflow-x-auto">
              <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
                <thead class="bg-gray-50 text-xs uppercase text-gray-500 dark:bg-dark-900 dark:text-dark-400">
                  <tr>
                    <th class="px-3 py-2 text-left">{{ t('admin.accounts.duplicateCheck.accountId') }}</th>
                    <th class="px-3 py-2 text-left">{{ t('admin.accounts.columns.name') }}</th>
                    <th class="px-3 py-2 text-left">{{ t('admin.accounts.columns.platformType') }}</th>
                    <th class="px-3 py-2 text-left">{{ t('admin.accounts.columns.status') }}</th>
                    <th class="px-3 py-2 text-left">{{ t('admin.accounts.columns.groups') }}</th>
                    <th class="px-3 py-2 text-left">{{ t('admin.accounts.columns.proxy') }}</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-gray-200 dark:divide-dark-700">
                  <tr v-for="account in group.accounts" :key="account.id">
                    <td class="px-3 py-2">
                      <button
                        type="button"
                        class="font-mono text-primary-600 hover:text-primary-700 dark:text-primary-400"
                        @click="copyAccountId(account.id)"
                      >
                        #{{ account.id }}
                      </button>
                    </td>
                    <td class="px-3 py-2 font-medium text-gray-900 dark:text-white">{{ account.name }}</td>
                    <td class="px-3 py-2">
                      <PlatformTypeBadge :platform="account.platform" :type="account.type" />
                    </td>
                    <td class="px-3 py-2">{{ statusLabel(account.status) }}</td>
                    <td class="px-3 py-2">
                      <span v-if="account.groups.length === 0" class="text-gray-400">-</span>
                      <span v-else class="text-gray-700 dark:text-gray-200">
                        {{ account.groups.map(g => g.name).join(', ') }}
                      </span>
                    </td>
                    <td class="px-3 py-2">
                      <span v-if="account.proxy_id" class="font-mono">#{{ account.proxy_id }}</span>
                      <span v-else class="text-gray-400">-</span>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </article>
        </div>
      </section>
    </div>

    <template #footer>
      <div class="flex justify-end">
        <button type="button" class="btn btn-secondary" @click="emit('close')">
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
import { useClipboard } from '@/composables/useClipboard'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import PlatformTypeBadge from '@/components/common/PlatformTypeBadge.vue'
import type { AdminGroup, DuplicateAccountCheckResult } from '@/types'

const props = defineProps<{
  show: boolean
  groups: AdminGroup[]
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const { t } = useI18n()
const { copyToClipboard } = useClipboard()

const selectedPlatforms = ref<string[]>([])
const selectedStatuses = ref<string[]>([])
const selectedGroupIds = ref<number[]>([])
const includeInactive = ref(true)
const checking = ref(false)
const errorMessage = ref('')
const result = ref<DuplicateAccountCheckResult | null>(null)

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
    { label: t('admin.accounts.duplicateCheck.totalAccounts'), value: result.value.total_accounts },
    { label: t('admin.accounts.duplicateCheck.duplicateAccounts'), value: result.value.duplicate_account_count },
    { label: t('admin.accounts.duplicateCheck.groupsFound'), value: result.value.duplicate_group_count },
    { label: t('admin.accounts.duplicateCheck.strongGroups'), value: result.value.strong_group_count },
    { label: t('admin.accounts.duplicateCheck.weakGroups'), value: result.value.weak_group_count }
  ]
})

watch(() => props.show, (visible) => {
  if (!visible) return
  errorMessage.value = ''
  if (!result.value) {
    void runCheck()
  }
})

function toggleValue<T>(target: T[], value: T) {
  const index = target.indexOf(value)
  if (index >= 0) {
    target.splice(index, 1)
    return
  }
  target.push(value)
}

async function runCheck() {
  checking.value = true
  errorMessage.value = ''
  try {
    result.value = await adminAPI.accounts.checkDuplicates({
      platforms: selectedPlatforms.value,
      group_ids: selectedGroupIds.value,
      statuses: selectedStatuses.value,
      include_inactive: includeInactive.value
    })
  } catch (error: any) {
    errorMessage.value = error.response?.data?.detail || t('admin.accounts.duplicateCheck.failed')
  } finally {
    checking.value = false
  }
}

async function copyAccountId(id: number) {
  await copyToClipboard(String(id), t('admin.accounts.duplicateCheck.idCopied'))
}

function severityLabel(severity: 'strong' | 'weak') {
  return severity === 'strong'
    ? t('admin.accounts.duplicateCheck.strong')
    : t('admin.accounts.duplicateCheck.weak')
}

function statusLabel(status: string) {
  const key = `admin.accounts.status.${status}`
  const translated = t(key)
  return translated === key ? status : translated
}
</script>
