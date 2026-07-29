import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { checkinAPI, type CheckinStatus, type CheckinResult, type BlindboxResult } from '@/custom/modules/activity/api/checkin'
import { useAuthStore } from '@/stores/auth'

export const useCheckinStore = defineStore('checkin', () => {
  const status = ref<CheckinStatus | null>(null)
  const statusLoading = ref(false)
  const statusError = ref<unknown | null>(null)
  const loading = ref(false)
  const actionError = ref<unknown | null>(null)
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

  async function fetchStatus(): Promise<boolean> {
    statusLoading.value = true
    statusError.value = null
    try {
      status.value = await checkinAPI.getCheckinStatus()
      return true
    } catch (cause) {
      statusError.value = cause
      return false
    } finally {
      statusLoading.value = false
    }
  }

  async function doCheckin(): Promise<CheckinResult | null> {
    if (loading.value) return null
    loading.value = true
    actionError.value = null
    try {
      const result = await checkinAPI.checkin()
      checkinResult.value = result
      blindboxResult.value = result.blindbox ?? null

      if (status.value) {
        status.value.can_checkin = false
        status.value.streak_days = result.streak_days
        status.value.today_reward = result.reward_amount
        status.value.today_checkin_type = result.checkin_type
        status.value.balance += result.reward_amount
      }

      const authStore = useAuthStore()
      applyBalanceReward(authStore, result.reward_amount)
      await refreshUserBalance(authStore)

      return result
    } catch (cause) {
      actionError.value = cause
      return null
    } finally {
      loading.value = false
    }
  }

  async function doLuckCheckin(betAmount: number, useMaxBalance = false): Promise<CheckinResult | null> {
    if (loading.value) return null
    loading.value = true
    actionError.value = null
    try {
      const result = await checkinAPI.luckCheckin(betAmount, useMaxBalance)
      checkinResult.value = result
      blindboxResult.value = result.blindbox ?? null

      if (status.value) {
        status.value.can_checkin = false
        status.value.streak_days = result.streak_days
        status.value.today_reward = result.reward_amount
        status.value.today_checkin_type = result.checkin_type
        status.value.balance += result.reward_amount
        if (result.multiplier !== undefined) {
          status.value.today_multiplier = result.multiplier
        }
      }

      const authStore = useAuthStore()
      applyBalanceReward(authStore, result.reward_amount)
      await refreshUserBalance(authStore)

      return result
    } catch (cause) {
      actionError.value = cause
      return null
    } finally {
      loading.value = false
    }
  }

  function clearBlindboxResult() {
    blindboxResult.value = null
  }

  function applyBalanceReward(authStore: ReturnType<typeof useAuthStore>, rewardAmount: number): void {
    if (!authStore.user || !Number.isFinite(rewardAmount)) return
    authStore.user.balance += rewardAmount
  }

  async function refreshUserBalance(authStore: ReturnType<typeof useAuthStore>): Promise<void> {
    try {
      await authStore.refreshUser()
    } catch (error) {
      // The completed check-in is authoritative. The next background refresh
      // will reconcile the profile if this best-effort display refresh fails.
      console.warn('Failed to refresh user balance after check-in:', error)
    }
  }

  function clearActionError() {
    actionError.value = null
  }

  function $reset() {
    status.value = null
    statusLoading.value = false
    statusError.value = null
    loading.value = false
    actionError.value = null
    checkinResult.value = null
    blindboxResult.value = null
  }

  return {
    status,
    statusLoading,
    statusError,
    loading,
    actionError,
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
    clearActionError,
    $reset
  }
})
