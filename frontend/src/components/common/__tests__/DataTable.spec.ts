import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import DataTable from '../DataTable.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

describe('DataTable sorting', () => {
  it('keeps empty values at the end for both ascending and descending sorts', async () => {
    const win = window as any
    win.matchMedia = vi.fn().mockReturnValue({
      matches: true,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
    })
    const globalObject = globalThis as any
    globalObject.ResizeObserver = class {
      observe() {}
      disconnect() {}
    }

    const wrapper = mount(DataTable, {
      props: {
        columns: [
          { key: 'name', label: 'Name', sortable: true },
        ],
        data: [
          { id: 1, name: null },
          { id: 2, name: 'b' },
          { id: 3, name: 'a' },
          { id: 4, name: '' },
        ],
      },
    })

    const sortableHeader = wrapper.find('th')
    await sortableHeader.trigger('click')

    const setupState = (wrapper.vm as any).$?.setupState ?? (wrapper.vm as any)
    expect(setupState.sortedData.map((row: { name: string | null }) => row.name ?? '')).toEqual(['a', 'b', '', ''])

    await sortableHeader.trigger('click')

    expect(setupState.sortedData.map((row: { name: string | null }) => row.name ?? '')).toEqual(['b', 'a', '', ''])
  })

  it('humanizes fallback labels instead of showing raw i18n keys', async () => {
    const win = window as any
    win.matchMedia = vi.fn().mockReturnValue({
      matches: true,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
    })
    const globalObject = globalThis as any
    globalObject.ResizeObserver = class {
      observe() {}
      disconnect() {}
    }

    const wrapper = mount(DataTable, {
      props: {
        columns: [
          { key: 'created_at', label: 'admin.accounts.columns.createdAt', sortable: true },
          { key: 'status', label: 'Status', sortable: true },
        ],
        data: [
          { id: 1, created_at: '2026-05-01T00:00:00Z', status: 'active' },
        ],
      },
    })

    const headers = wrapper.findAll('th')
    expect(headers[0].text()).toContain('Created At')
    expect(headers[1].text()).toContain('Status')
  })
})
