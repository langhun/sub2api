import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { checkinAPI, type CheckinStatus, type CheckinResult, type BlindboxResult } from '@/api/checkin'
import { i18n } from '@/i18n'
import { extractApiErrorCode, extractApiErrorMessage } from '@/utils/apiError'
import { useAuthStore } from './auth'
import { useAppStore } from './app'

export const useCheckinStore = defineStore('checkin', () => {
  const status = ref<CheckinStatus | null>(null)
  const loading = ref(false)
  const checkinResult = ref<CheckinResult | null>(null)
  const blindboxResult = ref<BlindboxResult | null>(null)

  const canCheckin = computed(() => status.value?.can_checkin ?? false)
  const enabled = computed(() => (status.value?.enabled ?? false) || (status.value?.luck_enabled ?? false))
  const normalEnabled = computed(() => status.value?.enabled ?? false)
  const luckEnabled = computed(() => status.value?.luck_enabled ?? false)
  const checkedInToday = computed(() => enabled.value && !canCheckin.value && status.value !== null)
  const streakDays = computed(() => status.value?.streak_days ?? 0)
  const todayReward = computed(() => status.value?.today_reward ?? null)
  const todayCheckinType = computed(() => status.value?.today_checkin_type ?? null)
  const todayMultiplier = computed(() => status.value?.today_multiplier ?? null)
  const t = i18n.global.t

  function buildCheckinSuccessMessage(result: CheckinResult): string {
    if (result.checkin_type === 'luck') {
      const multiplier = typeof result.multiplier === 'number' ? result.multiplier.toFixed(2) : '1.00'
      if (result.reward_amount > 0) {
        return t('checkin.luckSuccess', { multiplier, amount: result.reward_amount.toFixed(2) })
      }
      if (result.reward_amount < 0) {
        return t('checkin.luckLoss', { multiplier, amount: Math.abs(result.reward_amount).toFixed(2) })
      }
      return t('checkin.luckEven')
    }

    return t('checkin.success', { amount: result.reward_amount.toFixed(2) })
  }

  function buildCheckinErrorMessage(error: unknown): string {
    switch (extractApiErrorCode(error)) {
      case 'ALREADY_CHECKED_IN':
        return t('checkin.alreadyChecked')
      case 'INVALID_BET_AMOUNT':
        return t('checkin.insufficientBalance')
      default:
        return extractApiErrorMessage(error, t('common.error'))
    }
  }

  function shouldRefreshStatusAfterError(error: unknown): boolean {
    const code = extractApiErrorCode(error)
    return code === 'ALREADY_CHECKED_IN'
      || code === 'CHECKIN_DISABLED'
      || code === 'CHECKIN_LUCK_DISABLED'
      || code === 'INVALID_BET_AMOUNT'
  }

  async function fetchStatus() {
    try {
      status.value = await checkinAPI.getCheckinStatus()
    } catch {
      status.value = null
    }
  }

  async function doCheckin(): Promise<CheckinResult | null> {
    if (loading.value) return null
    loading.value = true
    try {
      const result = await checkinAPI.checkin()
      checkinResult.value = result
      blindboxResult.value = result.blindbox ?? null

      if (status.value) {
        status.value.can_checkin = false
        status.value.streak_days = result.streak_days
        status.value.today_reward = result.reward_amount
        status.value.today_checkin_type = result.checkin_type
      }

      const authStore = useAuthStore()
      await authStore.refreshUser()
      if (status.value && typeof authStore.user?.balance === 'number') {
        status.value.balance = authStore.user.balance
      }

      useAppStore().showSuccess(buildCheckinSuccessMessage(result))

      return result
    } catch (error) {
      if (shouldRefreshStatusAfterError(error)) {
        await fetchStatus()
      }
      useAppStore().showError(buildCheckinErrorMessage(error))
      return null
    } finally {
      loading.value = false
    }
  }

  async function doLuckCheckin(betAmount: number): Promise<CheckinResult | null> {
    if (loading.value) return null
    loading.value = true
    try {
      const result = await checkinAPI.luckCheckin(betAmount)
      checkinResult.value = result
      blindboxResult.value = result.blindbox ?? null

      if (status.value) {
        status.value.can_checkin = false
        status.value.streak_days = result.streak_days
        status.value.today_reward = result.reward_amount
        status.value.today_checkin_type = result.checkin_type
        if (result.multiplier !== undefined) {
          status.value.today_multiplier = result.multiplier
        }
      }

      const authStore = useAuthStore()
      await authStore.refreshUser()
      if (status.value && typeof authStore.user?.balance === 'number') {
        status.value.balance = authStore.user.balance
      }

      useAppStore().showSuccess(buildCheckinSuccessMessage(result))

      return result
    } catch (error) {
      if (shouldRefreshStatusAfterError(error)) {
        await fetchStatus()
      }
      useAppStore().showError(buildCheckinErrorMessage(error))
      return null
    } finally {
      loading.value = false
    }
  }

  function clearBlindboxResult() {
    blindboxResult.value = null
  }

  function $reset() {
    status.value = null
    loading.value = false
    checkinResult.value = null
    blindboxResult.value = null
  }

  return {
    status,
    loading,
    checkinResult,
    blindboxResult,
    canCheckin,
    enabled,
    normalEnabled,
    luckEnabled,
    checkedInToday,
    streakDays,
    todayReward,
    todayCheckinType,
    todayMultiplier,
    fetchStatus,
    doCheckin,
    doLuckCheckin,
    clearBlindboxResult,
    $reset
  }
})
