import overview from './overview'
import channels from './channels'
import accounts from './accounts'
import resources from './resources'
import ops from './ops'
import settings from './settings'
import audit from './audit'
import promptAudit from './promptAudit'
import registrationRisk from './registrationRisk'
import { mergeCustomAdminLocale } from '@/custom/locales'

const base = {
  ...overview,
  ...channels,
  ...accounts,
  ...resources,
  ...ops,
  ...settings,
  ...audit,
  ...promptAudit,
  ...registrationRisk,
}

export default mergeCustomAdminLocale(base, 'zh')
