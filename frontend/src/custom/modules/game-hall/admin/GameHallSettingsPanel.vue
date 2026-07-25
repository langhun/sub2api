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
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Toggle from '@/components/common/Toggle.vue'

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

function localText(zh: string, en: string): string {
  return locale.value.startsWith('zh') ? zh : en
}

watch(() => props.modelValue, (value) => Object.assign(form, value), { deep: true })
watch(form, (value) => emit('update:modelValue', { ...value }), { deep: true })
</script>
