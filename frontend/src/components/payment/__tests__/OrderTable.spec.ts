import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import type { PaymentOrder } from '@/types/payment'
import OrderTable from '../OrderTable.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}))

const DataTableStub = {
  props: ['columns', 'data', 'loading'],
  template: '<div />',
}

const OrderStatusBadgeStub = { template: '<span />' }

function orderFactory(): PaymentOrder {
  return {
    id: 42,
    user_id: 7,
    amount: 10,
    pay_amount: 10,
    currency: 'USD',
    fee_rate: 0,
    payment_type: 'stripe',
    out_trade_no: 'order-public-number',
    status: 'COMPLETED',
    order_type: 'balance',
    created_at: '2026-08-07T00:00:00Z',
    expires_at: '2026-08-07T00:30:00Z',
    refund_amount: 0,
  }
}

describe('OrderTable order ID visibility', () => {
  it('hides the internal ID column when requested by the user order view', () => {
    const wrapper = mount(OrderTable, {
      props: { orders: [orderFactory()], loading: false, showOrderId: false },
      global: { stubs: { DataTable: DataTableStub, OrderStatusBadge: OrderStatusBadgeStub } },
    })

    const columns = wrapper.findComponent(DataTableStub).props('columns') as Array<{ key: string }>
    expect(columns.map(column => column.key)).not.toContain('id')
    expect(columns.map(column => column.key)).toContain('out_trade_no')
  })

  it('keeps the ID column by default for existing callers', () => {
    const wrapper = mount(OrderTable, {
      props: { orders: [orderFactory()], loading: false },
      global: { stubs: { DataTable: DataTableStub, OrderStatusBadge: OrderStatusBadgeStub } },
    })

    const columns = wrapper.findComponent(DataTableStub).props('columns') as Array<{ key: string }>
    expect(columns.map(column => column.key)).toContain('id')
  })
})
