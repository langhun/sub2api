<template>
  <div>
    <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
      {{ localText('默认首页', 'Default homepage') }}
    </label>
    <select v-model="form.default_homepage" class="input">
      <option value="default">{{ localText('默认介绍页', 'Default introduction') }}</option>
      <option value="pixel">{{ localText('像素风主页', 'Pixel home') }}</option>
      <option value="dino">Dino</option>
    </select>
    <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
      {{ localText('访问根路径 / 时使用的首页；三个页面始终可通过 /home、/Pixel 和 /Dino 访问。', 'Selects the page shown at /. All options remain available at /home, /Pixel, and /Dino.') }}
    </p>

    <div class="mt-5 border-t border-gray-100 pt-5 dark:border-dark-700">
      <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
        {{ localText('主题选择', 'Theme') }}
      </label>
      <select v-model="form.theme_style" class="input">
        <option value="glass">{{ localText('液态玻璃主题', 'Liquid glass theme') }}</option>
        <option value="classic">{{ localText('原有主题', 'Classic theme') }}</option>
      </select>
      <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
        {{ localText('选择站点全局视觉主题。保存后立即生效。', 'Select the global visual theme. Changes apply after saving.') }}
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'

interface BrandHomeSettings {
  default_homepage: 'default' | 'dino' | 'pixel'
  theme_style: 'glass' | 'classic'
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
