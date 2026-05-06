<template>
  <div
    v-if="visibleLinks.length > 0"
    :class="containerClass"
  >
    <template v-if="variant === 'menu'">
      <router-link
        v-for="link in visibleLinks"
        :key="link.path"
        :to="link.path"
        :class="[
          'dropdown-item',
          route.path === link.path
            ? 'bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300'
            : ''
        ]"
        @click="emit('navigate')"
      >
        <Icon :name="link.icon" size="sm" />
        {{ link.label }}
      </router-link>
    </template>
    <div v-else :class="linksClass">
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
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import Icon from '@/components/icons/Icon.vue'

type PublicLinkKey = 'leaderboard' | 'keyUsage' | 'monitoring' | 'pricing'

const props = withDefaults(defineProps<{
  inline?: boolean
  variant?: 'default' | 'menu'
}>(), {
  inline: false,
  variant: 'default',
})
const emit = defineEmits<{
  (e: 'navigate'): void
}>()

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
  { key: 'leaderboard' as const, path: '/leaderboard', label: t('leaderboard.title'), icon: 'badge' as const },
  { key: 'keyUsage' as const, path: '/key-usage', label: t('home.keyUsage'), icon: 'chartBar' as const },
  { key: 'monitoring' as const, path: '/monitoring', label: t('admin.monitoring.title'), icon: 'server' as const },
  { key: 'pricing' as const, path: '/pricing', label: t('pricing.title'), icon: 'calculator' as const },
])

const visibleLinks = computed(() => links.value.filter((link) => visibility.value[link.key] !== false))

const containerClass = computed(() => props.variant === 'menu'
  ? 'py-1'
  : props.inline
  ? 'w-full'
  : 'border-b border-gray-200/60 bg-white/75 px-4 backdrop-blur dark:border-dark-800/60 dark:bg-dark-950/75 md:px-6')

const linksClass = computed(() => props.inline
  ? 'flex items-center justify-end gap-2 overflow-x-auto whitespace-nowrap'
  : 'flex min-h-12 items-center gap-2 overflow-x-auto py-2')
</script>
