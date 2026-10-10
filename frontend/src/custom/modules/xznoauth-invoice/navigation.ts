import { h } from 'vue'
import Icon from '@/components/icons/Icon.vue'
import type { CustomNavigationItem } from '../../registry'

const InvoiceIcon = { render: () => h(Icon, { name: 'document' }) }

export const xznoauthInvoiceNavigation: readonly CustomNavigationItem[] = [
  {
    path: '/invoices',
    label: 'Invoice records',
    labelKey: 'invoices.recordsTitle',
    icon: InvoiceIcon,
    slot: 'after-affiliate',
  },
]
