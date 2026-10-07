<template>
  <div :class="cardClasses">
    <!-- Header Slot -->
    <div v-if="$slots.header || title" :class="headerClasses">
      <slot name="header">
        <h3 v-if="title" class="text-lg font-semibold text-gray-900 dark:text-white">
          {{ title }}
        </h3>
      </slot>
    </div>

    <!-- Body Slot -->
    <div :class="bodyClasses">
      <slot></slot>
    </div>

    <!-- Footer Slot -->
    <div v-if="$slots.footer" :class="footerClasses">
      <slot name="footer"></slot>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  variant?: 'default' | 'glass' | 'flat'
  padding?: 'none' | 'sm' | 'md' | 'lg'
  title?: string
  noDivider?: boolean
  class?: string
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'default',
  padding: 'md',
  noDivider: false
})

const cardClasses = computed(() => [
  'card',
  props.variant === 'glass' && 'card-glass',
  props.variant === 'flat' && 'border-0 shadow-none bg-transparent',
  props.class
])

const headerClasses = computed(() => [
  !props.noDivider && 'border-b border-gray-100 dark:border-dark-700',
  props.padding === 'none' && 'p-0',
  props.padding === 'sm' && 'px-4 py-3',
  props.padding === 'md' && 'px-6 py-4',
  props.padding === 'lg' && 'px-8 py-6',
])

const bodyClasses = computed(() => [
  props.padding === 'none' && 'p-0',
  props.padding === 'sm' && 'p-4',
  props.padding === 'md' && 'p-6',
  props.padding === 'lg' && 'p-8',
])

const footerClasses = computed(() => [
  !props.noDivider && 'border-t border-gray-100 dark:border-dark-700',
  props.padding === 'none' && 'p-0',
  props.padding === 'sm' && 'px-4 py-3',
  props.padding === 'md' && 'px-6 py-4',
  props.padding === 'lg' && 'px-8 py-6',
])
</script>
