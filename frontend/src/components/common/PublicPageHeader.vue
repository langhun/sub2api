<template>
  <header class="relative z-20 border-b border-[var(--border)] bg-[var(--card)]">
    <nav class="mx-auto flex h-16 max-w-7xl items-center justify-between px-4 sm:px-6">
      <router-link to="/home" class="flex items-center gap-3">
        <div class="h-8 w-8 overflow-hidden rounded-lg border border-[var(--border)] bg-[var(--card)] shadow-sm">
          <img :src="siteLogo || '/logo.png'" alt="Logo" class="h-full w-full object-contain" />
        </div>
        <span class="text-lg font-bold text-[var(--foreground)]">{{ siteName }}</span>
      </router-link>
      <div class="flex items-center gap-2">
        <router-link v-for="link in visibleNavLinks" :key="link.path" :to="link.path"
          :class="[
            'hidden items-center gap-1.5 rounded-full px-3 py-1.5 text-sm transition-colors sm:flex',
            activePath === link.path
              ? 'bg-[var(--accent)] font-medium text-[var(--accent-foreground)]'
              : 'text-[var(--muted-foreground)] hover:bg-[var(--muted)] hover:text-[var(--foreground)]'
          ]">
          {{ link.label }}
        </router-link>
        <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer"
          class="hidden items-center gap-1.5 rounded-full px-3 py-1.5 text-sm text-[var(--muted-foreground)] transition-colors hover:bg-[var(--muted)] hover:text-[var(--foreground)] sm:flex">
          {{ t('home.docs') }}
        </a>
        <LocaleSwitcher />
        <button @click="toggleTheme"
          class="rounded-full p-2 text-[var(--muted-foreground)] transition-colors hover:bg-[var(--muted)] hover:text-[var(--foreground)]">
          <Icon v-if="isDark" name="sun" size="sm" />
          <Icon v-else name="moon" size="sm" />
        </button>
        <router-link v-if="isAuthenticated" :to="dashboardPath"
          class="btn btn-primary ml-1 hidden sm:inline-flex">
          {{ t('home.dashboard') }}
          <Icon name="arrowRight" size="xs" :stroke-width="2" />
        </router-link>
        <router-link v-else to="/login"
          class="btn btn-primary ml-1 hidden sm:inline-flex">
          {{ t('home.login') }}
        </router-link>
        <button @click="mobileMenuOpen = !mobileMenuOpen"
          class="rounded-full p-2 text-[var(--muted-foreground)] transition-colors hover:bg-[var(--muted)] hover:text-[var(--foreground)] sm:hidden">
          <Icon v-if="mobileMenuOpen" name="x" size="sm" />
          <Icon v-else name="menu" size="sm" />
        </button>
      </div>
    </nav>
    <div v-if="mobileMenuOpen" class="border-t border-[var(--border)] bg-[var(--card)] px-4 pb-4 pt-2 sm:hidden">
      <div class="flex flex-col gap-1">
        <router-link v-for="link in visibleNavLinks" :key="link.path" :to="link.path"
          @click="mobileMenuOpen = false"
          :class="[
            'rounded-xl px-3 py-2.5 text-sm transition-colors',
            activePath === link.path
              ? 'bg-[var(--accent)] font-medium text-[var(--accent-foreground)]'
              : 'text-[var(--muted-foreground)] hover:bg-[var(--muted)] hover:text-[var(--foreground)]'
          ]">
          {{ link.label }}
        </router-link>
        <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer"
          class="rounded-xl px-3 py-2.5 text-sm text-[var(--muted-foreground)] transition-colors hover:bg-[var(--muted)] hover:text-[var(--foreground)]">
          {{ t('home.docs') }}
        </a>
        <div class="my-1 border-t border-[var(--border)]"></div>
        <router-link v-if="isAuthenticated" :to="dashboardPath" @click="mobileMenuOpen = false"
          class="btn btn-primary w-full">
          {{ t('home.dashboard') }}
          <Icon name="arrowRight" size="xs" :stroke-width="2" />
        </router-link>
        <router-link v-else to="/login" @click="mobileMenuOpen = false"
          class="btn btn-primary w-full">
          {{ t('home.login') }}
        </router-link>
      </div>
    </div>
  </header>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { useAuthStore } from '@/stores/auth'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'

type NavLinkKey = 'leaderboard' | 'keyUsage' | 'monitoring' | 'pricing'
type NavLinkVisibility = Partial<Record<NavLinkKey, boolean>>

const props = withDefaults(defineProps<{
  activePath?: string
  navLinkVisibility?: NavLinkVisibility
}>(), {
  navLinkVisibility: () => ({}),
})

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()

const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API')
const siteLogo = computed(() => appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '')
const docUrl = computed(() => appStore.cachedPublicSettings?.doc_url || appStore.docUrl || '')
const isAuthenticated = computed(() => authStore.isAuthenticated)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => isAdmin.value ? '/admin/dashboard' : '/dashboard')

const isDark = ref(document.documentElement.classList.contains('dark'))
const mobileMenuOpen = ref(false)

const navLinks = computed(() => [
  { key: 'leaderboard' as const, path: '/leaderboard', label: t('leaderboard.title') },
  { key: 'keyUsage' as const, path: '/key-usage', label: t('home.keyUsage') },
  { key: 'monitoring' as const, path: '/monitoring', label: t('admin.monitoring.title') },
  { key: 'pricing' as const, path: '/pricing', label: t('pricing.title') },
])

const resolvedNavLinkVisibility = computed<Record<NavLinkKey, boolean>>(() => {
  const settings = appStore.cachedPublicSettings
  const legacyEnabled = settings?.home_nav_links_enabled !== false
  const resolve = (settingsValue: boolean | undefined, overrideValue: boolean | undefined) => overrideValue ?? settingsValue ?? legacyEnabled

  return {
    leaderboard: resolve(settings?.home_nav_leaderboard_enabled, props.navLinkVisibility.leaderboard),
    keyUsage: resolve(settings?.home_nav_key_usage_enabled, props.navLinkVisibility.keyUsage),
    monitoring: resolve(settings?.home_nav_monitoring_enabled, props.navLinkVisibility.monitoring),
    pricing: resolve(settings?.home_nav_pricing_enabled, props.navLinkVisibility.pricing),
  }
})

const visibleNavLinks = computed(() => navLinks.value.filter(link => resolvedNavLinkVisibility.value[link.key] !== false))

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

onMounted(() => {
  const savedTheme = localStorage.getItem('theme')
  if (savedTheme === 'dark' || (!savedTheme && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
    isDark.value = true
    document.documentElement.classList.add('dark')
  }
  authStore.checkAuth()
  appStore.fetchPublicSettings()
})
</script>
