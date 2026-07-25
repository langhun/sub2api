import CheckinHeaderActions from './components/CheckinHeaderActions.vue'
import type { CustomHeaderAction } from '../../registry'

export const activityHeaderActions: readonly CustomHeaderAction[] = [
  {
    id: 'activity-checkin',
    component: CheckinHeaderActions,
  },
]
