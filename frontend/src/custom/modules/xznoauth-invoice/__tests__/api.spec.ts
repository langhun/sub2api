import { beforeEach, describe, expect, it, vi } from 'vitest'

const apiClient = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }))

vi.mock('@/api/client', () => ({ apiClient }))

import { invoiceAPI } from '../api'

describe('XZNOAuth invoice facade API contract', () => {
  beforeEach(() => vi.clearAllMocks())

  it('validates without creating tax payment orders', async () => {
    apiClient.post.mockResolvedValue({ data: {} })

    await invoiceAPI.validateOrders({ orderNos: ['ORDER-1'], needPayTax: true })

    expect(apiClient.post).toHaveBeenCalledWith('/custom/invoices/validate', {
      orderNos: ['ORDER-1'],
      needPayTax: true,
      createTaxPayments: false,
    })
  })

  it('sends only the selected tax order number for payment status', async () => {
    apiClient.post.mockResolvedValue({ data: { paid: false } })

    await invoiceAPI.checkTaxPayment({ taxOrderNo: 'INV-TAX-1' })

    expect(apiClient.post).toHaveBeenCalledWith('/custom/invoices/tax-payments/status', { taxOrderNo: 'INV-TAX-1' })
  })

  it('requests and cancels a saved application without a request body', async () => {
    apiClient.post.mockResolvedValue({ data: {} })

    await invoiceAPI.getTaxPayments(42)
    await invoiceAPI.cancelInvoice(42)

    expect(apiClient.post).toHaveBeenNthCalledWith(1, '/custom/invoices/42/tax-payments')
    expect(apiClient.post).toHaveBeenNthCalledWith(2, '/custom/invoices/42/cancel')
  })

  it('uses the local facade page_size parameter', async () => {
    apiClient.get.mockResolvedValue({ data: {} })

    await invoiceAPI.listInvoices({ page: 2, pageSize: 50 })

    expect(apiClient.get).toHaveBeenCalledWith('/custom/invoices', { params: { page: 2, page_size: 50 } })
  })

  it('downloads completed invoices as binary data', async () => {
    apiClient.get.mockResolvedValue({ data: new Blob(['%PDF-1.7']) })

    await invoiceAPI.downloadInvoicePDF(42)

    expect(apiClient.get).toHaveBeenCalledWith('/custom/invoices/42/pdf', { responseType: 'blob' })
  })
})
