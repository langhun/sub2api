import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import CodeFormatSettingsEditor from '../CodeFormatSettingsEditor.vue'
import type { CodeFormatSettings } from '../../settings'

function settings(): CodeFormatSettings {
  const rule = { prefix: 'CODE', character_set: 'alphanumeric' as const, separator: '-', group_length: 4, group_count: 3 }
  return { balance: { ...rule }, concurrency: { ...rule }, subscription: { ...rule }, invitation: { ...rule }, redpacket: { ...rule } }
}

function mountEditor(modelValue = settings()) {
  return mount(CodeFormatSettingsEditor, { props: { modelValue }, global: { plugins: [createI18n({ legacy: false, locale: 'en', missingWarn: false, fallbackWarn: false, messages: { en: { admin: { codeFormat: { combinationSpace: 'Available combinations: {count}' } } } } })] } })
}

describe('CodeFormatSettingsEditor', () => {
  it('renders a live formatted preview and accepts valid rules', () => { const wrapper = mountEditor(); expect(wrapper.text()).toContain('CODE-'); expect(wrapper.emitted('validity')?.at(-1)).toEqual([true]) })
  it('omits hexadecimal and emits changed settings for saving', async () => { const wrapper = mountEditor(); expect(wrapper.findAll('option').some(option => option.attributes('value') === 'hex')).toBe(false); await wrapper.find('select').setValue('numeric'); const updated = wrapper.emitted('update:modelValue')?.at(-1)?.[0] as CodeFormatSettings; expect(updated.balance.character_set).toBe('numeric') })
  it('rejects prefixes containing the configured separator', async () => { const modelValue = settings(); modelValue.balance.prefix = 'BAD-PREFIX'; const wrapper = mountEditor(modelValue); await wrapper.vm.$nextTick(); expect(wrapper.emitted('validity')?.at(-1)).toEqual([false]) })
  it('accepts the backend maximum of 16 groups', () => { const modelValue = settings(); modelValue.balance.group_count = 16; const wrapper = mountEditor(modelValue); expect(wrapper.emitted('validity')?.at(-1)).toEqual([true]) })
  it('supports a custom single-character separator and shows combination space', async () => { const modelValue = settings(); modelValue.balance.separator = '.'; const wrapper = mountEditor(modelValue); expect(wrapper.text()).toContain('CODE.'); expect(wrapper.find('input[maxlength="1"]').exists()).toBe(true); expect(wrapper.text()).toContain('admin.codeFormat.combinationSpace') })
})
