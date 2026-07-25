<template>
  <div class="flex items-center justify-between gap-4">
    <span class="text-sm font-medium">{{ localText('显示用量查询入口', 'Show usage query entry') }}</span>
    <Toggle v-model="form.usage_query_enabled" />
  </div>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Toggle from '@/components/common/Toggle.vue'

interface ActivityUsageSettings {
  usage_query_enabled: boolean
}

const props = defineProps<{ modelValue: ActivityUsageSettings }>()
const emit = defineEmits<{ 'update:modelValue': [value: ActivityUsageSettings] }>()
const { locale } = useI18n()
const form = reactive<ActivityUsageSettings>({ ...props.modelValue })

function localText(zh: string, en: string): string {
  return locale.value.startsWith('zh') ? zh : en
}

watch(() => props.modelValue, (value) => Object.assign(form, value), { deep: true })
watch(form, (value) => emit('update:modelValue', { ...value }), { deep: true })
</script>
