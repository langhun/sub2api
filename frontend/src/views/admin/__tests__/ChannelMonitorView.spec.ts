import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import ChannelMonitorView from '../ChannelMonitorView.vue'

const {
  listMonitors,
  listHistory,
  runNow,
  updateMonitor,
  deleteMonitor,
  showError,
  showSuccess,
} = vi.hoisted(() => ({
  listMonitors: vi.fn(),
  listHistory: vi.fn(),
  runNow: vi.fn(),
  updateMonitor: vi.fn(),
  deleteMonitor: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    channelMonitor: {
      list: listMonitors,
      listHistory,
      runNow,
      update: updateMonitor,
      del: deleteMonitor,
    },
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    cachedPublicSettings: {
      channel_monitor_enabled: true,
    },
    showError,
    showSuccess,
  }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

const DataTableStub = {
  props: ['columns', 'data', 'loading'],
  template: `
    <div data-test="data-table">
      <template v-if="data?.length">
        <div v-for="row in data" :key="row.id" class="monitor-row">
          <div data-test="row-name">
            <slot name="cell-name" :row="row" :value="row.name" />
          </div>
          <div data-test="row-provider">
            <slot name="cell-provider" :row="row" :value="row.provider" />
          </div>
          <div data-test="row-primary-model">
            <slot name="cell-primary_model" :row="row" :value="row.primary_model" />
          </div>
          <div data-test="row-availability">
            <slot name="cell-availability_7d" :row="row" :value="row.availability_7d" />
          </div>
          <div data-test="row-latest-check">
            <slot name="cell-latest_check" :row="row" :value="row.last_checked_at" />
          </div>
          <div data-test="row-latency">
            <slot name="cell-latency" :row="row" :value="row.primary_latency_ms" />
          </div>
          <div data-test="row-enabled">
            <slot name="cell-enabled" :row="row" :value="row.enabled" />
          </div>
          <div data-test="row-actions">
            <slot name="cell-actions" :row="row" :value="null" />
          </div>
        </div>
      </template>
      <template v-else>
        <slot name="empty" />
      </template>
    </div>
  `,
}

const SelectStub = {
  props: ['modelValue', 'options'],
  emits: ['update:modelValue'],
  template: `
    <select
      data-test="history-model-filter"
      :value="modelValue"
      @change="$emit('update:modelValue', $event.target.value)"
    >
      <option v-for="option in options" :key="String(option.value)" :value="option.value">
        {{ option.label }}
      </option>
    </select>
  `,
}

const BaseDialogStub = {
  props: ['show', 'title'],
  emits: ['close'],
  template: `
    <div v-if="show" data-test="history-dialog">
      <div data-test="history-title">{{ title }}</div>
      <slot />
      <slot name="footer" />
    </div>
  `,
}

const MonitorActionsCellStub = {
  props: ['row', 'running'],
  emits: ['run', 'history', 'edit', 'delete'],
  template: `
    <div>
      <button data-test="run-button" @click="$emit('run', row)">run</button>
      <button data-test="history-button" @click="$emit('history', row)">history</button>
      <button data-test="edit-button" @click="$emit('edit', row)">edit</button>
      <button data-test="delete-button" @click="$emit('delete', row)">delete</button>
    </div>
  `,
}

const ToggleStub = {
  props: ['modelValue'],
  emits: ['update:modelValue'],
  template: '<button data-test="toggle" @click="$emit(\'update:modelValue\', !modelValue)">{{ String(modelValue) }}</button>',
}

describe('admin ChannelMonitorView', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-05-14T12:10:00Z'))
    localStorage.clear()

    listMonitors.mockReset()
    listHistory.mockReset()
    runNow.mockReset()
    updateMonitor.mockReset()
    deleteMonitor.mockReset()
    showError.mockReset()
    showSuccess.mockReset()

    listMonitors.mockResolvedValue({
      items: [
        {
          id: 7,
          name: 'OpenAI 主线路',
          provider: 'openai',
          endpoint: 'https://api.example.com',
          api_key_masked: 'sk-***',
          api_key_decrypt_failed: false,
          primary_model: 'gpt-4o-mini',
          extra_models: ['gpt-4.1-mini'],
          group_name: 'main',
          enabled: true,
          interval_seconds: 60,
          last_checked_at: '2026-05-14T12:08:00Z',
          created_by: 1,
          created_at: '2026-05-14T10:00:00Z',
          updated_at: '2026-05-14T10:00:00Z',
          primary_status: 'operational',
          primary_latency_ms: 420,
          availability_7d: 99.25,
          extra_models_status: [
            { model: 'gpt-4.1-mini', status: 'degraded', latency_ms: 820 },
          ],
          template_id: null,
          extra_headers: {},
          body_override_mode: 'off',
          body_override: null,
        },
      ],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
    })

    listHistory.mockResolvedValue({
      items: [
        {
          id: 101,
          model: 'gpt-4o-mini',
          status: 'operational',
          latency_ms: 420,
          ping_latency_ms: 55,
          message: 'upstream HTTP 200',
          checked_at: '2026-05-14T12:08:00Z',
        },
      ],
    })
  })

  it('renders latest automatic check info in the list', async () => {
    const wrapper = mount(ChannelMonitorView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>' },
          DataTable: DataTableStub,
          Pagination: true,
          BaseDialog: BaseDialogStub,
          ConfirmDialog: true,
          EmptyState: { template: '<div><slot /></div>' },
          HelpTooltip: { template: '<div><slot name="trigger" /><slot /></div>' },
          Select: SelectStub,
          Icon: true,
          Toggle: ToggleStub,
          MonitorFiltersBar: { template: '<div data-test="filters"></div>' },
          MonitorFormDialog: true,
          MonitorTemplateManagerDialog: true,
          MonitorRunResultDialog: true,
          MonitorPrimaryModelCell: { props: ['row'], template: '<div>{{ row.primary_model }}</div>' },
          MonitorActionsCell: MonitorActionsCellStub,
          AutoRefreshButton: { template: '<div data-test="auto-refresh-button"></div>' },
        },
      },
    })

    await flushPromises()

    expect(listMonitors).toHaveBeenCalledTimes(1)
    expect(wrapper.get('[data-test="row-latest-check"]').text()).toContain('monitorCommon.relativeMinutesAgo')
    expect(wrapper.text()).toContain('99.25%')
    expect(wrapper.find('[data-test="auto-refresh-button"]').exists()).toBe(true)
  })

  it('loads and displays persisted history in the page dialog', async () => {
    const wrapper = mount(ChannelMonitorView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>' },
          DataTable: DataTableStub,
          Pagination: true,
          BaseDialog: BaseDialogStub,
          ConfirmDialog: true,
          EmptyState: { template: '<div><slot /></div>' },
          HelpTooltip: { template: '<div><slot name="trigger" /><slot /></div>' },
          Select: SelectStub,
          Icon: true,
          Toggle: ToggleStub,
          MonitorFiltersBar: { template: '<div data-test="filters"></div>' },
          MonitorFormDialog: true,
          MonitorTemplateManagerDialog: true,
          MonitorRunResultDialog: true,
          MonitorPrimaryModelCell: { props: ['row'], template: '<div>{{ row.primary_model }}</div>' },
          MonitorActionsCell: MonitorActionsCellStub,
          AutoRefreshButton: { template: '<div data-test="auto-refresh-button"></div>' },
        },
      },
    })

    await flushPromises()
    await wrapper.get('[data-test="history-button"]').trigger('click')
    await flushPromises()

    expect(listHistory).toHaveBeenCalledWith(7, { model: undefined, limit: 20 })
    expect(wrapper.get('[data-test="history-dialog"]').text()).toContain('upstream HTTP 200')
    expect(wrapper.get('[data-test="history-dialog"]').text()).toContain('420 ms')
  })
})
