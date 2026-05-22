import { describe, expect, it, vi, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import UserAttributeForm from '../UserAttributeForm.vue'

const {
  listEnabledDefinitions,
  getUserAttributeValues,
} = vi.hoisted(() => ({
  listEnabledDefinitions: vi.fn(),
  getUserAttributeValues: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    userAttributes: {
      listEnabledDefinitions,
      getUserAttributeValues,
    },
  },
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

function deferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((res) => {
    resolve = res
  })
  return { promise, resolve }
}

describe('UserAttributeForm', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    listEnabledDefinitions.mockResolvedValue([])
  })

  it('ignores stale async user attribute loads after switching users', async () => {
    const first = deferred<Array<{ attribute_id: number; value: string }>>()
    const second = deferred<Array<{ attribute_id: number; value: string }>>()
    getUserAttributeValues
      .mockReturnValueOnce(first.promise)
      .mockReturnValueOnce(second.promise)

    const wrapper = mount(UserAttributeForm, {
      props: {
        userId: 1,
        modelValue: {},
      },
    })

    await wrapper.setProps({
      userId: 2,
      modelValue: {},
    })

    second.resolve([{ attribute_id: 2, value: 'user-2' }])
    await flushPromises()

    const setupState = (wrapper.vm as any).$?.setupState ?? (wrapper.vm as any)
    expect(setupState.localValues).toEqual({ 2: 'user-2' })

    first.resolve([{ attribute_id: 1, value: 'user-1' }])
    await flushPromises()

    expect(setupState.localValues).toEqual({ 2: 'user-2' })
    expect(wrapper.emitted('update:modelValue')?.at(-1)?.[0]).toEqual({ 2: 'user-2' })
  })

  it('shows load error state and retries when attribute definitions fail to load', async () => {
    listEnabledDefinitions.mockRejectedValueOnce(new Error('boom'))
    listEnabledDefinitions.mockResolvedValueOnce([])

    const wrapper = mount(UserAttributeForm, {
      props: {
        modelValue: {},
      },
    })

    await flushPromises()
    expect(wrapper.text()).toContain('admin.settings.attributes.failedToLoad')

    const retryButton = wrapper.get('button')
    expect(retryButton.text()).toBe('common.retry')

    await retryButton.trigger('click')
    await flushPromises()

    expect(listEnabledDefinitions).toHaveBeenCalledTimes(2)
    expect(wrapper.text()).not.toContain('admin.settings.attributes.failedToLoad')
  })
})
