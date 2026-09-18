import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import UserAllowedGroupsModal from '../UserAllowedGroupsModal.vue'
import UserApiKeysModal from '../UserApiKeysModal.vue'
import UserBalanceHistoryModal from '../UserBalanceHistoryModal.vue'

const api = vi.hoisted(() => ({
  listGroups: vi.fn(), getGroups: vi.fn(), updateUser: vi.fn(), getKeys: vi.fn(), getHistory: vi.fn()
}))
vi.mock('@/api/admin', () => ({ adminAPI: {
  groups: { list: api.listGroups, getAll: api.getGroups },
  users: { update: api.updateUser, getUserApiKeys: api.getKeys, getUserBalanceHistory: api.getHistory }
} }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showError: vi.fn(), showSuccess: vi.fn() }) }))
vi.mock('@/custom/registry', () => ({ customLedgerDisplay: { isBalanceType: () => false, historyTypeKey: () => '' } }))
vi.mock('vue-i18n', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue-i18n')>()),
  useI18n: () => ({ t: (key: string) => key, te: () => false })
}))

const user = (id: number) => ({ id, email: `user-${id}@example.test`, allowed_groups: [id], group_rates: {} }) as never
const group = (id: number) => ({ id, name: `group-${id}`, is_exclusive: true, status: 'active', subscription_type: 'standard', platform: 'openai', rate_multiplier: 1 })
const key = (id: number) => ({ id, name: `key-${id}`, key: `sk-${id}`, status: 'active' })
const history = (id: number) => ({ id, type: 'admin_balance', value: id, notes: `history-${id}` })
function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (error: Error) => void
  const promise = new Promise<T>((res, rej) => { resolve = res; reject = rej })
  return { promise, resolve, reject }
}
const global = { stubs: {
  BaseDialog: { props: ['show'], template: '<div v-if="show"><slot /><slot name="footer" /></div>' },
  Select: true, Icon: true, PlatformIcon: true, GroupBadge: true, GroupOptionItem: true
} }

describe('admin user modal loading boundaries', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    api.getGroups.mockResolvedValue([])
    api.updateUser.mockResolvedValue({})
  })
  afterEach(() => { vi.restoreAllMocks() })

  it('cannot overwrite group permissions before load succeeds or after load fails', async () => {
    const pending = deferred<never>()
    api.listGroups.mockReturnValueOnce(pending.promise)
    vi.spyOn(console, 'error').mockImplementation(() => {})
    const wrapper = mount(UserAllowedGroupsModal, { props: { show: true, user: user(1) }, global })
    const save = wrapper.get<HTMLButtonElement>('.btn-primary')
    expect(save.element.disabled).toBe(true)
    await save.trigger('click')
    expect(api.updateUser).not.toHaveBeenCalled()
    pending.reject(new Error('offline'))
    await flushPromises()
    expect(wrapper.get('[role="alert"]').text()).toContain('common.error')
    expect(save.element.disabled).toBe(true)
    api.listGroups.mockResolvedValueOnce({ items: [group(1)] })
    await wrapper.get('[role="alert"] button').trigger('click')
    await flushPromises()
    expect(save.element.disabled).toBe(false)
    await save.trigger('click')
    await flushPromises()
    expect(api.updateUser).toHaveBeenCalledWith(1, expect.objectContaining({ allowed_groups: [1] }))
    wrapper.unmount()
  })

  it('ignores group responses for the previously selected user', async () => {
    const first = deferred<{ items: ReturnType<typeof group>[] }>()
    api.listGroups.mockReturnValueOnce(first.promise).mockResolvedValueOnce({ items: [group(2)] })
    const wrapper = mount(UserAllowedGroupsModal, { props: { show: true, user: user(1) }, global })
    await wrapper.setProps({ user: user(2) })
    await flushPromises()
    first.resolve({ items: [group(1)] })
    await flushPromises()
    expect(wrapper.text()).toContain('group-2')
    expect(wrapper.text()).not.toContain('group-1')
    await wrapper.get('.btn-primary').trigger('click')
    await flushPromises()
    expect(api.updateUser).toHaveBeenCalledWith(2, expect.objectContaining({ allowed_groups: [2] }))
    wrapper.unmount()
  })

  it('does not show an old user API key list under a newly selected user', async () => {
    const first = deferred<{ items: ReturnType<typeof key>[] }>()
    api.getKeys.mockReturnValueOnce(first.promise).mockResolvedValueOnce({ items: [key(2)] })
    const wrapper = mount(UserApiKeysModal, { props: { show: true, user: user(1) }, global })
    await wrapper.setProps({ user: user(2) })
    await flushPromises()
    first.resolve({ items: [key(1)] })
    await flushPromises()
    expect(wrapper.text()).toContain('key-2')
    expect(wrapper.text()).not.toContain('key-1')
    wrapper.unmount()
  })

  it('does not show stale balance history after changing users', async () => {
    const first = deferred<{ items: ReturnType<typeof history>[] }>()
    api.getHistory.mockReturnValueOnce(first.promise).mockResolvedValueOnce({ items: [history(2)], total: 1 })
    const wrapper = mount(UserBalanceHistoryModal, { props: { show: true, user: user(1) }, global })
    await wrapper.setProps({ user: user(2) })
    await flushPromises()
    first.resolve({ items: [history(1)] })
    await flushPromises()
    expect(wrapper.text()).toContain('history-2')
    expect(wrapper.text()).not.toContain('history-1')
    wrapper.unmount()
  })
})
