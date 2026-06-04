<template>
  <div class="relative min-h-screen bg-[var(--sidebar)] text-[var(--foreground)]">

    <!-- Sidebar -->
    <AppSidebar />

    <!-- Main Content Area -->
    <div
      class="relative min-h-screen bg-[var(--sidebar)] transition-all duration-300"
      :class="[sidebarCollapsed ? 'lg:ml-[2.75rem]' : 'lg:ml-[13rem]']"
    >
      <!-- Header -->
      <AppHeader />

      <!-- Main Content -->
      <main class="app-content-panel min-h-[calc(100vh-3rem)] px-4 pb-6 pt-5 sm:px-5 md:px-6">
        <div class="mb-5">
          <h1 class="text-2xl font-bold tracking-tight text-[var(--foreground)]">
            {{ pageTitle }}
          </h1>
          <p v-if="pageDescription" class="mt-1 text-sm text-[var(--muted-foreground)]">
            {{ pageDescription }}
          </p>
        </div>
        <slot />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import '@/styles/onboarding.css'
import { computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { useAuthStore } from '@/stores/auth'
import { useOnboardingTour } from '@/composables/useOnboardingTour'
import { useOnboardingStore } from '@/stores/onboarding'
import AppSidebar from './AppSidebar.vue'
import AppHeader from './AppHeader.vue'

const appStore = useAppStore()
const authStore = useAuthStore()
const route = useRoute()
const { t } = useI18n()
const sidebarCollapsed = computed(() => appStore.sidebarCollapsed)
const isAdmin = computed(() => authStore.user?.role === 'admin')

const pageTitle = computed(() => {
  const titleKey = route.meta.titleKey as string
  return titleKey ? t(titleKey) : (route.meta.title as string) || ''
})

const pageDescription = computed(() => {
  const descriptionKey = route.meta.descriptionKey as string
  return descriptionKey ? t(descriptionKey) : (route.meta.description as string) || ''
})

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
