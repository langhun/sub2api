import { flushPromises, mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import RedPacketView from '@/custom/modules/activity/views/RedPacketView.vue'

const redPacketApi = vi.hoisted(() => ({
  getMyRedPackets: vi.fn(), getRedPacketDetail: vi.fn(), createRedPacket: vi.fn(), claimRedPacket: vi.fn(),
}))
const activityApi = vi.hoisted(() => ({
  createActivityIdempotencyKey: vi.fn((scope: string) => `${scope}-stable-key`),
}))
const authStore = vi.hoisted(() => ({ user: { id: 1, balance: 88.25 }, refreshUser: vi.fn().mockResolvedValue(undefined) }))
const appStore = vi.hoisted(() => ({ showSuccess: vi.fn(), showError: vi.fn() }))

vi.mock('@/custom/modules/activity/api/redpacket', () => redPacketApi)
vi.mock('@/custom/modules/activity/api/idempotency', () => activityApi)
vi.mock('@/stores/auth', () => ({ useAuthStore: () => authStore }))
vi.mock('@/stores/app', () => ({ useAppStore: () => appStore }))

const AppLayoutStub = defineComponent({ template: '<main><slot /></main>' })
const BaseDialogStub = defineComponent({ props: { show: Boolean }, template: '<section v-if="show"><slot /></section>' })

function mountView() {
  return mount(RedPacketView, {
    global: {
      plugins: [createI18n({
        legacy: false,
        locale: 'en',
        missingWarn: false,
        fallbackWarn: false,
        messages: {
          en: {
            redpacket: {
              remaining: '{remaining}/{total} portions left',
              validUntil: 'Valid until {date}',
              status: { active: 'Active' },
            },
          },
        },
      })],
      stubs: { AppLayout: AppLayoutStub, BaseDialog: BaseDialogStub, Icon: true, LoadingSpinner: true, Pagination: true },
    },
  })
}

describe('RedPacketView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    authStore.user = { id: 1, balance: 88.25 }
    redPacketApi.getMyRedPackets.mockResolvedValue({
      items: [{ id: 9, sender_id: 1, total_amount: 20, total_count: 4, remaining_amount: 10, remaining_count: 2, redpacket_type: 'random', fee: 0, fee_rate: 0, code: 'RP-TEST-CODE', status: 'active', memo: 'Best wishes', expire_at: '2026-07-12T00:00:00Z', created_at: '2026-07-11T00:00:00Z' }],
      total: 1, page: 1, page_size: 10,
    })
    redPacketApi.getRedPacketDetail.mockResolvedValue({ redpacket: {}, claims: [{ id: 3, redpacket_id: 9, user_id: 7, user_display: 'Bob', amount: 5, transfer_id: 12, created_at: '2026-07-11T01:00:00Z' }] })
  })

  it('renders the hero, action entries, packet code, and claim progress', async () => {
    const wrapper = mountView()
    await flushPromises()
    expect(wrapper.text()).toContain('$88.25')
    expect(wrapper.text()).toContain('redpacket.create')
    expect(wrapper.text()).toContain('redpacket.claim')
    expect(wrapper.text()).toContain('RP-TEST-CODE')
    const progressBar = wrapper.findAll('div').find((node) => node.attributes('style')?.includes('width: 50%'))
    expect(progressBar).toBeDefined()
  })

  it('loads permission-filtered claim details when a packet expands', async () => {
    const wrapper = mountView()
    await flushPromises()
    await wrapper.find('article > button').trigger('click')
    await flushPromises()
    expect(redPacketApi.getRedPacketDetail).toHaveBeenCalledWith(9)
    expect(wrapper.text()).toContain('Bob')
    expect(wrapper.text()).not.toContain('#7')
    expect(wrapper.text()).toContain('$5.00')
  })

  it('reuses operation keys when red packet writes are retried after a timeout', async () => {
    redPacketApi.createRedPacket.mockRejectedValue(new Error('timeout'))
    redPacketApi.claimRedPacket.mockRejectedValue(new Error('timeout'))
    const wrapper = mountView()
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

    expect(redPacketApi.createRedPacket.mock.calls[0][1]).toBe('redpacket-create-stable-key')
    expect(redPacketApi.createRedPacket.mock.calls[1][1]).toBe('redpacket-create-stable-key')
    expect(redPacketApi.claimRedPacket.mock.calls[0][1]).toBe('redpacket-claim-stable-key')
    expect(redPacketApi.claimRedPacket.mock.calls[1][1]).toBe('redpacket-claim-stable-key')
    expect(redPacketApi.claimRedPacket.mock.calls[0][0]).toBe('rp-Lower-code')
    expect(redPacketApi.claimRedPacket.mock.calls[1][0]).toBe('rp-Lower-code')
  })
})
