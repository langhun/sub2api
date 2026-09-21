import { ref } from 'vue'
import { flushPromises, mount, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ActiveProbeSourceBadge from '../ActiveProbeSourceBadge.vue'
import ActiveProbeSettingsExtension from '../ActiveProbeSettingsExtension.vue'
import {
  channelMonitorV2SettingsExtensions,
  channelMonitorV2SourceExtensions,
} from '../extensions'
import { getActiveProbeConfig, updateActiveProbeConfig } from '../api'

const { showError, showSuccess } = vi.hoisted(() => ({
  showError: vi.fn(),
  showSuccess: vi.fn(),
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    locale: ref('zh-CN'),
    t: (key: string, params?: Record<string, unknown>) => params?.time ? `${key}:${params.time}` : key,
  }),
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError, showSuccess }),
}))

vi.mock('../api', () => ({
  getActiveProbeConfig: vi.fn(),
  updateActiveProbeConfig: vi.fn(),
}))

describe('channel monitor V2 active-probe Overlay', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('registers one settings mount and one source mount', () => {
    expect(channelMonitorV2SettingsExtensions.map((item) => item.id)).toEqual(['channel-monitor-v2-active-probes'])
    expect(channelMonitorV2SourceExtensions.map((item) => item.id)).toEqual(['channel-monitor-v2-active-probes'])
  })

  it('renders only namespaced source data and distinguishes healthy from failed probes', async () => {
    const wrapper = mount(ActiveProbeSourceBadge, {
      props: {
        extensions: {
          unrelated: { active_probe: true },
          channel_monitor_v2_active_probes: {
            active_probe: true,
            passive_traffic: true,
            active_probe_status: 'operational',
            last_active_probe_at: '2026-09-19T10:00:00Z',
          },
        },
        admin: true,
      },
    })
    expect(wrapper.text()).toContain('channelMonitorV2ActiveProbes.source.active')
    expect(wrapper.text()).toContain('channelMonitorV2ActiveProbes.source.passive')
    expect(wrapper.find('.badge-success').exists()).toBe(true)

    await wrapper.setProps({
      extensions: {
        channel_monitor_v2_active_probes: { active_probe: true, active_probe_status: 'degraded' },
      },
    })
    expect(wrapper.find('.badge-warning').exists()).toBe(true)

    await wrapper.setProps({
      extensions: {
        channel_monitor_v2_active_probes: { active_probe: true, active_probe_status: 'failed' },
      },
    })
    expect(wrapper.find('.badge-danger').exists()).toBe(true)

    await wrapper.setProps({ extensions: { unrelated: { active_probe: true } } })
    expect(wrapper.text()).toBe('')

    await wrapper.setProps({
      admin: false,
      extensions: {
        channel_monitor_v2_active_probes: { active_probe: true, passive_traffic: true },
      },
    })
    expect(wrapper.text()).toBe('')
  })

  it('loads and saves the disabled-by-default admin switch', async () => {
    vi.mocked(getActiveProbeConfig).mockResolvedValue({ enabled: false })
    vi.mocked(updateActiveProbeConfig).mockResolvedValue({ enabled: true })
    const wrapper = shallowMount(ActiveProbeSettingsExtension, {
      global: {
        stubs: {
          Icon: true,
          Toggle: {
            name: 'Toggle',
            props: ['modelValue', 'disabled'],
            emits: ['update:modelValue'],
            template: '<button data-test="toggle" />',
          },
        },
      },
    })
    await flushPromises()

    expect(getActiveProbeConfig).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('channelMonitorV2ActiveProbes.settings.off')
    wrapper.findComponent({ name: 'Toggle' }).vm.$emit('update:modelValue', true)
    await flushPromises()

    expect(updateActiveProbeConfig).toHaveBeenCalledWith({ enabled: true })
    expect(showSuccess).toHaveBeenCalledWith('channelMonitorV2ActiveProbes.settings.saved')
    expect(wrapper.text()).toContain('channelMonitorV2ActiveProbes.settings.on')

    await wrapper.find('button').trigger('click')
    expect(wrapper.emitted('open-targets')).toHaveLength(1)
  })
})
