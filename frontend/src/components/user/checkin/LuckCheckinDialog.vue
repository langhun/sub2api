<template>
  <BaseDialog :show="show" :title="t('checkin.luckTitle')" width="narrow" :close-on-click-outside="true" @close="emit('close')">
    <div class="mb-3 rounded-lg bg-purple-50 p-3 dark:bg-purple-900/20">
      <p class="text-xs text-purple-700 dark:text-purple-300">
        {{ t('checkin.multiplierRange', { min: minMultiplierText, max: maxMultiplierText }) }}
      </p>
    </div>

    <div class="space-y-4">
      <div>
        <label class="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t('checkin.betAmount') }}
        </label>
        <input
          v-model.number="betAmount"
          type="number"
          step="0.01"
          :min="0.01"
          :max="balanceLimit"
          class="input"
          :placeholder="t('checkin.betAmountPlaceholder')"
          @keyup.enter="handleSubmit"
        />
      </div>

      <div class="flex items-center justify-between text-xs text-gray-500 dark:text-gray-400">
        <span>{{ t('profile.accountBalance') }}: ${{ balanceLimit.toFixed(2) }}</span>
        <button type="button" class="text-primary-600 hover:text-primary-700 dark:text-primary-400" @click="betAmount = balanceLimit">
          MAX
        </button>
      </div>

      <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
        <p class="text-xs text-gray-500 dark:text-gray-400">
          {{ t('checkin.luckDesc', { min: minMultiplierText, max: maxMultiplierText }) }}
        </p>
      </div>
    </div>

    <template #footer>
      <div class="flex flex-row items-center justify-end gap-3">
        <button type="button" class="btn btn-secondary" @click="emit('close')">
          {{ t('common.cancel') }}
        </button>
        <button
          type="button"
          :disabled="isSubmitDisabled"
          class="rounded-xl bg-purple-500 px-5 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-purple-600 disabled:opacity-50"
          @click="handleSubmit"
        >
          {{ loading ? '...' : t('checkin.luckButton') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'

interface Props {
  show: boolean
  loading: boolean
  balanceLimit: number
  minMultiplier?: number | null
  maxMultiplier?: number | null
}

const props = defineProps<Props>()
const emit = defineEmits<{
  close: []
  submit: []
}>()

const betAmount = defineModel<number>('betAmount', { required: true })
const { t } = useI18n()

const minMultiplierText = computed(() => formatMultiplier(props.minMultiplier))
const maxMultiplierText = computed(() => formatMultiplier(props.maxMultiplier))

const isSubmitDisabled = computed(() =>
  props.loading
  || !betAmount.value
  || betAmount.value <= 0
  || betAmount.value > props.balanceLimit,
)

function formatMultiplier(value?: number | null): string {
  return Number.isFinite(value) ? Number(value).toFixed(1) : '0.0'
}

function handleSubmit() {
  if (isSubmitDisabled.value) return
  emit('submit')
}
</script>
