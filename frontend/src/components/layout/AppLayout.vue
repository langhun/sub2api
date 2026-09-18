<template>
  <div
    class="min-h-screen min-w-0 max-w-full overflow-x-clip"
    :style="{ '--app-sidebar-width': sidebarWidth }"
    :class="themeStyle === 'apple' ? 'bg-transparent' : 'bg-gray-50 dark:bg-dark-950'"
  >
    <!-- Default Background Decoration (non-apple) -->
    <div v-if="themeStyle !== 'apple'" class="pointer-events-none fixed inset-0 bg-mesh-gradient"></div>

    <!-- Sidebar -->
    <AppSidebar />

    <!-- Main Content Area -->
    <div
      class="relative min-h-screen min-w-0 max-w-full overflow-x-clip transition-all duration-300"
      style="position: relative; z-index: 10;"
      :class="[
        sidebarCollapsed
          ? 'lg:ml-[72px]'
          : locale === 'en'
            ? 'lg:ml-[232px]'
            : 'lg:ml-[182px]'
      ]"
    >
      <!-- Header -->
      <AppHeader />

      <!-- Main Content -->
      <main class="min-w-0 max-w-full px-4 pb-6 pt-7 md:px-6 lg:px-8 lg:pb-8 lg:pt-9">
        <div class="flex w-full min-w-0 max-w-full flex-col">
          <slot />
        </div>
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import '@/styles/onboarding.css'
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { useAuthStore } from '@/stores/auth'
import { useOnboardingTour } from '@/composables/useOnboardingTour'
import { useOnboardingStore } from '@/stores/onboarding'
import { useTheme } from '@/custom/modules/apple-theme/useTheme'
import AppSidebar from './AppSidebar.vue'
import AppHeader from './AppHeader.vue'

const appStore = useAppStore()
const authStore = useAuthStore()
const { locale } = useI18n()
const { themeStyle } = useTheme()
const sidebarCollapsed = computed(() => appStore.sidebarCollapsed)
const sidebarWidth = computed(() => {
  if (sidebarCollapsed.value) return '72px'
  return locale.value === 'en' ? '232px' : '182px'
})
const isAdmin = computed(() => authStore.user?.role === 'admin')

const { replayTour } = useOnboardingTour({
  storageKey: isAdmin.value ? 'admin_guide' : 'user_guide',
  autoStart: true
})

const onboardingStore = useOnboardingStore()

onMounted(() => {
  onboardingStore.setReplayCallback(replayTour)
})

defineExpose({ replayTour })
</script>
