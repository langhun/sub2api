<template>
  <div
    class="min-h-screen"
    :class="themeStyle === 'apple' ? 'bg-transparent' : 'bg-gray-50 dark:bg-dark-950'"
  >
    <!-- Default Background Decoration (non-apple) -->
    <div v-if="themeStyle !== 'apple'" class="pointer-events-none fixed inset-0 bg-mesh-gradient"></div>

    <!-- Sidebar -->
    <AppSidebar style="position: relative; z-index: 30;" />

    <!-- Main Content Area -->
    <div
      class="relative min-h-screen transition-all duration-300"
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
      <main class="p-4 md:p-6 lg:p-8">
        <slot />
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
