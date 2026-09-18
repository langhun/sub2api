<template>
  <div class="w-full" :class="$attrs.class" :style="$attrs.style as any">
    <label v-if="label" :for="inputId" class="input-label mb-1.5 block">
      {{ label }}
      <span v-if="required" class="text-red-500">*</span>
    </label>
    <div class="relative">
      <textarea
        v-bind="inputAttrs()"
        :id="inputId"
        :aria-invalid="error ? true : undefined"
        :aria-describedby="[inputAttrs()['aria-describedby'], error || hint ? `${inputId}-description` : undefined].filter(Boolean).join(' ') || undefined"
        ref="textAreaRef"
        :value="modelValue"
        :disabled="disabled"
        :required="required"
        :placeholder="placeholderText"
        :readonly="readonly"
        :rows="rows"
        :class="[
          'input w-full min-h-[80px] transition-all duration-200 resize-y',
          error ? 'input-error ring-2 ring-red-500/20' : '',
          disabled ? 'cursor-not-allowed bg-gray-100 opacity-60 dark:bg-dark-900' : ''
        ]"
        @input="onInput"
        @change="$emit('change', ($event.target as HTMLTextAreaElement).value)"
        @blur="$emit('blur', $event)"
        @focus="$emit('focus', $event)"
      ></textarea>
    </div>
    <!-- Hint / Error Text -->
    <p v-if="error" :id="`${inputId}-description`" class="input-error-text mt-1.5">
      {{ error }}
    </p>
    <p v-else-if="hint" :id="`${inputId}-description`" class="input-hint mt-1.5">
      {{ hint }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, useAttrs, useId } from 'vue'

defineOptions({ inheritAttrs: false })
const attrs = useAttrs()
const generatedId = useId()
const inputAttrs = () => Object.fromEntries(Object.entries(attrs).filter(([key]) => key !== 'class' && key !== 'style'))

interface Props {
  modelValue: string | null | undefined
  label?: string
  placeholder?: string
  disabled?: boolean
  required?: boolean
  readonly?: boolean
  error?: string
  hint?: string
  id?: string
  rows?: number | string
}

const props = withDefaults(defineProps<Props>(), {
  disabled: false,
  required: false,
  readonly: false,
  rows: 3
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'change', value: string): void
  (e: 'blur', event: FocusEvent): void
  (e: 'focus', event: FocusEvent): void
}>()

const textAreaRef = ref<HTMLTextAreaElement | null>(null)
const inputId = computed(() => props.id || generatedId)
const placeholderText = computed(() => props.placeholder || '')

const onInput = (event: Event) => {
  const value = (event.target as HTMLTextAreaElement).value
  emit('update:modelValue', value)
}

// Expose focus method
defineExpose({
  focus: () => textAreaRef.value?.focus(),
  select: () => textAreaRef.value?.select()
})
</script>
