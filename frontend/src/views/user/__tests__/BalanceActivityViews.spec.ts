import { flushPromises, mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import RedPacketView from '@/views/user/RedPacketView.vue'
import TransferView from '@/views/user/TransferView.vue'

const api = vi.hoisted(() => ({
  createActivityIdempotencyKey: vi.fn((scope: string) => `${scope}-stable-key`),
  getTransferStats: vi.fn(), getTransferHistory: vi.fn(), resolveTransferReceiver: vi.fn(), validateTransfer: vi.fn(), transferBalance: vi.fn(),
  getMyRedPackets: vi.fn(), getRedPacketDetail: vi.fn(), createRedPacket: vi.fn(), claimRedPacket: vi.fn(),
}))
const authStore = vi.hoisted(() => ({ user: { id: 1, balance: 125.5 }, refreshUser: vi.fn().mockResolvedValue(undefined) }))
const appStore = vi.hoisted(() => ({ showSuccess: vi.fn(), showError: vi.fn() }))

vi.mock('@/api/transfer', () => api)
vi.mock('@/stores/auth', () => ({ useAuthStore: () => authStore }))
vi.mock('@/stores/app', () => ({ useAppStore: () => appStore }))

const AppLayoutStub = defineComponent({ template: '<main><slot /></main>' })
const BaseDialogStub = defineComponent({ props: { show: Boolean }, template: '<section v-if="show"><slot /></section>' })
const ConfirmDialogStub = defineComponent({ props: { show: Boolean }, template: '<section v-if="show"><slot /></section>' })

function mountView(component: typeof TransferView | typeof RedPacketView) {
  return mount(component, {
    global: {
      plugins: [createI18n({
        legacy: false,
        locale: 'en',
        missingWarn: false,
        fallbackWarn: false,
        messages: {
          en: {
            transfer: { receiverResolved: 'Selected {recipient}' },
            redpacket: {
              remaining: '{remaining}/{total} portions left',
              validUntil: 'Valid until {date}',
              status: { active: 'Active' },
            },
          },
        },
      })],
      stubs: { AppLayout: AppLayoutStub, BaseDialog: BaseDialogStub, ConfirmDialog: ConfirmDialogStub, Icon: true, LoadingSpinner: true, Pagination: true },
    },
  })
}

describe('TransferView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    authStore.user = { id: 1, balance: 125.5 }
    api.getTransferStats.mockResolvedValue({ total_sent: 50, total_received: 12, total_fee_paid: 1.5 })
    api.getTransferHistory.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 10 })
    api.resolveTransferReceiver.mockResolvedValue({ receiver_id: 7, receiver_display: 'a***e' })
    api.validateTransfer.mockResolvedValue({ fee: 1, gross_amount: 11, receiver_id: 7, receiver_display: 'a***e' })
  })

  it('renders the balance hero and loads sender history by default', async () => {
    const wrapper = mountView(TransferView)
    await flushPromises()
    expect(wrapper.text()).toContain('$125.50')
    expect(wrapper.text()).toContain('$50.00')
    expect(wrapper.text()).toContain('$12.00')
    expect(wrapper.text()).toContain('$1.50')
    expect(api.getTransferHistory).toHaveBeenCalledWith({ role: 'sender', page: 1, page_size: 10 })
  })

  it('resolves a receiver and validates the fee preview', async () => {
    const wrapper = mountView(TransferView)
    await flushPromises()
    await wrapper.find('input[type="text"]').setValue('alice@example.com')
    const searchButton = wrapper.findAll('button').find((button) => button.text().includes('common.search'))
    expect(searchButton).toBeDefined()
    await searchButton!.trigger('click')
    await flushPromises()
    await wrapper.find('input[type="number"]').setValue('10')
    const feeButton = wrapper.findAll('button').find((button) => button.text().includes('transfer.calculateFee'))
    expect(feeButton).toBeDefined()
    await feeButton!.trigger('click')
    await flushPromises()
    expect(api.resolveTransferReceiver).toHaveBeenCalledWith('alice@example.com')
    expect(api.validateTransfer).toHaveBeenCalledWith(7, 10)
    expect(wrapper.find('[data-testid="resolved-receiver"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('$11.00')
  })

  it('uses the receiver role for incoming history', async () => {
    const wrapper = mountView(TransferView)
    await flushPromises()
    const receivedButton = wrapper.findAll('button').find((button) => button.text().includes('transfer.received'))
    expect(receivedButton).toBeDefined()
    await receivedButton!.trigger('click')
    await flushPromises()
    expect(api.getTransferHistory).toHaveBeenLastCalledWith({ role: 'receiver', page: 1, page_size: 10 })
  })

  it('discards a stale receiver response after the query changes', async () => {
    let resolveAlice!: (value: { receiver_id: number; receiver_display: string }) => void
    let resolveBob!: (value: { receiver_id: number; receiver_display: string }) => void
    api.resolveTransferReceiver.mockImplementation((query: string) => new Promise((resolve) => {
      if (query === 'alice') resolveAlice = resolve
      else resolveBob = resolve
    }))
    const wrapper = mountView(TransferView)
    await flushPromises()
    const input = wrapper.find('input[type="text"]')
    const searchButton = () => wrapper.findAll('button').find((button) => button.text().includes('common.search'))!

    await input.setValue('alice')
    await searchButton().trigger('click')
    await input.setValue('bob')
    await searchButton().trigger('click')
    resolveAlice({ receiver_id: 7, receiver_display: 'a***e' })
    await flushPromises()
    expect(wrapper.find('[data-testid="resolved-receiver"]').exists()).toBe(false)

    resolveBob({ receiver_id: 8, receiver_display: 'b*b' })
    await flushPromises()
    expect((wrapper.vm as unknown as { receiver: { receiver_id: number } }).receiver.receiver_id).toBe(8)
  })

  it('reuses the same transfer operation key after a failed response', async () => {
    api.transferBalance.mockRejectedValue(new Error('timeout'))
    const wrapper = mountView(TransferView)
    await flushPromises()
    await wrapper.find('input[type="text"]').setValue('alice')
    await wrapper.findAll('button').find((button) => button.text().includes('common.search'))!.trigger('click')
    await flushPromises()
    await wrapper.find('input[type="number"]').setValue('10')

    await (wrapper.vm as unknown as { submitTransfer: () => Promise<void> }).submitTransfer()
    await (wrapper.vm as unknown as { submitTransfer: () => Promise<void> }).submitTransfer()

    expect(api.transferBalance).toHaveBeenCalledTimes(2)
    expect(api.transferBalance.mock.calls[0][3]).toBe('balance-transfer-stable-key')
    expect(api.transferBalance.mock.calls[1][3]).toBe('balance-transfer-stable-key')
  })
})

describe('RedPacketView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    authStore.user = { id: 1, balance: 88.25 }
    api.getMyRedPackets.mockResolvedValue({
      items: [{ id: 9, sender_id: 1, total_amount: 20, total_count: 4, remaining_amount: 10, remaining_count: 2, redpacket_type: 'random', fee: 0, fee_rate: 0, code: 'RP-TEST-CODE', status: 'active', memo: 'Best wishes', expire_at: '2026-07-12T00:00:00Z', created_at: '2026-07-11T00:00:00Z' }],
      total: 1, page: 1, page_size: 10,
    })
    api.getRedPacketDetail.mockResolvedValue({ redpacket: {}, claims: [{ id: 3, redpacket_id: 9, user_id: 7, amount: 5, transfer_id: 12, created_at: '2026-07-11T01:00:00Z' }] })
  })

  it('renders the hero, action entries, packet code, and claim progress', async () => {
    const wrapper = mountView(RedPacketView)
    await flushPromises()
    expect(wrapper.text()).toContain('$88.25')
    expect(wrapper.text()).toContain('redpacket.create')
    expect(wrapper.text()).toContain('redpacket.claim')
    expect(wrapper.text()).toContain('RP-TEST-CODE')
    const progressBar = wrapper.findAll('div').find((node) => node.attributes('style')?.includes('width: 50%'))
    expect(progressBar).toBeDefined()
  })

  it('loads permission-filtered claim details when a packet expands', async () => {
    const wrapper = mountView(RedPacketView)
    await flushPromises()
    await wrapper.find('article > button').trigger('click')
    await flushPromises()
    expect(api.getRedPacketDetail).toHaveBeenCalledWith(9)
    expect(wrapper.text()).toContain('#7')
    expect(wrapper.text()).toContain('$5.00')
  })

  it('reuses operation keys when red packet writes are retried after a timeout', async () => {
    api.createRedPacket.mockRejectedValue(new Error('timeout'))
    api.claimRedPacket.mockRejectedValue(new Error('timeout'))
    const wrapper = mountView(RedPacketView)
    await flushPromises()
    const view = wrapper.vm as unknown as {
      createForm: { total_amount: number; count: number; redpacket_type: 'equal' | 'random'; memo: string }
      claimCode: string
      handleCreate: () => Promise<void>
      handleClaim: () => Promise<void>
    }
    view.createForm.total_amount = 10
    view.claimCode = '  rp-Lower-code  '

    await view.handleCreate()
    await view.handleCreate()
    await view.handleClaim()
    await view.handleClaim()

    expect(api.createRedPacket.mock.calls[0][1]).toBe('redpacket-create-stable-key')
    expect(api.createRedPacket.mock.calls[1][1]).toBe('redpacket-create-stable-key')
    expect(api.claimRedPacket.mock.calls[0][1]).toBe('redpacket-claim-stable-key')
    expect(api.claimRedPacket.mock.calls[1][1]).toBe('redpacket-claim-stable-key')
    expect(api.claimRedPacket.mock.calls[0][0]).toBe('rp-Lower-code')
    expect(api.claimRedPacket.mock.calls[1][0]).toBe('rp-Lower-code')
  })
})
