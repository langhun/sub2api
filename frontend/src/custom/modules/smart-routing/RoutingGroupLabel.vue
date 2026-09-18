<template>
  <span class="routing-group-label" :title="label">
    <PlatformIcon v-if="platform" :platform="platform" size="sm" />
    <span class="routing-group-name">{{ label }}</span>
    <span v-if="rate != null" class="routing-group-rate">{{ rate }}x</span>
  </span>
</template>

<script setup lang="ts">
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import type { GroupPlatform } from '@/types'
defineProps<{ label: string; platform?: GroupPlatform; rate?: number }>()
</script>

<style scoped>
.routing-group-label { display: inline-flex; min-width: 0; max-width: 100%; align-items: center; gap: 6px; padding: 3px 6px; border-radius: 5px; background: var(--route-tint); color: var(--route-accent); font-size: 12px; line-height: 1.55; }
.routing-group-label > :not(.routing-group-name) { flex-shrink: 0; }
.routing-group-name { min-width: 0; overflow-wrap: anywhere; }
.routing-group-rate { flex-shrink: 0; padding: 1px 4px; border-radius: 3px; background: rgb(9 157 145 / 10%); font-size: 11px; font-weight: 600; font-variant-numeric: tabular-nums; }
</style>
