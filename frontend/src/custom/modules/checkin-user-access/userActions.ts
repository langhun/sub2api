import type { AdminUser } from '@/types'
import type { CustomAdminUserAction } from '@/custom/registry'

export const checkinUserAccessActions: readonly CustomAdminUserAction[] = [
  {
    id: 'checkin-user-access',
    labelKey: 'admin.checkinUserAccess.menuAction',
    icon: 'shield',
    visible: (_user: AdminUser) => true,
  },
]
