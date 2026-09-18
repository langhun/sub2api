<template>
  <div class="flex min-w-0 flex-wrap items-baseline justify-between gap-x-2 gap-y-0.5">
    <span class="min-w-0 text-gray-500 dark:text-gray-400">{{ label }}</span>
    <span class="max-w-full break-words text-right font-mono tabular-nums">{{ display }}</span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { formatScaled } from '@/utils/pricing'

const props = withDefaults(
  defineProps<{
    label: string
    value: number | null
    unit: string
    scale: number
  }>(),
  { value: null }
)

const display = computed(() =>
  props.value == null ? '-' : `${formatScaled(props.value, props.scale)} ${props.unit}`
)
</script>
