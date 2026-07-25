<template>
  <div v-if="user && checkinStore.enabled" class="hidden items-center gap-2 sm:flex">
    <template v-if="checkinStore.canCheckin">
      <button
        v-if="checkinStore.normalEnabled"
        type="button"
        data-testid="header-normal-checkin"
        :disabled="checkinStore.loading"
        class="flex items-center gap-1.5 rounded-xl bg-amber-50 px-3 py-1.5 text-sm font-semibold text-amber-700 transition-colors hover:bg-amber-100 disabled:opacity-50 dark:bg-amber-900/20 dark:text-amber-300 dark:hover:bg-amber-900/30"
        @click="submitNormalCheckin"
      >
        <Icon name="checkCircle" size="sm" />
        {{ checkinStore.loading ? '...' : t('checkin.normalCheckin') }}
      </button>
      <button
        v-if="checkinStore.luckEnabled"
        type="button"
        data-testid="header-luck-checkin"
        :disabled="checkinStore.loading"
        class="flex items-center gap-1.5 rounded-xl bg-purple-50 px-3 py-1.5 text-sm font-semibold text-purple-700 transition-colors hover:bg-purple-100 disabled:opacity-50 dark:bg-purple-900/20 dark:text-purple-300 dark:hover:bg-purple-900/30"
        @click="showLuckDialog = true"
      >
        <Icon name="sparkles" size="sm" />
        {{ t('checkin.luckCheckin') }}
      </button>
    </template>
  </div>

  <LuckyCheckinDialog
    :show="showLuckDialog"
    @close="showLuckDialog = false"
    @success="handleCheckinSuccess"
  />
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import Icon from '@/components/icons/Icon.vue'
import type { CheckinResult } from '../api/checkin'
import { useCheckinStore } from '../stores/checkin'
import LuckyCheckinDialog from './LuckyCheckinDialog.vue'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const checkinStore = useCheckinStore()
const user = computed(() => authStore.user)
const showLuckDialog = ref(false)

async function submitNormalCheckin() {
  const result = await checkinStore.doCheckin()
  if (result) handleCheckinSuccess(result)
}

function handleCheckinSuccess(result: CheckinResult) {
  showLuckDialog.value = false
  const reward = Number(result.reward_amount || 0)
  const amount = Math.abs(reward).toFixed(2)
  let detail = `${reward >= 0 ? '+' : '-'}$${amount}`

  if (result.checkin_type === 'luck') {
    const multiplier = Number(result.multiplier || 0).toFixed(2)
    if (reward > 0) detail = t('checkin.luckSuccess', { multiplier, amount })
    else if (reward < 0) detail = t('checkin.luckLoss', { multiplier, amount })
    else detail = t('checkin.luckEven')
  }

  appStore.showSuccess(`${t('checkin.success')} ${detail}`)
}

onMounted(() => {
  if (user.value && !checkinStore.status && !checkinStore.statusLoading) {
    void checkinStore.fetchStatus()
  }
})
</script>
