import { describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { mount } from '@vue/test-utils'

import SubscriptionSourceDialog from '../SubscriptionSourceDialog.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

const BaseDialogStub = defineComponent({
  name: 'BaseDialog',
  props: {
    show: {
      type: Boolean,
      default: false
    },
    title: {
      type: String,
      default: ''
    },
    width: {
      type: String,
      default: ''
    }
  },
  emits: ['close'],
  template: `
    <div v-if="show" data-testid="base-dialog" :data-title="title" :data-width="width">
      <button type="button" data-testid="dialog-close" @click="$emit('close')">close</button>
      <slot />
      <slot name="footer" />
    </div>
  `
})

const SelectStub = defineComponent({
  name: 'SelectStub',
  props: {
    modelValue: {
      type: [String, Number, Boolean, null],
      default: ''
    },
    options: {
      type: Array,
      default: () => []
    }
  },
  emits: ['update:modelValue'],
  template: `
    <select
      data-testid="format-select"
      :value="modelValue"
      @change="$emit('update:modelValue', $event.target.value)"
    >
      <option v-for="option in options" :key="option.value" :value="option.value">
        {{ option.label }}
      </option>
    </select>
  `
})

function buildForm() {
  return {
    name: 'Main Feed',
    url: 'https://example.com/feed.yaml',
    source_format: 'auto',
    enabled: true,
    refresh_interval_hours: 6,
    target_entry_count: 3,
    auto_add_to_pool: false
  }
}

const formatOptions = [
  { value: 'auto', label: 'Auto' },
  { value: 'clash_yaml', label: 'Clash YAML' }
]

function mountDialog(overrides: Record<string, unknown> = {}) {
  return mount(SubscriptionSourceDialog, {
    props: {
      show: true,
      editing: false,
      submitting: false,
      form: buildForm(),
      formatOptions,
      ...overrides
    },
    global: {
      stubs: {
        BaseDialog: BaseDialogStub,
        Select: SelectStub
      }
    }
  })
}

function getLastFormUpdate(wrapper: ReturnType<typeof mount>) {
  const events = wrapper.emitted('update:form') || []
  return events[events.length - 1]?.[0]
}

describe('SubscriptionSourceDialog', () => {
  it('renders create and edit states and emits close and submit', async () => {
    const wrapper = mountDialog()

    expect(wrapper.get('[data-testid="base-dialog"]').attributes('data-title')).toBe('admin.proxies.subscriptions.createTitle')
    expect(wrapper.get('[data-testid="base-dialog"]').attributes('data-width')).toBe('normal')
    expect(wrapper.get('button.btn-primary').text()).toBe('common.create')

    await wrapper.get('button[data-testid="dialog-close"]').trigger('click')
    await wrapper.get('button.btn-secondary').trigger('click')
    await wrapper.get('form#create-subscription-form').trigger('submit.prevent')

    expect(wrapper.emitted('close')).toHaveLength(2)
    expect(wrapper.emitted('submit')).toHaveLength(1)

    await wrapper.setProps({
      editing: true,
      submitting: true
    })

    expect(wrapper.get('[data-testid="base-dialog"]').attributes('data-title')).toBe('admin.proxies.subscriptions.editTitle')
    expect(wrapper.get('button.btn-primary').text()).toBe('common.submitting')
    expect(wrapper.get('button.btn-primary').attributes('disabled')).toBeDefined()
  })

  it('emits update:form for text, select, number, and checkbox fields', async () => {
    const wrapper = mountDialog()

    await wrapper.get('input[type="text"]').setValue('Backup Feed')
    expect(getLastFormUpdate(wrapper)).toMatchObject({
      name: 'Backup Feed'
    })
    await wrapper.setProps({ form: getLastFormUpdate(wrapper) })

    await wrapper.get('input[type="url"]').setValue('https://example.com/backup.yaml')
    expect(getLastFormUpdate(wrapper)).toMatchObject({
      name: 'Backup Feed',
      url: 'https://example.com/backup.yaml'
    })
    await wrapper.setProps({ form: getLastFormUpdate(wrapper) })

    await wrapper.get('[data-testid="format-select"]').setValue('clash_yaml')
    expect(getLastFormUpdate(wrapper)).toMatchObject({
      source_format: 'clash_yaml'
    })
    await wrapper.setProps({ form: getLastFormUpdate(wrapper) })

    const numberInputs = wrapper.findAll('input[type="number"]')
    await numberInputs[0].setValue('12')
    expect(getLastFormUpdate(wrapper)).toMatchObject({
      refresh_interval_hours: 12
    })
    await wrapper.setProps({ form: getLastFormUpdate(wrapper) })

    await numberInputs[1].setValue('5')
    expect(getLastFormUpdate(wrapper)).toMatchObject({
      target_entry_count: 5
    })
    await wrapper.setProps({ form: getLastFormUpdate(wrapper) })

    const checkboxInputs = wrapper.findAll('input[type="checkbox"]')
    await checkboxInputs[0].setValue(false)
    expect(getLastFormUpdate(wrapper)).toMatchObject({
      enabled: false
    })
    await wrapper.setProps({ form: getLastFormUpdate(wrapper) })

    await checkboxInputs[1].setValue(true)
    expect(getLastFormUpdate(wrapper)).toMatchObject({
      auto_add_to_pool: true
    })
  })
})
