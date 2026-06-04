<template>
  <div class="mt-5 grid grid-cols-2 gap-2">
    <div
      class="feature-panel-info rounded-xl border p-3"
    >
      <div
        class="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-sky-600 dark:text-sky-300"
      >
        <Icon :name="primaryIcon" size="xs" />
        <span>{{ primaryLabel }}</span>
      </div>
      <div class="mt-1.5 text-lg font-bold font-mono tabular-nums text-gray-900 dark:text-gray-100">
        {{ primaryValue }}<span class="text-xs font-normal text-gray-400 ml-0.5">{{ primaryUnit }}</span>
      </div>
    </div>
    <div
      class="feature-panel-warning rounded-xl border p-3"
    >
      <div
        class="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-amber-600 dark:text-amber-300"
      >
        <Icon :name="secondaryIcon" size="xs" />
        <span>{{ secondaryLabel }}</span>
      </div>
      <div class="mt-1.5 text-lg font-bold font-mono tabular-nums text-gray-900 dark:text-gray-100">
        {{ secondaryValue }}<span class="text-xs font-normal text-gray-400 ml-0.5">{{ secondaryUnit }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import Icon from '@/components/icons/Icon.vue'

defineProps<{
  primaryLabel: string
  primaryValue: string
  primaryUnit: string
  primaryIcon: 'bolt' | 'globe' | 'clock' | 'link'
  secondaryLabel: string
  secondaryValue: string
  secondaryUnit: string
  secondaryIcon: 'bolt' | 'globe' | 'clock' | 'link'
}>()
</script>
