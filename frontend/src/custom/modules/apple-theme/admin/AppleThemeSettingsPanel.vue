<template>
  <div class="space-y-6">
    <div class="flex items-start gap-4">
      <div class="flex-1">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-white">
          {{ t('custom.appleTheme.title') }}
        </h3>
        <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
          {{ t('custom.appleTheme.description') }}
        </p>
      </div>
    </div>

    <div class="space-y-4">
      <!-- Enable Apple Theme Toggle -->
      <div class="flex items-center justify-between">
        <div>
          <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
            {{ t('custom.appleTheme.enableLabel') }}
          </label>
          <p class="text-xs text-gray-500 dark:text-dark-400">
            {{ t('custom.appleTheme.enableDescription') }}
          </p>
        </div>
        <button
          type="button"
          :class="[
            'switch',
            form.enable_apple_theme && 'switch-active'
          ]"
          @click="form.enable_apple_theme = !form.enable_apple_theme"
        >
          <span class="switch-thumb" />
        </button>
      </div>

      <!-- Default Theme Mode -->
      <div v-if="form.enable_apple_theme">
        <label class="input-label">
          {{ t('custom.appleTheme.defaultModeLabel') }}
        </label>
        <select
          v-model="form.default_theme_mode"
          class="input"
        >
          <option value="auto">{{ t('custom.appleTheme.modeAuto') }}</option>
          <option value="light">{{ t('custom.appleTheme.modeLight') }}</option>
          <option value="dark">{{ t('custom.appleTheme.modeDark') }}</option>
        </select>
        <p class="input-hint">
          {{ t('custom.appleTheme.defaultModeHint') }}
        </p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

interface AppleThemeForm {
  enable_apple_theme: boolean
  default_theme_mode: 'light' | 'dark' | 'auto'
}

const { t } = useI18n()

defineProps<{
  form: AppleThemeForm
}>()
</script>
