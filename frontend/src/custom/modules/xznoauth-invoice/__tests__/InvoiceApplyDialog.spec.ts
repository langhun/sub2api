import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'

const invoiceAPI = vi.hoisted(() => ({
  validateOrders: vi.fn(),
  createInvoice: vi.fn(),
  getTaxPayments: vi.fn(),
  checkTaxPayment: vi.fn(),
}))
const appStore = vi.hoisted(() => ({ showSuccess: vi.fn(), showError: vi.fn() }))

vi.mock('../api', () => ({ invoiceAPI }))
vi.mock('@/stores', () => ({ useAppStore: () => appStore }))
vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => ({
      'common.next': 'Next',
      'invoices.titlePlaceholder': 'Enter invoice title or name',
      'invoices.submit': 'Save application',
      'invoices.errors.incompleteValidation': 'The invoice service did not verify every selected order.',
    })[key] || key,
  }),
}))

import InvoiceApplyDialog from '../InvoiceApplyDialog.vue'

const DialogStub = defineComponent({
  props: { show: Boolean },
  template: '<section v-if="show"><slot /></section>',
})

function mountDialog(orderNo = 'ORDER-1') {
  return mount(InvoiceApplyDialog, {
    props: { show: true, orders: [{ out_trade_no: orderNo }] as never },
    global: {
      stubs: { BaseDialog: DialogStub },
    },
  })
}

function findButton(wrapper: ReturnType<typeof mountDialog>, label: string) {
  return wrapper.findAll('button').find(button => button.text().trim() === label)
}

describe('InvoiceApplyDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    invoiceAPI.validateOrders.mockImplementation(async ({ orderNos }: { orderNos: string[] }) => ({
      data: {
        orders: orderNos.map(orderNo => ({ orderNo, amount: '10.00', currency: 'CNY' })),
        totalAmount: '10.00', currency: 'CNY', taxAmount: '0.60', taxPaidAmount: '0.00', taxDueAmount: '0.60',
      },
    }))
    invoiceAPI.createInvoice.mockResolvedValue({ data: { id: 42, status: 'pending' } })
  })

  it('saves the exact order snapshot that was validated', async () => {
    const wrapper = mountDialog()
    await flushPromises()
    expect(invoiceAPI.validateOrders).toHaveBeenCalledWith({ orderNos: ['ORDER-1'], needPayTax: false })

    await findButton(wrapper, 'Next')?.trigger('click')
    await wrapper.find('input[placeholder="Enter invoice title or name"]').setValue('Example Company')
    await wrapper.find('input[type="email"]').setValue('billing@example.com')
    await wrapper.findAll('input').filter(input => input.attributes('type') !== 'radio' && input.attributes('type') !== 'email')[1].setValue('91310000')
    await findButton(wrapper, 'Save application')?.trigger('click')
    await flushPromises()

    expect(invoiceAPI.createInvoice).toHaveBeenCalledWith(expect.objectContaining({
      orderNos: ['ORDER-1'],
      needPayTax: false,
      title: 'Example Company',
      taxpayerId: '91310000',
      recipientEmail: 'billing@example.com',
    }))
    expect(wrapper.emitted('saved')).toHaveLength(1)
  })

  it('invalidates the previous validation when the selected order changes', async () => {
    const wrapper = mountDialog()
    await flushPromises()
    await wrapper.setProps({ orders: [{ out_trade_no: 'ORDER-2' }] as never })
    await flushPromises()

    expect(invoiceAPI.validateOrders).toHaveBeenLastCalledWith({ orderNos: ['ORDER-2'], needPayTax: false })
    await findButton(wrapper, 'Next')?.trigger('click')
    await wrapper.find('input[placeholder="Enter invoice title or name"]').setValue('Example Company')
    await wrapper.find('input[type="email"]').setValue('billing@example.com')
    await wrapper.findAll('input').filter(input => input.attributes('type') !== 'radio' && input.attributes('type') !== 'email')[1].setValue('91310000')
    await findButton(wrapper, 'Save application')?.trigger('click')
    await flushPromises()

    expect(invoiceAPI.createInvoice).toHaveBeenCalledWith(expect.objectContaining({ orderNos: ['ORDER-2'] }))
  })

  it('does not advance when the remote validation omits a selected order', async () => {
    invoiceAPI.validateOrders.mockResolvedValueOnce({
      data: { orders: [{ orderNo: 'OTHER', amount: '10.00', currency: 'CNY' }], totalAmount: '10.00', currency: 'CNY' },
    })
    const wrapper = mountDialog()
    await flushPromises()

    expect(findButton(wrapper, 'Next')?.attributes('disabled')).toBeDefined()
    expect(wrapper.text()).toContain('did not verify every selected order')
  })
})
