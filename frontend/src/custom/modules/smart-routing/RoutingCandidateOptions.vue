<template>
  <div class="routing-candidate-options">
    <div v-if="platform" class="routing-candidate-platform">{{ platform }}</div>
    <div ref="choices" class="routing-candidate-choices" role="group" :aria-label="t('keys.selectRoutingCandidates')">
      <slot />
      <label v-for="option in options" :key="option.value" class="routing-candidate-choice" :class="{ disabled }">
        <input type="checkbox" :checked="isSelected(option.value)" :disabled="disabled" :aria-label="option.label" @change="selectOption(option.value)" />
        <RoutingGroupLabel :label="option.label" :platform="option.platform" :rate="option.rate" />
      </label>
      <p v-if="!options.length && !selectedLabels?.length" class="routing-candidate-no-match">{{ t('keys.routingCandidatesNoMatch') }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { nextTick, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import RoutingGroupLabel from './RoutingGroupLabel.vue'
import type { GroupPlatform } from '@/types'

const props = defineProps<{
  options: Array<{ value: number; label: string; description: string; platform?: GroupPlatform; rate?: number }>
  platform?: string
  selectedLabels?: string[]
  selectedValues?: number[]
  disabled: boolean
}>()
const emit = defineEmits<{ (event: 'select', value: number): void }>()
const { t } = useI18n()
const choices = ref<HTMLElement | null>(null)

function isSelected(value: number): boolean {
  return props.selectedValues?.includes(value) ?? false
}

async function selectOption(value: number) {
  if (props.disabled) return
  const index = props.options.findIndex((option) => option.value === value)
  emit('select', value)
  await nextTick()
  // Selected rows move into the ordered list; keep keyboard selection usable.
  const remaining = choices.value?.querySelectorAll<HTMLInputElement>('.routing-candidate-choice input:not(:disabled)')
  const target = remaining?.[Math.min(index, remaining.length - 1)] ?? choices.value?.querySelector<HTMLInputElement>('input:not(:disabled)')
  target?.focus()
}
</script>

<style scoped>
.routing-candidate-options { margin-top: 12px; padding: 12px; border: 1px solid var(--route-border); border-radius: 8px; }
.routing-candidate-platform { margin-bottom: 10px; color: var(--route-muted); font-size: 12px; font-weight: 650; text-transform: uppercase; }
.routing-candidate-choices { display: grid; gap: 8px; max-height: 280px; margin-top: 8px; overflow-y: auto; scrollbar-width: thin; }
.routing-candidate-choice { display: flex; min-width: 0; min-height: 48px; align-items: center; gap: 10px; padding: 8px 10px; border: 1px solid var(--route-border); border-radius: 8px; background: var(--route-surface); cursor: pointer; }
.routing-candidate-choice:hover { border-color: var(--route-accent); background: var(--route-tint); }
.routing-candidate-choice.disabled { cursor: not-allowed; opacity: .55; }
.routing-candidate-choice input { flex: 0 0 16px; width: 16px; height: 16px; accent-color: var(--route-accent); }
.routing-candidate-choice input:focus-visible { outline: 2px solid var(--route-accent); outline-offset: 3px; }
.routing-candidate-no-match { color: var(--route-muted); font-size: 12px; line-height: 1.5; }
.routing-candidate-no-match { margin: 0; padding: 8px 0; }
</style>
