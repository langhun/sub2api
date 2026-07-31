<template>
  <div class="flex items-center justify-between gap-4">
    <span class="text-sm font-medium">{{ localText('显示排行榜入口', 'Show leaderboard entry') }}</span>
    <Toggle v-model="form.leaderboard_enabled" />
  </div>
  <template v-if="form.leaderboard_enabled">
    <div class="flex items-center justify-between gap-4"><span class="text-sm">{{ localText('显示余额排行标签', 'Show balance ranking') }}</span><Toggle v-model="form.leaderboard_balance_enabled" /></div>
    <div class="flex items-center justify-between gap-4"><span class="text-sm">{{ localText('显示消费排行标签', 'Show consumption ranking') }}</span><Toggle v-model="form.leaderboard_consumption_enabled" /></div>
    <div class="flex items-center justify-between gap-4"><span class="text-sm">{{ localText('显示签到排行标签', 'Show check-in ranking') }}</span><Toggle v-model="form.leaderboard_checkin_enabled" /></div>
    <div class="flex items-center justify-between gap-4"><span class="text-sm">{{ localText('排行榜包含管理员', 'Include administrators') }}</span><Toggle v-model="form.leaderboard_include_admin" /></div>
  </template>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Toggle from '@/components/common/Toggle.vue'

interface LeaderboardSettings {
  leaderboard_enabled: boolean
  leaderboard_balance_enabled: boolean
  leaderboard_consumption_enabled: boolean
  leaderboard_checkin_enabled: boolean
  leaderboard_include_admin: boolean
}

const props = defineProps<{ modelValue: LeaderboardSettings }>()
const emit = defineEmits<{ 'update:modelValue': [value: LeaderboardSettings] }>()
const { locale } = useI18n()
const form = reactive<LeaderboardSettings>({ ...props.modelValue })

function localText(zh: string, en: string): string {
  return locale.value.startsWith('zh') ? zh : en
}

watch(() => props.modelValue, (value) => Object.assign(form, value), { deep: true })
watch(form, (value) => emit('update:modelValue', { ...value }), { deep: true })
</script>
