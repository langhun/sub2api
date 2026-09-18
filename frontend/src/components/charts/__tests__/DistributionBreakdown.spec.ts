import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ModelDistributionChart from '../ModelDistributionChart.vue'
import GroupDistributionChart from '../GroupDistributionChart.vue'
import EndpointDistributionChart from '../EndpointDistributionChart.vue'

const { getUserBreakdown } = vi.hoisted(() => ({ getUserBreakdown: vi.fn() }))
vi.mock('@/api/admin/dashboard', () => ({ getUserBreakdown }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('vue-chartjs', () => ({ Doughnut: { template: '<div />' } }))

const metrics = { requests: 2, total_tokens: 10, actual_cost: 1, cost: 1 }
const cases = [
  { name: 'model', component: ModelDistributionChart, props: { modelStats: [
    { ...metrics, model: 'first', input_tokens: 10, output_tokens: 0, cache_creation_tokens: 0, cache_read_tokens: 0 },
    { ...metrics, model: 'second', input_tokens: 10, output_tokens: 0, cache_creation_tokens: 0, cache_read_tokens: 0 },
  ] } },
  { name: 'group', component: GroupDistributionChart, props: { groupStats: [
    { ...metrics, group_id: 1, group_name: 'first' },
    { ...metrics, group_id: 2, group_name: 'second' },
  ] } },
  { name: 'endpoint', component: EndpointDistributionChart, props: { endpointStats: [
    { ...metrics, endpoint: 'first' },
    { ...metrics, endpoint: 'second' },
  ] } },
]

function deferred() {
  let resolve!: (value: unknown) => void
  let reject!: (error: Error) => void
  const promise = new Promise((onResolve, onReject) => { resolve = onResolve; reject = onReject })
  return { promise, resolve, reject }
}

const response = (email: string) => ({ users: [{ ...metrics, user_id: 1, email }] })

describe.each(cases)('$name distribution breakdown', ({ component, props }) => {
  beforeEach(() => getUserBreakdown.mockReset())

  const render = () => mount(component as typeof ModelDistributionChart, {
    props: { ...props, startDate: '2026-09-17', endDate: '2026-09-18' } as any,
    global: { stubs: { LoadingSpinner: { template: '<span data-test="loading" />' } } },
  })

  it('ignores an older row response while the current row is still loading', async () => {
    const first = deferred()
    const second = deferred()
    getUserBreakdown.mockReturnValueOnce(first.promise).mockReturnValueOnce(second.promise)
    const wrapper = render()
    await wrapper.get('td[title="first"]').trigger('click')
    await wrapper.get('td[title="second"]').trigger('click')

    first.resolve(response('stale@example.com'))
    await flushPromises()
    expect(wrapper.find('[data-test="loading"]').exists()).toBe(true)
    expect(wrapper.text()).not.toContain('stale@example.com')

    second.resolve(response('current@example.com'))
    await flushPromises()
    expect(wrapper.text()).toContain('current@example.com')
    expect(wrapper.find('[data-test="loading"]').exists()).toBe(false)
    wrapper.unmount()
  })

  it('invalidates pending details when date filters change', async () => {
    const request = deferred()
    getUserBreakdown.mockReturnValueOnce(request.promise)
    const wrapper = render()
    await wrapper.get('td[title="first"]').trigger('click')
    await wrapper.setProps({ startDate: '2026-09-16' })
    request.resolve(response('old-range@example.com'))
    await flushPromises()
    expect(wrapper.text()).not.toContain('old-range@example.com')
    expect(wrapper.find('[data-test="loading"]').exists()).toBe(false)
    wrapper.unmount()
  })

  it('shows failed requests as errors and can retry the same row', async () => {
    getUserBreakdown.mockRejectedValueOnce(new Error('offline')).mockResolvedValueOnce(response('retry@example.com'))
    const wrapper = render()
    await wrapper.get('td[title="first"]').trigger('click')
    await flushPromises()
    expect(wrapper.get('[role="alert"]').text()).toContain('admin.dashboard.failedToLoad')
    expect(wrapper.get('[role="alert"]').text()).not.toContain('admin.dashboard.noDataAvailable')

    await wrapper.get('[role="alert"] button').trigger('click')
    await flushPromises()
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('retry@example.com')
    expect(getUserBreakdown).toHaveBeenCalledTimes(2)
    wrapper.unmount()
  })
})
