<template>
  <span
    :class="[
      'inline-flex items-center',
      sizeClass,
      variantClass,
      props.class
    ]"
  >
    <slot />
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  variant?: 'primary' | 'success' | 'warning' | 'danger' | 'gray' | 'purple' | 'blue'
  size?: 'sm' | 'md' | 'lg'
  class?: string
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'gray',
  size: 'md'
})

const sizeClass = computed(() => {
  switch (props.size) {
    case 'sm':
      // 紧凑模式(表格) - rounded-md px-1.5 py-px text-[10px]
      return 'gap-0.5 rounded-md px-1.5 py-px text-[10px] font-medium leading-tight'
    case 'lg':
      // 大尺寸 - rounded-full px-3 py-1 text-sm
      return 'gap-1.5 rounded-full px-3 py-1 text-sm font-semibold'
    case 'md':
    default:
      // 标准模式 - rounded-full px-2.5 py-0.5 text-xs
      return 'gap-1 rounded-full px-2.5 py-0.5 text-xs font-semibold'
  }
})

const variantClass = computed(() => {
  switch (props.variant) {
    case 'primary':
      return 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-400'
    case 'success':
      return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
    case 'warning':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400'
    case 'danger':
      return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'
    case 'purple':
      return 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400'
    case 'blue':
      return 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400'
    case 'gray':
    default:
      return 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-dark-300'
  }
})
</script>
