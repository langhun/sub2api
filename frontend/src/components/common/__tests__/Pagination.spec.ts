import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import Pagination from '../Pagination.vue'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

describe('Pagination', () => {
  it('accepts the numeric value produced by the jump input', async () => {
    const wrapper = mount(Pagination, { props: { page: 1, pageSize: 10, total: 100, showJump: true } })
    await wrapper.get('input[type="number"]').setValue('7')
    await wrapper.get('input[type="number"]').trigger('keyup', { key: 'Enter' })
    expect(wrapper.emitted('update:page')).toEqual([[7]])
    expect((wrapper.get('input').element as HTMLInputElement).value).toBe('')
    wrapper.unmount()
  })

  it('honors explicit page size options and preserves the current size', () => {
    const wrapper = mount(Pagination, { props: { page: 1, pageSize: 20, total: 100, pageSizeOptions: [10, 50] } })
    expect(wrapper.findComponent({ name: 'Select' }).props('options')).toEqual([
      { value: 10, label: '10' }, { value: 20, label: '20' }, { value: 50, label: '50' },
    ])
    wrapper.unmount()
  })
})
