<template>
  <div class="flex flex-wrap items-start justify-between gap-4 px-5 py-4">
    <div class="min-w-0 flex-1">
      <div class="flex flex-wrap items-center gap-2">
        <strong class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('channelMonitorV2ActiveProbes.settings.title') }}</strong>
        <span class="badge" :class="enabled ? 'badge-success' : 'badge-gray'">
          {{ enabled ? t('channelMonitorV2ActiveProbes.settings.on') : t('channelMonitorV2ActiveProbes.settings.off') }}
        </span>
      </div>
      <p class="mt-1 text-xs leading-5 text-gray-500 dark:text-dark-400">
        {{ t('channelMonitorV2ActiveProbes.settings.hint') }}
      </p>
      <button type="button" class="btn btn-ghost btn-sm mt-2 !px-0" @click="$emit('open-targets')">
        {{ t('channelMonitorV2ActiveProbes.settings.manageTargets') }}
        <Icon name="arrowRight" size="sm" />
      </button>
    </div>
    <Toggle :model-value="enabled" :disabled="loading || saving" @update:model-value="save" />
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import Toggle from '@/components/common/Toggle.vue'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { getActiveProbeConfig, updateActiveProbeConfig } from './api'

defineEmits<{ 'open-targets': [] }>()

const { t } = useI18n()
const appStore = useAppStore()
const enabled = ref(false)
const loading = ref(true)
const saving = ref(false)

async function load() {
  loading.value = true
  try {
    enabled.value = (await getActiveProbeConfig()).enabled
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('channelMonitorV2ActiveProbes.settings.loadFailed')))
  } finally {
    loading.value = false
  }
}

async function save(value: boolean) {
  const previous = enabled.value
  enabled.value = value
  saving.value = true
  try {
    enabled.value = (await updateActiveProbeConfig({ enabled: value })).enabled
    appStore.showSuccess(t('channelMonitorV2ActiveProbes.settings.saved'))
  } catch (error) {
    enabled.value = previous
    appStore.showError(extractApiErrorMessage(error, t('channelMonitorV2ActiveProbes.settings.saveFailed')))
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>
