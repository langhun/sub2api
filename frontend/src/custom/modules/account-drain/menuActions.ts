import AccountDrainMenuAction from './components/AccountDrainMenuAction.vue'
import type { CustomAccountMenuAction } from '../../registry'

export const accountDrainMenuActions: readonly CustomAccountMenuAction[] = [
  {
    id: 'account-drain-target',
    component: AccountDrainMenuAction,
  },
]
