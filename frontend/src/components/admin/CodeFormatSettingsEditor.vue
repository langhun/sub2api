<template>
  <div class="card overflow-hidden p-0">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.codeFormat.title') }}</h2>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.codeFormat.description') }}</p>
    </div>
    <div class="grid gap-4 p-6 md:grid-cols-2 xl:grid-cols-3">
      <fieldset v-for="kind in kinds" :key="kind.key" class="rounded-lg border border-gray-200 p-4 dark:border-dark-700">
        <legend class="px-1 text-sm font-semibold text-gray-900 dark:text-white">{{ kind.label }}</legend>
        <div class="space-y-3">
          <div class="rounded-md bg-gray-50 px-3 py-2 dark:bg-dark-800">
            <p class="truncate font-mono text-sm font-medium text-primary-700 dark:text-primary-300">{{ preview(modelValue[kind.key]) }}</p>
            <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ t('admin.codeFormat.length', { length: preview(modelValue[kind.key]).length }) }}</p>
          </div>
          <label class="block"><span class="input-label">{{ t('admin.codeFormat.prefix') }}</span><input :value="modelValue[kind.key].prefix" class="input" type="text" maxlength="12" :placeholder="kind.placeholder" @input="updateText(kind.key, 'prefix', $event)" /></label>
          <label class="block"><span class="input-label">{{ t('admin.codeFormat.characterSet') }}</span><select :value="modelValue[kind.key].character_set" class="input" @change="updateText(kind.key, 'character_set', $event)"><option v-for="set in characterSets" :key="set.value" :value="set.value">{{ set.label }}</option></select></label>
          <div class="grid grid-cols-3 gap-2">
            <label><span class="input-label">{{ t('admin.codeFormat.separator') }}</span><select :value="separatorMode(modelValue[kind.key].separator)" class="input" @change="updateSeparatorMode(kind.key, $event)"><option value="">{{ t('admin.codeFormat.none') }}</option><option value="-">-</option><option value="_">_</option><option value="custom">{{ t('admin.codeFormat.custom') }}</option></select></label>
            <label><span class="input-label">{{ t('admin.codeFormat.groupLength') }}</span><input :value="modelValue[kind.key].group_length" class="input" type="number" min="1" max="32" @input="updateNumber(kind.key, 'group_length', $event)" /></label>
            <label><span class="input-label">{{ t('admin.codeFormat.groupCount') }}</span><input :value="modelValue[kind.key].group_count" class="input" type="number" min="1" max="16" @input="updateNumber(kind.key, 'group_count', $event)" /></label>
          </div>
          <label v-if="separatorMode(modelValue[kind.key].separator) === 'custom'" class="block"><span class="input-label">{{ t('admin.codeFormat.customSeparator') }}</span><input :value="modelValue[kind.key].separator" class="input" type="text" maxlength="1" @input="updateText(kind.key, 'separator', $event)" /></label>
          <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.codeFormat.combinationSpace', { count: combinationSpace(modelValue[kind.key]) }) }}</p>
          <p v-if="ruleError(modelValue[kind.key])" class="text-xs text-red-600 dark:text-red-400">{{ ruleError(modelValue[kind.key]) }}</p>
        </div>
      </fieldset>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue'; import { useI18n } from 'vue-i18n'; import type { CodeCharacterSet, CodeFormatRule, CodeFormatSettings } from '@/api/admin/settings'
const props = defineProps<{ modelValue: CodeFormatSettings }>(); const emit = defineEmits<{ 'update:modelValue': [value: CodeFormatSettings]; validity: [value: boolean] }>(); const { t } = useI18n()
const kinds = computed(() => [{ key: 'balance' as const, label: t('admin.codeFormat.balance'), placeholder: 'BAL' }, { key: 'concurrency' as const, label: t('admin.codeFormat.concurrency'), placeholder: 'CON' }, { key: 'subscription' as const, label: t('admin.codeFormat.subscription'), placeholder: 'SUB' }, { key: 'invitation' as const, label: t('admin.codeFormat.invitation'), placeholder: 'INV' }, { key: 'redpacket' as const, label: t('admin.codeFormat.redpacket'), placeholder: 'RP' }])
const characterSets = computed((): { value: CodeCharacterSet; label: string }[] => [{ value: 'uppercase', label: t('admin.codeFormat.uppercase') }, { value: 'numeric', label: t('admin.codeFormat.numeric') }, { value: 'alphanumeric', label: t('admin.codeFormat.alphanumeric') }])
type CodeKind = keyof CodeFormatSettings
function updateRule(kind: CodeKind, patch: Partial<CodeFormatRule>) { emit('update:modelValue', { ...props.modelValue, [kind]: { ...props.modelValue[kind], ...patch } }) }
function updateText(kind: CodeKind, field: 'prefix' | 'separator' | 'character_set', event: Event) { updateRule(kind, { [field]: (event.target as HTMLInputElement).value.trim() }) }
function updateNumber(kind: CodeKind, field: 'group_length' | 'group_count', event: Event) { updateRule(kind, { [field]: Number((event.target as HTMLInputElement).value) }) }
function separatorMode(separator: string) { return ['', '-', '_'].includes(separator) ? separator : 'custom' }
function updateSeparatorMode(kind: CodeKind, event: Event) { const value = (event.target as HTMLSelectElement).value; updateRule(kind, { separator: value === 'custom' ? '.' : value }) }
function sample(set: CodeCharacterSet) { return { uppercase: 'ABCDEFGH', numeric: '27490163', alphanumeric: 'A7K2P9Q4' }[set] }
function preview(rule: CodeFormatRule) { const source = sample(rule.character_set) || 'ABCDEFGH'; const group = source.repeat(Math.ceil(rule.group_length / source.length)).slice(0, Math.max(1, rule.group_length)); const body = Array.from({ length: Math.max(1, rule.group_count) }, () => group).join(rule.separator); return `${rule.prefix}${rule.prefix && body ? rule.separator : ''}${body}` }
function combinationSpace(rule: CodeFormatRule) { const alphabet = { uppercase: 26, numeric: 10, alphanumeric: 36 }[rule.character_set] || 0; const exponent = Math.max(0, rule.group_length * rule.group_count); if (!alphabet || !exponent) return '0'; const logarithm = exponent * Math.log10(alphabet); return logarithm < 15 ? Math.pow(alphabet, exponent).toLocaleString() : `10^${logarithm.toFixed(1)}` }
function ruleError(rule: CodeFormatRule) { if (!['uppercase', 'numeric', 'alphanumeric'].includes(rule.character_set)) return t('admin.codeFormat.invalidCharset'); if (!Number.isInteger(rule.group_length) || rule.group_length < 1 || rule.group_length > 32) return t('admin.codeFormat.invalidGroupLength'); if (!Number.isInteger(rule.group_count) || rule.group_count < 1 || rule.group_count > 16) return t('admin.codeFormat.invalidGroupCount'); if (rule.separator.length > 1 || (rule.separator && !/^[\x21-\x7E]$/.test(rule.separator))) return t('admin.codeFormat.invalidSeparator'); if (rule.prefix && (!/^[\x21-\x7E]+$/.test(rule.prefix) || (rule.separator && rule.prefix.includes(rule.separator)))) return t('admin.codeFormat.invalidPrefix'); return '' }
watch(() => props.modelValue, (value) => { emit('validity', Object.values(value).every(rule => !ruleError(rule))) }, { deep: true, immediate: true })
</script>
