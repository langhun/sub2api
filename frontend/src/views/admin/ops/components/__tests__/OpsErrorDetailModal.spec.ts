import { describe, it, expect, beforeEach, vi } from 'vitest'
import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import OpsErrorDetailModal from '../OpsErrorDetailModal.vue'

const mockGetRequestErrorDetail = vi.fn()
const mockListRequestErrorUpstreamErrors = vi.fn()
const mockShowError = vi.fn()

vi.mock('@/api/admin/ops', () => ({
  opsAPI: {
    getRequestErrorDetail: (...args: any[]) => mockGetRequestErrorDetail(...args),
    listRequestErrorUpstreamErrors: (...args: any[]) => mockListRequestErrorUpstreamErrors(...args),
  },
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    showError: mockShowError,
  }),
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

vi.mock('@/utils/format', () => ({
  formatDateTime: (value: string) => value,
}))

vi.mock('../utils/errorDetailResponse', () => ({
  resolvePrimaryResponseBody: () => '{"error":"bad request"}',
  resolveUpstreamPayload: (detail: { upstream_error_detail?: string; error_body?: string }) =>
    detail.upstream_error_detail || detail.error_body || '',
}))

const BaseDialogStub = defineComponent({
  name: 'BaseDialogStub',
  props: {
    show: { type: Boolean, default: false },
    title: { type: String, default: '' },
  },
  emits: ['close'],
  template: '<div v-if="show"><slot /></div>',
})

const baseDetail = {
  id: 11,
  created_at: '2026-05-21T00:00:00Z',
  phase: 'request',
  type: 'request_error',
  error_owner: 'client',
  error_source: 'client_request',
  severity: 'error',
  status_code: 400,
  platform: 'openai',
  model: 'gpt-4o',
  resolved: false,
  client_request_id: 'client-1',
  request_id: 'request-1',
  message: 'request failed',
  user_email: 'user@example.com',
  account_name: '',
  group_name: 'default',
  error_body: '{"error":"bad request"}',
  user_agent: 'test-agent',
  is_business_limited: false,
}

describe('OpsErrorDetailModal', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockGetRequestErrorDetail.mockResolvedValue(baseDetail)
  })

  it('loads every page of correlated upstream errors for request details', async () => {
    mockListRequestErrorUpstreamErrors
      .mockResolvedValueOnce({
        items: [
          {
            ...baseDetail,
            id: 101,
            phase: 'upstream',
            error_owner: 'provider',
            type: 'upstream_error',
            status_code: 502,
            message: 'upstream page 1',
            upstream_error_detail: '{"page":1}',
          },
        ],
        total: 2,
        page: 1,
        page_size: 100,
        pages: 2,
      })
      .mockResolvedValueOnce({
        items: [
          {
            ...baseDetail,
            id: 102,
            phase: 'upstream',
            error_owner: 'provider',
            type: 'upstream_error',
            status_code: 503,
            message: 'upstream page 2',
            upstream_error_detail: '{"page":2}',
          },
        ],
        total: 2,
        page: 2,
        page_size: 100,
        pages: 2,
      })

    const wrapper = mount(OpsErrorDetailModal, {
      props: {
        show: true,
        errorId: 11,
        errorType: 'request',
      },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          Icon: true,
        },
      },
    })

    await flushPromises()

    expect(mockGetRequestErrorDetail).toHaveBeenCalledWith(11)
    expect(mockListRequestErrorUpstreamErrors).toHaveBeenNthCalledWith(
      1,
      11,
      { page: 1, page_size: 100, view: 'all' },
      { include_detail: true },
    )
    expect(mockListRequestErrorUpstreamErrors).toHaveBeenNthCalledWith(
      2,
      11,
      { page: 2, page_size: 100, view: 'all' },
      { include_detail: true },
    )
    expect(wrapper.text()).toContain('upstream page 1')
    expect(wrapper.text()).toContain('upstream page 2')
  })
})
