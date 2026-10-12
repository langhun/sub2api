import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import UserOrdersView from '../UserOrdersView.vue'
import Select from '@/components/common/Select.vue'
import Pagination from '@/components/common/Pagination.vue'
import OrderTable from '@/components/payment/OrderTable.vue'
import type { PaymentOrder } from '@/types/payment'

const api = vi.hoisted(() => ({ getMyOrders: vi.fn(), getRefundEligibleProviders: vi.fn() }))
const invoice = vi.hoisted(() => ({ listEligibleOrderNos: vi.fn() }))
vi.mock('@/api/payment', () => ({ paymentAPI: api }))
vi.mock('@/custom/modules/xznoauth-invoice/api', () => ({ invoiceAPI: invoice }))
vi.mock('@/stores', () => ({ useAppStore: () => ({ showError: vi.fn() }) }))
vi.mock('vue-router', () => ({ useRouter: () => ({ push: vi.fn() }) }))
vi.mock('vue-i18n', async (importOriginal) => ({
  ...await importOriginal<typeof import('vue-i18n')>(),
  useI18n: () => ({ t: (key: string) => key })
}))
enableAutoUnmount(afterEach)

beforeEach(() => {
  vi.clearAllMocks()
  api.getMyOrders.mockResolvedValue({ data: { items: [], total: 100, pages: 5 } })
  api.getRefundEligibleProviders.mockResolvedValue({ data: { provider_instance_ids: [] } })
  invoice.listEligibleOrderNos.mockResolvedValue({ data: { orderNos: [] } })
})

async function openOrders() {
  const wrapper = mount(UserOrdersView, {
    global: { stubs: {
      AppLayout: { template: '<div><slot /></div>' },
      OrderTable: true, BaseDialog: true, Icon: true, Pagination: true, teleport: true
    } }
  })
  await flushPromises()
  wrapper.getComponent(Pagination).vm.$emit('update:page', 4)
  await flushPromises()
  return wrapper
}

describe('order status filtering', () => {
  it('uses the invoice action to show completed orders when nothing is selected', async () => {
    const wrapper = await openOrders()
    const invoiceButton = wrapper.findAll('button').find(button => button.text() === 'invoices.apply')

    expect(invoiceButton).toBeDefined()
    await invoiceButton!.trigger('click')
    await flushPromises()

    expect(api.getMyOrders).toHaveBeenLastCalledWith({ page: 1, page_size: 1000, status: 'COMPLETED' })
    expect(wrapper.getComponent(Select).props('modelValue')).toBe('COMPLETED')
    expect(invoiceButton!.attributes('aria-pressed')).toBe('true')
  })

  it('shows only merchant-verified completed orders in the invoice picker', async () => {
    const orders = [
      { id: 1, out_trade_no: 'OLD-ORDER', status: 'COMPLETED' },
      { id: 2, out_trade_no: 'NEW-ORDER', status: 'COMPLETED' },
    ] as PaymentOrder[]
    api.getMyOrders.mockResolvedValue({ data: { items: orders, total: orders.length, pages: 1 } })
    invoice.listEligibleOrderNos.mockResolvedValue({ data: { orderNos: ['NEW-ORDER'] } })

    const wrapper = await openOrders()
    const invoiceButton = wrapper.findAll('button').find(button => button.text() === 'invoices.apply')
    await invoiceButton!.trigger('click')
    await flushPromises()

    expect(invoice.listEligibleOrderNos).toHaveBeenLastCalledWith(['OLD-ORDER', 'NEW-ORDER'])
    expect(wrapper.getComponent(OrderTable).props('orders')).toEqual([orders[1]])
    expect(wrapper.getComponent(OrderTable).props('selectionDisabled')?.(orders[0])).toBe(true)
    expect(wrapper.getComponent(OrderTable).props('selectionDisabled')?.(orders[1])).toBe(false)
  })

  it('loads the first page with the selected status', async () => {
    const wrapper = await openOrders()
    const select = wrapper.getComponent(Select)
    await select.get('button').trigger('click')
    await select.findAll('[role="option"]').find(option => option.text() === 'payment.status.pending')!.trigger('click')
    await flushPromises()
    expect(api.getMyOrders).toHaveBeenLastCalledWith({ page: 1, page_size: 20, status: 'PENDING' })
    expect(wrapper.getComponent(Pagination).props('page')).toBe(1)
    expect(api.getMyOrders).toHaveBeenCalledTimes(3)
  })

  it('keeps the current page on manual refresh', async () => {
    const wrapper = await openOrders()
    await wrapper.get('[title="common.refresh"]').trigger('click')
    await flushPromises()
    expect(api.getMyOrders).toHaveBeenLastCalledWith({ page: 4, page_size: 20, status: undefined })
  })
})
