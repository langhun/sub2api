<template>
  <div class="relative inline-block">
    <button
      @click.stop="toggleDropdown"
      class="btn-ghost btn-icon"
      :aria-label="t('theme.toggle')"
    >
      <Icon v-if="effectiveMode === 'dark'" name="moon" size="md" />
      <Icon v-else name="sun" size="md" />
    </button>

    <!-- Dropdown -->
    <Transition name="dropdown">
      <div
        v-if="isOpen"
        v-click-outside="closeDropdown"
        class="dropdown absolute right-0 top-full mt-2 w-56"
      >
        <!-- Theme Style Section -->
        <div class="px-3 py-2 text-xs font-semibold text-gray-500 dark:text-dark-400">
          {{ t('theme.styleLabel') }}
        </div>

        <button
          v-for="style in themeStyles"
          :key="style.value"
          @click="selectStyle(style.value)"
          class="dropdown-item"
          :class="{ 'bg-gray-100 dark:bg-dark-700': themeStyle === style.value }"
        >
          <Icon :name="style.icon" size="sm" />
          <span>{{ t(style.label) }}</span>
          <Icon
            v-if="themeStyle === style.value"
            name="check"
            size="sm"
            class="ml-auto text-primary-600 dark:text-primary-400"
          />
        </button>

        <div class="my-1 h-px bg-gray-200 dark:bg-dark-700" />

        <!-- Theme Mode Section -->
        <div class="px-3 py-2 text-xs font-semibold text-gray-500 dark:text-dark-400">
          {{ t('theme.modeLabel') }}
        </div>

        <button
          v-for="mode in themeModes"
          :key="mode.value"
          @click="selectMode(mode.value)"
          class="dropdown-item"
          :class="{ 'bg-gray-100 dark:bg-dark-700': themeMode === mode.value }"
        >
          <Icon :name="mode.icon" size="sm" />
          <span>{{ t(mode.label) }}</span>
          <Icon
            v-if="themeMode === mode.value"
            name="check"
            size="sm"
            class="ml-auto text-primary-600 dark:text-primary-400"
          />
        </button>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { useTheme, type ThemeMode, type ThemeStyle } from './useTheme'

const { t } = useI18n()
const { themeMode, themeStyle, effectiveMode, setThemeMode, setThemeStyle } = useTheme()

const isOpen = ref(false)

const themeStyles = [
  { value: 'default' as ThemeStyle, label: 'theme.styleDefault', icon: 'cog' as const },
  { value: 'apple' as ThemeStyle, label: 'theme.styleApple', icon: 'sparkles' as const },
]

const themeModes = [
  { value: 'light' as ThemeMode, label: 'theme.modeLight', icon: 'sun' as const },
  { value: 'dark' as ThemeMode, label: 'theme.modeDark', icon: 'moon' as const },
  { value: 'auto' as ThemeMode, label: 'theme.modeAuto', icon: 'globe' as const },
]

function toggleDropdown() {
  isOpen.value = !isOpen.value
}

function closeDropdown() {
  isOpen.value = false
}

function selectStyle(style: ThemeStyle) {
  setThemeStyle(style)
  closeDropdown()
}

function selectMode(mode: ThemeMode) {
  setThemeMode(mode)
  closeDropdown()
}

// Click outside directive
interface ClickOutsideHTMLElement extends HTMLElement {
  _clickOutside?: (event: MouseEvent) => void
}

const vClickOutside = {
  mounted(el: ClickOutsideHTMLElement, binding: { value: () => void }) {
    el._clickOutside = (event: MouseEvent) => {
      if (!(el === event.target || el.contains(event.target as Node))) {
        binding.value()
      }
    }
    document.addEventListener('click', el._clickOutside)
  },
  unmounted(el: ClickOutsideHTMLElement) {
    if (el._clickOutside) {
      document.removeEventListener('click', el._clickOutside)
    }
  },
}
</script>

<style scoped>
.dropdown-enter-active,
.dropdown-leave-active {
  transition: opacity 0.2s, transform 0.2s;
}

.dropdown-enter-from,
.dropdown-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>
