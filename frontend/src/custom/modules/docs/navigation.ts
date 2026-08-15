import { h } from 'vue'
import Icon from '@/components/icons/Icon.vue'
import type { CustomNavigationItem } from '../../registry'

const HelpDocsIcon = {
  render: () => h(Icon, { name: 'book' }),
}

export const docsNavigation: readonly CustomNavigationItem[] = [
  {
    path: '/docs',
    label: 'Help documentation',
    labelKey: 'nav.helpDocs',
    icon: HelpDocsIcon,
    slot: 'after-affiliate',
  },
]
