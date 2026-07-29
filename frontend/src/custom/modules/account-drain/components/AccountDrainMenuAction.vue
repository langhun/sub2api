<template>
  <button
    v-if="isOpenAI"
    class="flex w-full items-center gap-2 px-4 py-2 text-sm hover:bg-gray-100 disabled:cursor-wait disabled:opacity-60 dark:hover:bg-dark-700"
    :class="active ? 'text-amber-600 dark:text-amber-400' : 'text-emerald-600 dark:text-emerald-400'"
    :disabled="loading || updating"
    @click="toggle"
  >
    <Icon :name="active ? 'ban' : 'bolt'" size="sm" />
    {{ label }}
  </button>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Icon } from '@/components/icons'
import { useAppStore } from '@/stores/app'
import type { Account } from '@/types'
import {
  disableAccountDrainTarget,
  enableAccountDrainTarget,
  getAccountDrainTargetStatus,
} from '../api'

const props = defineProps<{ account: Account; show: boolean }>()
const emit = defineEmits<{ close: [] }>()
const appStore = useAppStore()
const loading = ref(false)
const updating = ref(false)
const active = ref(false)
const isOpenAI = computed(() => props.account.platform === 'openai')
const label = computed(() => {
  if (loading.value) return '读取定向消耗状态...'
  if (updating.value) return active.value ? '停止中...' : '启动中...'
  return active.value ? '停止定向消耗' : '启动定向消耗'
})

watch(
  () => [props.show, props.account.id, props.account.platform] as const,
  ([visible]) => {
    if (visible && isOpenAI.value) void loadStatus()
  },
  { immediate: true },
)

async function loadStatus() {
  loading.value = true
  try {
    active.value = (await getAccountDrainTargetStatus(props.account.id)).active
  } catch (error) {
    console.error('Failed to load directed-consumption status', error)
    appStore.showError('读取定向消耗状态失败')
  } finally {
    loading.value = false
  }
}

async function toggle() {
  updating.value = true
  try {
    if (active.value) {
      await disableAccountDrainTarget(props.account.id)
      active.value = false
      appStore.showSuccess('已停止该账号的定向消耗')
    } else {
      await enableAccountDrainTarget(props.account.id)
      active.value = true
      appStore.showSuccess('已启动该账号的定向消耗')
    }
    emit('close')
  } catch (error) {
    console.error('Failed to update directed-consumption target', error)
    appStore.showError(active.value ? '停止定向消耗失败' : '启动定向消耗失败')
  } finally {
    updating.value = false
  }
}
</script>
