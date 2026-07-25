import { flushPromises, mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import DirectTransferView from '../views/DirectTransferView.vue'

const api = vi.hoisted(() => {
  const createDirectTransferIdempotencyKey = vi.fn((scope: string) => `${scope}-stable-key`)
  const getDirectTransferStats = vi.fn()
  const getDirectTransferHistory = vi.fn()
  const searchDirectTransferReceivers = vi.fn()
  const validateDirectTransfer = vi.fn()
  const submitDirectTransfer = vi.fn()

  return {
    createDirectTransferIdempotencyKey,
    getDirectTransferStats,
    getDirectTransferHistory,
    searchDirectTransferReceivers,
    validateDirectTransfer,
    submitDirectTransfer,
    directTransferAPI: {
      searchTransferReceivers: searchDirectTransferReceivers,
      resolveTransferReceiver: vi.fn(),
      transferBalance: submitDirectTransfer,
      validateTransfer: validateDirectTransfer,
      getTransferHistory: getDirectTransferHistory,
      getTransferStats: getDirectTransferStats,
    },
  }
})
const authStore = vi.hoisted(() => ({ user: { id: 1, balance: 125.5 }, refreshUser: vi.fn().mockResolvedValue(undefined) }))
const appStore = vi.hoisted(() => ({ showSuccess: vi.fn(), showError: vi.fn() }))

vi.mock('../api', () => api)
vi.mock('@/stores/auth', () => ({ useAuthStore: () => authStore }))
vi.mock('@/stores/app', () => ({ useAppStore: () => appStore }))

const AppLayoutStub = defineComponent({ template: '<main><slot /></main>' })
const ConfirmDialogStub = defineComponent({ props: { show: Boolean }, template: '<section v-if="show"><slot /></section>' })

function mountView() {
  return mount(DirectTransferView, {
    global: {
      plugins: [createI18n({
        legacy: false,
        locale: 'en',
        missingWarn: false,
        fallbackWarn: false,
        messages: { en: { transfer: { receiverResolved: 'Selected {recipient}', toUser: 'To', fromUser: 'From' } } },
      })],
      stubs: { AppLayout: AppLayoutStub, ConfirmDialog: ConfirmDialogStub, Icon: true, LoadingSpinner: true, Pagination: true },
    },
  })
}

describe('DirectTransferView', () => {
  beforeEach(() => {
    vi.useRealTimers()
    vi.clearAllMocks()
    authStore.user = { id: 1, balance: 125.5 }
    api.getDirectTransferStats.mockResolvedValue({ total_sent: 50, total_received: 12, total_fee_paid: 1.5 })
    api.getDirectTransferHistory.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 10 })
    api.searchDirectTransferReceivers.mockResolvedValue([
      { receiver_id: 7, receiver_display: 'a***e', receiver_username: 'a***e', receiver_email: 'a***e@example.com' },
    ])
    api.validateDirectTransfer.mockResolvedValue({ fee: 1, gross_amount: 11, receiver_id: 7, receiver_display: 'a***e' })
  })

  it('renders the balance hero and loads sender history by default', async () => {
    const wrapper = mountView()
    await flushPromises()
    expect(wrapper.text()).toContain('$125.50')
    expect(wrapper.text()).toContain('$50.00')
    expect(wrapper.text()).toContain('$12.00')
    expect(wrapper.text()).toContain('$1.50')
    expect(api.getDirectTransferHistory).toHaveBeenCalledWith({ role: 'sender', page: 1, page_size: 10 })
  })

  it('renders the counterparty display name instead of a raw user id', async () => {
    api.getDirectTransferHistory.mockResolvedValue({
      items: [{ id: 1, sender_id: 1, receiver_id: 10, sender_display: 'Current User', receiver_display: 'Alice', amount: 5, fee: 0, fee_rate: 0, gross_amount: 5, transfer_type: 'direct', status: 'completed', memo: null, redpacket_id: null, created_at: '2026-07-11T00:00:00Z' }],
      total: 1, page: 1, page_size: 10,
    })
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('Alice')
    expect(wrapper.text()).not.toContain('#10')
  })

  it('shows receiver candidates and requires an explicit selection', async () => {
    vi.useFakeTimers()
    api.searchDirectTransferReceivers.mockResolvedValue([
      { receiver_id: 7, receiver_display: 'Alice', receiver_username: 'Alice', receiver_email: 'a***e@example.com' },
      { receiver_id: 8, receiver_display: 'Alex', receiver_username: 'Alex', receiver_email: 'a***x@example.com' },
    ])
    const wrapper = mountView()
    await flushPromises()
    const input = wrapper.get('[data-testid="receiver-search-input"]')
    await input.setValue('ali')
    expect(api.searchDirectTransferReceivers).not.toHaveBeenCalled()
    await vi.advanceTimersByTimeAsync(250)
    await flushPromises()
    expect(api.searchDirectTransferReceivers).toHaveBeenCalledWith('ali')
    expect(wrapper.findAll('[data-testid="receiver-candidate"]')).toHaveLength(2)
    expect(wrapper.text()).toContain('Alice')
    expect(wrapper.find('[data-testid="resolved-receiver"]').exists()).toBe(false)

    await wrapper.findAll('[data-testid="receiver-candidate"]')[0].trigger('click')
    expect(wrapper.find('[data-testid="receiver-candidates"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="resolved-receiver"]').exists()).toBe(true)

    await wrapper.find('input[type="number"]').setValue('10')
    const feeButton = wrapper.findAll('button').find((button) => button.text().includes('transfer.calculateFee'))
    expect(feeButton).toBeDefined()
    await feeButton!.trigger('click')
    await flushPromises()
    expect(api.validateDirectTransfer).toHaveBeenCalledWith(7, 10)
    expect(wrapper.text()).toContain('$11.00')

    await input.setValue('alex')
    expect(wrapper.find('[data-testid="resolved-receiver"]').exists()).toBe(false)
    expect((wrapper.vm as unknown as { receiver: unknown }).receiver).toBeNull()
  })

  it('uses the receiver role for incoming history', async () => {
    const wrapper = mountView()
    await flushPromises()
    const receivedButton = wrapper.findAll('button').find((button) => button.text().includes('transfer.received'))
    expect(receivedButton).toBeDefined()
    await receivedButton!.trigger('click')
    await flushPromises()
    expect(api.getDirectTransferHistory).toHaveBeenLastCalledWith({ role: 'receiver', page: 1, page_size: 10 })
  })

  it('discards a stale receiver response after the query changes', async () => {
    vi.useFakeTimers()
    type Receiver = { receiver_id: number; receiver_display: string; receiver_username: string; receiver_email: string }
    let resolveAlice!: (value: Receiver[]) => void
    let resolveBob!: (value: Receiver[]) => void
    api.searchDirectTransferReceivers.mockImplementation((query: string) => new Promise((resolve) => {
      if (query === 'alice') resolveAlice = resolve
      else resolveBob = resolve
    }))
    const wrapper = mountView()
    await flushPromises()
    const input = wrapper.get('[data-testid="receiver-search-input"]')

    await input.setValue('alice')
    await vi.advanceTimersByTimeAsync(250)
    await input.setValue('bob')
    await vi.advanceTimersByTimeAsync(250)
    resolveAlice([{ receiver_id: 7, receiver_display: 'a***e', receiver_username: 'a***e', receiver_email: 'a***e@example.com' }])
    await flushPromises()
    expect(wrapper.findAll('[data-testid="receiver-candidate"]')).toHaveLength(0)

    resolveBob([{ receiver_id: 8, receiver_display: 'b*b', receiver_username: 'b*b', receiver_email: 'b*b@example.com' }])
    await flushPromises()
    const candidates = wrapper.findAll('[data-testid="receiver-candidate"]')
    expect(candidates).toHaveLength(1)
    expect(candidates[0].text()).toContain('b*b')
    expect((wrapper.vm as unknown as { receiver: unknown }).receiver).toBeNull()
  })

  it('reuses the same transfer operation key after a failed response', async () => {
    vi.useFakeTimers()
    api.submitDirectTransfer.mockRejectedValue(new Error('timeout'))
    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('[data-testid="receiver-search-input"]').setValue('alice')
    await vi.advanceTimersByTimeAsync(250)
    await flushPromises()
    await wrapper.get('[data-testid="receiver-candidate"]').trigger('click')
    await wrapper.find('input[type="number"]').setValue('10')

    await (wrapper.vm as unknown as { submitTransfer: () => Promise<void> }).submitTransfer()
    await (wrapper.vm as unknown as { submitTransfer: () => Promise<void> }).submitTransfer()

    expect(api.submitDirectTransfer).toHaveBeenCalledTimes(2)
    expect(api.submitDirectTransfer.mock.calls[0][3]).toBe('balance-transfer-stable-key')
    expect(api.submitDirectTransfer.mock.calls[1][3]).toBe('balance-transfer-stable-key')
  })
})
