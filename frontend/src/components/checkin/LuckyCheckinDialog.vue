<template>
  <BaseDialog
    :show="show"
    :title="t('checkin.luckTitle')"
    width="narrow"
    :close-on-click-outside="!checkinStore.loading"
    :close-on-escape="!checkinStore.loading"
    @close="closeDialog"
  >
    <div class="mb-3 rounded-lg bg-purple-50 p-3 dark:bg-purple-900/20">
      <p class="text-xs text-purple-700 dark:text-purple-300">
        {{ t('checkin.multiplierRange', { min: checkinStore.status?.min_multiplier?.toFixed(1), max: checkinStore.status?.max_multiplier?.toFixed(1) }) }}
      </p>
    </div>

    <div v-if="step === 'input'" class="space-y-4">
      <label class="block">
        <span class="input-label">{{ t('checkin.betAmount') }}</span>
        <input
          v-model.number="betAmount"
          data-testid="luck-bet-input"
          type="number"
          step="0.01"
          :min="0.01"
          :max="availableBalance"
          class="input"
          :placeholder="t('checkin.betAmountPlaceholder')"
          @keyup.enter="reviewRisk"
        />
      </label>
      <div class="flex items-center justify-between text-xs text-gray-500 dark:text-dark-400">
        <span>{{ t('profile.accountBalance') }}: {{ formatCurrency(availableBalance) }}</span>
        <button type="button" class="text-primary-600 hover:text-primary-700 dark:text-primary-400" @click="betAmount = availableBalance">MAX</button>
      </div>
    </div>

    <div v-else class="space-y-3" data-testid="luck-risk-review">
      <div class="rounded-lg border border-amber-200 bg-amber-50 p-3 dark:border-amber-800/60 dark:bg-amber-900/20">
        <p class="text-sm font-semibold text-amber-800 dark:text-amber-200">{{ t('checkin.luckRiskTitle') }}</p>
        <p class="mt-1 text-xs leading-5 text-amber-700 dark:text-amber-300">{{ t('checkin.luckRiskWarning', { amount: formatCurrency(maxPotentialLoss) }) }}</p>
      </div>
      <dl class="divide-y divide-gray-100 rounded-lg border border-gray-200 px-3 text-sm dark:divide-dark-700 dark:border-dark-700">
        <div class="flex items-center justify-between gap-4 py-2.5">
          <dt class="text-gray-500 dark:text-dark-400">{{ t('checkin.betAmount') }}</dt>
          <dd class="font-semibold text-gray-900 dark:text-white">{{ formatCurrency(betAmount) }}</dd>
        </div>
        <div class="flex items-center justify-between gap-4 py-2.5">
          <dt class="text-gray-500 dark:text-dark-400">{{ t('checkin.luckOutcomeRange') }}</dt>
          <dd class="text-right font-semibold text-gray-900 dark:text-white">{{ formatSignedCurrency(minPotentialChange) }} - {{ formatSignedCurrency(maxPotentialChange) }}</dd>
        </div>
      </dl>
    </div>

    <div v-if="checkinStore.actionError" class="mt-3 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-900/20 dark:text-red-300" role="alert">
      {{ actionErrorMessage }}
    </div>

    <template #footer>
      <div class="flex flex-row items-center justify-end gap-3">
        <button type="button" class="btn btn-secondary" :disabled="checkinStore.loading" @click="step === 'confirm' ? (step = 'input') : closeDialog()">
          {{ step === 'confirm' ? t('common.back') : t('common.cancel') }}
        </button>
        <button v-if="step === 'input'" type="button" data-testid="luck-review" :disabled="!validBet" class="rounded-xl bg-purple-500 px-5 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-purple-600 disabled:opacity-50" @click="reviewRisk">
          {{ t('checkin.luckReviewAction') }}
        </button>
        <button v-else type="button" data-testid="luck-submit" :disabled="checkinStore.loading" class="rounded-xl bg-purple-500 px-5 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-purple-600 disabled:opacity-50" @click="submit">
          {{ checkinStore.loading ? t('common.loading') : t('checkin.luckButton') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { useCheckinStore } from '@/stores/checkin'
import { extractApiErrorMessage } from '@/utils/apiError'
import type { CheckinResult } from '@/api/checkin'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{
  close: []
  success: [result: CheckinResult]
}>()

const { t } = useI18n()
const checkinStore = useCheckinStore()
const betAmount = ref(0)
const step = ref<'input' | 'confirm'>('input')

const availableBalance = computed(() => checkinStore.status?.balance ?? 0)
const validBet = computed(() => Number.isFinite(betAmount.value) && betAmount.value > 0 && betAmount.value <= availableBalance.value)
const minPotentialChange = computed(() => betAmount.value * ((checkinStore.status?.min_multiplier ?? 0) - 1))
const maxPotentialChange = computed(() => betAmount.value * ((checkinStore.status?.max_multiplier ?? 0) - 1))
const maxPotentialLoss = computed(() => Math.max(0, -minPotentialChange.value))
const actionErrorMessage = computed(() => extractApiErrorMessage(checkinStore.actionError, t('checkin.actionFailed')))

watch(() => props.show, (show) => {
  if (!show) return
  betAmount.value = 0
  step.value = 'input'
  checkinStore.clearActionError()
})

function reviewRisk() {
  if (!validBet.value) return
  checkinStore.clearActionError()
  step.value = 'confirm'
}

async function submit() {
  if (step.value !== 'confirm' || !validBet.value) return
  const result = await checkinStore.doLuckCheckin(betAmount.value)
  if (!result) return
  emit('success', result)
  emit('close')
  betAmount.value = 0
  step.value = 'input'
}

function closeDialog() {
  if (checkinStore.loading) return
  checkinStore.clearActionError()
  emit('close')
}

function formatCurrency(value: number) {
  return `$${Math.max(0, Number.isFinite(value) ? value : 0).toFixed(2)}`
}

function formatSignedCurrency(value: number) {
  const safeValue = Number.isFinite(value) ? value : 0
  const normalized = Math.abs(safeValue) < 0.005 ? 0 : safeValue
  return `${normalized >= 0 ? '+' : '-'}$${Math.abs(normalized).toFixed(2)}`
}

</script>
