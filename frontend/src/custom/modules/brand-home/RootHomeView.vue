<template>
  <main class="min-h-screen" aria-busy="true"></main>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores'

const router = useRouter()
const appStore = useAppStore()

onMounted(async () => {
  if (!appStore.publicSettingsLoaded) {
    try {
      await appStore.fetchPublicSettings()
    } catch {
      // Failed settings requests retain the backwards-compatible default page.
    }
  }

  const target = getDefaultHomepage(appStore.cachedPublicSettings) === 'dino' ? '/Dino' : '/home'
  await router.replace(target)
})

function getDefaultHomepage(settings: unknown): 'default' | 'dino' {
  if (typeof settings !== 'object' || settings === null) return 'default'
  return (settings as { default_homepage?: unknown }).default_homepage === 'dino'
    ? 'dino'
    : 'default'
}
</script>
