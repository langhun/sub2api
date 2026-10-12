import { apiClient } from '@/api/client'

export interface InvoiceOrder {
  orderNo: string
  externalNo?: string
  transactionId?: string
  productName?: string
  amount: string
  currency: string
  paidAt?: string
}

export interface InvoiceValidation {
  orders: InvoiceOrder[]
  totalAmount: string
  currency: string
  taxAmount?: string
  taxPaidAmount?: string
  taxDueAmount?: string
}

export interface InvoiceApplication {
  id: number
  status: 'awaiting_tax_payment' | 'pending' | 'approved' | 'rejected' | 'completed' | 'canceled' | string
  orderNos?: string[]
  buyerType?: string
  title?: string
  taxpayerId?: string
  recipientEmail?: string
  totalAmount?: string
  currency?: string
  taxAmount?: string
  taxPaidAmount?: string
  taxDueAmount?: string
  reviewNote?: string
  hasPdf?: boolean
  createdAt?: string
  updatedAt?: string
}

export interface TaxPayment {
  taxOrderNo: string
  payUrl: string
}

export interface TaxCheckout {
  application: InvoiceApplication
  taxPaidAmount: string
  taxDueAmount: string
  taxPayments?: Record<string, TaxPayment>
}

export interface InvoicePage {
  items: InvoiceApplication[]
  total: number
  page: number
  page_size: number
  pages: number
}

export interface InvoiceApplicationInput {
  orderNos: string[]
  needPayTax: boolean
  buyerType: 'individual' | 'company'
  title: string
  taxpayerId: string
  buyerAddress?: string
  buyerPhone?: string
  buyerBank?: string
  buyerBankAccount?: string
  recipientEmail: string
}

export const invoiceAPI = {
  validateOrders(data: { orderNos: string[]; needPayTax: boolean }) {
    return apiClient.post<InvoiceValidation>('/custom/invoices/validate', {
      ...data,
      createTaxPayments: false,
    })
  },
  createInvoice(data: InvoiceApplicationInput) {
    return apiClient.post<InvoiceApplication>('/custom/invoices', data)
  },
  getTaxPayments(id: number) {
    return apiClient.post<TaxCheckout>(`/custom/invoices/${id}/tax-payments`)
  },
  checkTaxPayment(input: { taxOrderNo: string }) {
    return apiClient.post<{ paid: boolean }>('/custom/invoices/tax-payments/status', input)
  },
  listInvoices(params: { page: number; pageSize: number }) {
    return apiClient.get<InvoicePage>('/custom/invoices', {
      params: { page: params.page, page_size: params.pageSize },
    })
  },
  cancelInvoice(id: number) {
    return apiClient.post<InvoiceApplication>(`/custom/invoices/${id}/cancel`)
  },
  downloadInvoicePDF(id: number) {
    return apiClient.get<Blob>(`/custom/invoices/${id}/pdf`, { responseType: 'blob' })
  },
}
