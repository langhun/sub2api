<template>
  <nav
    v-if="visibleLinks.length > 0"
    class="border-b border-gray-200/60 bg-white/75 px-4 backdrop-blur dark:border-dark-800/60 dark:bg-dark-950/75 md:px-6"
  >
    <div class="flex min-h-12 items-center gap-2 overflow-x-auto py-2">
      <router-link
        v-for="link in visibleLinks"
        :key="link.path"
        :to="link.path"
        :class="[
          'shrink-0 rounded-lg px-3 py-1.5 text-sm font-medium transition-colors',
          route.path === link.path
            ? 'bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300'
            : 'text-gray-600 hover:bg-gray-100 hover:text-gray-900 dark:text-dark-300 dark:hover:bg-dark-800 dark:hover:text-white'
        ]"
      >
        {{ link.label }}
      </router-link>
    </div>
  </nav>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'

type PublicLinkKey = 'leaderboard' | 'keyUsage' | 'monitoring' | 'pricing'

const route = useRoute()
const { t } = useI18n()
const appStore = useAppStore()

const visibility = computed<Partial<Record<PublicLinkKey, boolean>>>(() => {
  const settings = appStore.cachedPublicSettings
  const legacyEnabled = settings?.home_nav_links_enabled !== false
  const resolve = (value?: boolean) => value ?? legacyEnabled

  return {
    leaderboard: resolve(settings?.home_nav_leaderboard_enabled),
    keyUsage: resolve(settings?.home_nav_key_usage_enabled),
    monitoring: resolve(settings?.home_nav_monitoring_enabled),
    pricing: resolve(settings?.home_nav_pricing_enabled),
  }
})

const links = computed(() => [
  { key: 'leaderboard' as const, path: '/leaderboard', label: t('leaderboard.title') },
  { key: 'keyUsage' as const, path: '/key-usage', label: t('home.keyUsage') },
  { key: 'monitoring' as const, path: '/monitoring', label: t('admin.monitoring.title') },
  { key: 'pricing' as const, path: '/pricing', label: t('pricing.title') },
])

const visibleLinks = computed(() => links.value.filter((link) => visibility.value[link.key] !== false))
</script>
