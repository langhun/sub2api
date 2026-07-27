<template>
  <div class="card">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ localText('娱乐大厅与游戏', 'Game hall and games') }}</h2>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ localText('统一控制大厅入口、DG 钱包兑换和游戏投注范围。', 'Control the hall entry, DG wallet exchange, and game bet limits.') }}</p>
    </div>
    <div class="space-y-5 p-6">
      <div class="flex items-center justify-between gap-4">
        <div><p class="font-medium text-gray-900 dark:text-white">{{ localText('娱乐大厅入口', 'Game hall entry') }}</p><p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ localText('关闭后隐藏入口，并由后端拒绝兑换和新游戏。', 'Hides the entry and blocks exchanges and new rounds.') }}</p></div>
        <Toggle v-model="form.game_hall_enabled" />
      </div>
      <template v-if="form.game_hall_enabled">
        <div class="grid gap-4 border-t border-gray-100 pt-4 dark:border-dark-700 md:grid-cols-3">
          <label class="text-sm text-gray-600 dark:text-gray-300">{{ localText('单次兑换下限', 'Minimum exchange') }}<input v-model.number="form.game_exchange_min_amount" type="number" min="0.01" step="0.01" class="input mt-1" /></label>
          <label class="text-sm text-gray-600 dark:text-gray-300">{{ localText('单次兑换上限', 'Maximum exchange') }}<input v-model.number="form.game_exchange_max_amount" type="number" min="0" step="0.01" class="input mt-1" /><span class="mt-1 block text-xs text-gray-500 dark:text-dark-400">{{ localText('0 表示不限', '0 means unlimited') }}</span></label>
          <label class="text-sm text-gray-600 dark:text-gray-300">{{ localText('每日兑换上限', 'Daily exchange limit') }}<input v-model.number="form.game_exchange_daily_limit" type="number" min="0" step="0.01" class="input mt-1" /><span class="mt-1 block text-xs text-gray-500 dark:text-dark-400">{{ localText('双向成功金额合计，0 表示不限', 'Successful amounts in both directions; 0 means unlimited') }}</span></label>
        </div>
        <div class="flex items-center justify-between gap-4">
          <div><p class="font-medium text-gray-900 dark:text-white">{{ localText('允许 DG 转回主余额', 'Allow DG return to main balance') }}</p><p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ localText('关闭后只允许主余额兑换为 DG。', 'When disabled, users can only exchange main balance to DG.') }}</p></div>
          <Toggle v-model="form.game_exchange_allow_dg_to_balance" />
        </div>
        <div class="flex items-center justify-between gap-4 border-t border-gray-100 pt-4 dark:border-dark-700">
          <span class="font-medium text-gray-900 dark:text-white">{{ localText('三轴老虎机', 'Three-reel slots') }}</span>
          <Toggle v-model="form.game_slots_enabled" />
        </div>
        <div v-if="form.game_slots_enabled" class="grid gap-4 md:grid-cols-2">
          <label class="text-sm text-gray-600 dark:text-gray-300">{{ localText('最小投注（DG）', 'Minimum bet (DG)') }}<input v-model.number="form.game_slots_min_bet" type="number" min="0.01" step="0.01" class="input mt-1" /></label>
          <label class="text-sm text-gray-600 dark:text-gray-300">{{ localText('最大投注（DG）', 'Maximum bet (DG)') }}<input v-model.number="form.game_slots_max_bet" type="number" min="0.01" step="0.01" class="input mt-1" /></label>
        </div>
        <p v-if="form.game_slots_enabled && Number(form.game_slots_min_bet) > Number(form.game_slots_max_bet)" class="text-sm text-red-600 dark:text-red-400">{{ localText('最小投注不能大于最大投注', 'Minimum bet cannot exceed maximum bet') }}</p>
      </template>

      <section class="space-y-4 border-t border-gray-100 pt-5 dark:border-dark-700" data-testid="game-hall-user-access">
        <div>
          <h3 class="font-medium text-gray-900 dark:text-white">{{ localText('用户访问控制', 'User access control') }}</h3>
          <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ localText('按用户 ID 停用或恢复娱乐大厅。停用后不会清算该用户已有的 DG 余额。', 'Disable or restore the game hall by user ID. Disabling access does not settle the user\'s existing DG balance.') }}</p>
        </div>

        <div class="flex flex-wrap items-end gap-3">
          <label class="w-full max-w-xs text-sm text-gray-600 dark:text-gray-300">
            {{ localText('用户 ID', 'User ID') }}
            <input v-model.number="accessUserID" data-testid="game-hall-access-user-id" type="number" min="1" step="1" class="input mt-1" />
          </label>
          <button type="button" data-testid="game-hall-access-load" class="btn btn-secondary" :disabled="accessLoading" @click="loadUserAccess">
            {{ accessLoading ? localText('查询中...', 'Loading...') : localText('查询权限', 'Check access') }}
          </button>
        </div>

        <p v-if="accessError" class="text-sm text-red-600 dark:text-red-400" data-testid="game-hall-access-error">{{ accessError }}</p>
        <p v-else-if="accessMessage" class="text-sm text-green-600 dark:text-green-400" data-testid="game-hall-access-message">{{ accessMessage }}</p>

        <div v-if="userAccess" class="flex items-center justify-between gap-4 rounded border border-gray-200 p-4 dark:border-dark-600">
          <div>
            <p class="text-sm font-medium text-gray-900 dark:text-white">{{ localText(`用户 #${userAccess.user_id}`, `User #${userAccess.user_id}`) }}</p>
            <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ userAccess.disabled ? localText('娱乐大厅已停用', 'Game hall is disabled') : localText('娱乐大厅可用', 'Game hall is enabled') }}</p>
          </div>
          <div :class="accessSaving && 'pointer-events-none opacity-60'">
            <Toggle :model-value="userAccess.disabled" @update:model-value="updateUserAccess" />
          </div>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Toggle from '@/components/common/Toggle.vue'
import {
  getGameHallUserAccess,
  updateGameHallUserAccess,
  type GameHallUserAccess,
} from '../api/admin'

interface GameHallSettings {
  game_hall_enabled: boolean
  game_slots_enabled: boolean
  game_slots_min_bet: number
  game_slots_max_bet: number
  game_exchange_min_amount: number
  game_exchange_max_amount: number
  game_exchange_daily_limit: number
  game_exchange_allow_dg_to_balance: boolean
}

const props = defineProps<{ modelValue: GameHallSettings }>()
const emit = defineEmits<{ 'update:modelValue': [value: GameHallSettings] }>()
const { locale } = useI18n()
const form = reactive<GameHallSettings>({ ...props.modelValue })
const accessUserID = ref<number | null>(null)
const userAccess = ref<GameHallUserAccess | null>(null)
const accessLoading = ref(false)
const accessSaving = ref(false)
const accessError = ref('')
const accessMessage = ref('')

function localText(zh: string, en: string): string {
  return locale.value.startsWith('zh') ? zh : en
}

function validatedAccessUserID(): number | null {
  const userID = Number(accessUserID.value)
  if (!Number.isSafeInteger(userID) || userID <= 0) {
    accessError.value = localText('请输入有效的用户 ID', 'Enter a valid user ID')
    return null
  }
  return userID
}

function errorMessage(error: unknown): string {
  const detail = (error as { response?: { data?: { detail?: unknown } } })?.response?.data?.detail
  return typeof detail === 'string' && detail.trim()
    ? detail
    : localText('无法更新娱乐大厅权限，请稍后重试', 'Could not update game hall access. Try again later.')
}

async function loadUserAccess() {
  const userID = validatedAccessUserID()
  if (userID === null) return

  accessLoading.value = true
  accessError.value = ''
  accessMessage.value = ''
  try {
    userAccess.value = await getGameHallUserAccess(userID)
  } catch (error) {
    userAccess.value = null
    accessError.value = errorMessage(error)
  } finally {
    accessLoading.value = false
  }
}

async function updateUserAccess(disabled: boolean) {
  if (!userAccess.value || accessSaving.value) return

  accessSaving.value = true
  accessError.value = ''
  accessMessage.value = ''
  try {
    userAccess.value = await updateGameHallUserAccess(userAccess.value.user_id, disabled)
    accessMessage.value = disabled
      ? localText('已停用该用户的娱乐大厅', 'Game hall disabled for this user')
      : localText('已恢复该用户的娱乐大厅', 'Game hall restored for this user')
  } catch (error) {
    accessError.value = errorMessage(error)
  } finally {
    accessSaving.value = false
  }
}

watch(() => props.modelValue, (value) => Object.assign(form, value), { deep: true })
watch(form, (value) => emit('update:modelValue', { ...value }), { deep: true })
watch(accessUserID, () => {
  userAccess.value = null
  accessError.value = ''
  accessMessage.value = ''
})
</script>
