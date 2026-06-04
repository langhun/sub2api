<template>
  <div class="relative flex min-h-screen flex-col bg-[var(--background)] text-[var(--foreground)]">
    <PublicPageHeader active-path="/pricing" />

    <main class="mx-auto w-full max-w-7xl flex-1 px-4 py-6 sm:px-6 sm:py-8">
      <div class="space-y-6">
        <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h1 class="text-xl font-bold text-[var(--foreground)] sm:text-2xl">{{ t('pricing.title') }}</h1>
            <p class="mt-1 text-sm text-[var(--muted-foreground)]">{{ t('pricing.description') }}</p>
          </div>
          <button @click="refresh" :disabled="loading"
            class="btn btn-secondary inline-flex items-center gap-2 px-4 py-2 text-sm">
            <svg class="h-4 w-4" :class="{ 'animate-spin': loading }" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182m0-4.991v4.99" />
            </svg>
            {{ t('common.refresh') }}
          </button>
        </div>

        <div v-for="group in data?.groups" :key="group.id" class="card overflow-hidden">
          <div class="flex items-center gap-3 border-b border-[var(--border)] px-6 py-4">
            <span class="text-lg font-semibold text-[var(--foreground)]">{{ group.name }}</span>
            <span v-if="group.platform" class="badge badge-gray">{{ group.platform }}</span>
            <span v-if="group.rate_multiplier !== 1" class="badge badge-warning">×{{ group.rate_multiplier.toFixed(2) }}</span>
            <span class="badge badge-gray">
              {{ group.models.length }} {{ group.models.length === 1 ? t('pricing.model') : t('pricing.models') }}
            </span>
          </div>
          <div v-if="group.models.length > 0">
            <div class="hidden sm:block overflow-x-auto">
              <table class="w-full" style="table-layout: fixed;">
                <colgroup>
                  <col style="width: 36%;">
                  <col style="width: 16%;">
                  <col style="width: 16%;">
                  <col style="width: 16%;">
                  <col style="width: 16%;">
                </colgroup>
                <thead>
                  <tr class="border-b border-[var(--border)] bg-[var(--muted)]">
                    <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">{{ t('pricing.modelName') }}</th>
                    <th class="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">{{ t('pricing.inputPrice') }} ($)</th>
                    <th class="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">{{ t('pricing.outputPrice') }} ($)</th>
                    <th class="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-600 dark:text-gray-400">{{ t('pricing.effectiveInput') }} ($)</th>
                    <th class="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-600 dark:text-gray-400">{{ t('pricing.effectiveOutput') }} ($)</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="m in group.models" :key="m.model_name" class="border-b border-[var(--border)] last:border-b-0">
                    <td class="px-6 py-3 text-sm font-medium text-gray-900 dark:text-white font-mono truncate" :title="m.model_name">{{ m.model_name }}</td>
                    <td class="px-4 py-3 text-right text-sm text-gray-600 dark:text-dark-300 tabular-nums">{{ formatPrice(m.input_cost_per_million) }}</td>
                    <td class="px-4 py-3 text-right text-sm text-gray-600 dark:text-dark-300 tabular-nums">{{ formatPrice(m.output_cost_per_million) }}</td>
                    <td class="px-4 py-3 text-right text-sm font-medium tabular-nums" :class="group.rate_multiplier !== 1 ? 'text-gray-600 dark:text-gray-400' : 'text-gray-700 dark:text-dark-200'">{{ formatPrice(m.effective_input) }}</td>
                    <td class="px-4 py-3 text-right text-sm font-medium tabular-nums" :class="group.rate_multiplier !== 1 ? 'text-gray-600 dark:text-gray-400' : 'text-gray-700 dark:text-dark-200'">{{ formatPrice(m.effective_output) }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
            <div class="sm:hidden divide-y divide-gray-100 dark:divide-dark-700">
              <div v-for="m in group.models" :key="m.model_name" class="px-4 py-3 space-y-2">
                <p class="text-sm font-medium text-gray-900 dark:text-white font-mono break-all">{{ m.model_name }}</p>
                <div class="grid grid-cols-2 gap-x-4 gap-y-1 text-xs">
                  <div class="flex justify-between"><span class="text-gray-500 dark:text-dark-400">{{ t('pricing.inputPrice') }}</span><span class="tabular-nums text-gray-700 dark:text-dark-200">{{ formatPrice(m.input_cost_per_million) }}</span></div>
                  <div class="flex justify-between"><span class="text-gray-500 dark:text-dark-400">{{ t('pricing.outputPrice') }}</span><span class="tabular-nums text-gray-700 dark:text-dark-200">{{ formatPrice(m.output_cost_per_million) }}</span></div>
                  <div class="flex justify-between"><span class="text-gray-600 dark:text-gray-400">{{ t('pricing.effectiveInput') }}</span><span class="tabular-nums font-medium" :class="group.rate_multiplier !== 1 ? 'text-gray-600 dark:text-gray-400' : 'text-gray-700 dark:text-dark-200'">{{ formatPrice(m.effective_input) }}</span></div>
                  <div class="flex justify-between"><span class="text-gray-600 dark:text-gray-400">{{ t('pricing.effectiveOutput') }}</span><span class="tabular-nums font-medium" :class="group.rate_multiplier !== 1 ? 'text-gray-600 dark:text-gray-400' : 'text-gray-700 dark:text-dark-200'">{{ formatPrice(m.effective_output) }}</span></div>
                </div>
              </div>
            </div>
          </div>
          <div v-else class="py-12 text-center">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('pricing.noModelsInGroup') }}</p>
          </div>
        </div>

        <div v-if="data && (!data.groups || data.groups.length === 0)" class="card py-16 text-center">
          <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('pricing.noData') }}</p>
        </div>
      </div>
    </main>

    <PublicPageFooter />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import PublicPageHeader from '@/components/common/PublicPageHeader.vue'
import PublicPageFooter from '@/components/common/PublicPageFooter.vue'
import { getPublicPricing, type PublicPricingResponse } from '@/api/pricing'

const { t } = useI18n()

const data = ref<PublicPricingResponse | null>(null)
const loading = ref(false)

async function refresh() {
  loading.value = true
  try {
    data.value = await getPublicPricing()
  } catch {
    data.value = null
  } finally {
    loading.value = false
  }
}

function formatPrice(value: number): string {
  if (!value || value === 0) return '-'
  if (value >= 1) return '$' + value.toFixed(2)
  if (value >= 0.01) return '$' + value.toFixed(4)
  return '$' + value.toFixed(6)
}

onMounted(refresh)
</script>
