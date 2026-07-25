import { flushPromises, mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'

const api = vi.hoisted(() => ({
  batchDistribute: vi.fn(),
  freezeTransfer: vi.fn(),
  getFeeStats: vi.fn(),
  listTransfers: vi.fn(),
  revokeTransfer: vi.fn(),
}))

vi.mock('../api/admin', () => ({ adminTransferAPI: api }))

import TransferManageView from '../views/TransferManageView.vue'

const AppLayoutStub = defineComponent({ template: '<main><slot /></main>' })

function mountView() {
  return mount(TransferManageView, {
    global: {
      plugins: [createI18n({
        legacy: false,
        locale: 'zh',
        missingWarn: false,
        fallbackWarn: false,
        messages: { zh: { nav: { transferManage: '转账管理' }, transfer: { transferTypes: { direct: '余额转账' }, status: { completed: '已完成' } } } },
      })],
      stubs: { AppLayout: AppLayoutStub },
    },
  })
}

describe('TransferManageView', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    vi.clearAllMocks()
    api.listTransfers.mockResolvedValue({
      items: [{
        id: 7,
        sender_id: 1,
        receiver_id: 2,
        sender_display: 'Alice',
        receiver_display: 'Bob',
        amount: 12.5,
        fee: 0.5,
        fee_rate: 0.04,
        gross_amount: 13,
        transfer_type: 'direct',
        status: 'completed',
        memo: null,
        redpacket_id: null,
        created_at: '2026-07-25T00:00:00Z',
      }],
      total: 1,
      page: 1,
      page_size: 20,
    })
    api.getFeeStats.mockResolvedValue([{ date: '2026-07-25', total_fee: 0.5, count: 1 }])
    api.freezeTransfer.mockResolvedValue(undefined)
    api.revokeTransfer.mockResolvedValue(undefined)
    api.batchDistribute.mockResolvedValue({ items: [], count: 1 })
  })

  it('preserves administrator transfer loading and operations', async () => {
    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(true)
    const promptSpy = vi.spyOn(window, 'prompt').mockReturnValue('manual review')
    const wrapper = mountView()
    await flushPromises()

    expect(api.listTransfers).toHaveBeenCalledWith({ page: 1, page_size: 20 })
    expect(api.getFeeStats).toHaveBeenCalledWith({})
    expect(wrapper.text()).toContain('Alice')
    expect(wrapper.text()).toContain('Bob')
    expect(wrapper.text()).toContain('0.5000')

    await wrapper.get('button.text-yellow-600').trigger('click')
    await flushPromises()
    expect(confirmSpy).toHaveBeenCalledWith('确认冻结此转账？')
    expect(api.freezeTransfer).toHaveBeenCalledWith(7)

    await wrapper.get('button.text-red-600').trigger('click')
    await flushPromises()
    expect(promptSpy).toHaveBeenCalledWith('请输入撤回原因:')
    expect(api.revokeTransfer).toHaveBeenCalledWith(7, 'manual review')
  })

  it('preserves batch distribution input filtering', async () => {
    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('button.btn-primary').trigger('click')

    const numberInputs = wrapper.findAll('input[type="number"]')
    await numberInputs[0].setValue(9)
    await numberInputs[1].setValue(25)
    await wrapper.find('input[type="text"]').setValue('monthly reward')
    await wrapper.findAll('button').find((button) => button.text() === '确认发放')?.trigger('click')
    await flushPromises()

    expect(api.batchDistribute).toHaveBeenCalledWith([{ user_id: 9, amount: 25 }], 'monthly reward')
  })
})
