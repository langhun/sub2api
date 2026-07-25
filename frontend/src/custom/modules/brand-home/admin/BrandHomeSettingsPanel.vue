<template>
  <div>
    <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
      {{ localText('默认首页', 'Default homepage') }}
    </label>
    <select v-model="form.default_homepage" class="input">
      <option value="default">{{ localText('默认介绍页', 'Default introduction') }}</option>
      <option value="dino">Dino</option>
    </select>
    <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
      {{ localText('访问根路径 / 时使用的首页；两个页面始终可通过 /home 和 /Dino 访问。', 'Selects the page shown at /. Both pages remain available at /home and /Dino.') }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'

interface BrandHomeSettings {
  default_homepage: 'default' | 'dino'
}

const props = defineProps<{ modelValue: BrandHomeSettings }>()
const emit = defineEmits<{ 'update:modelValue': [value: BrandHomeSettings] }>()
const { locale } = useI18n()
const form = reactive<BrandHomeSettings>({ ...props.modelValue })

function localText(zh: string, en: string): string {
  return locale.value.startsWith('zh') ? zh : en
}

watch(() => props.modelValue, (value) => Object.assign(form, value), { deep: true })
watch(form, (value) => emit('update:modelValue', { ...value }), { deep: true })
</script>
