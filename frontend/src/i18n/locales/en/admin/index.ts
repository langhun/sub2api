import overview from './overview'
import channels from './channels'
import accounts from './accounts'
import resources from './resources'
import ops from './ops'
import settings from './settings'
import balanceFeatures from './balanceFeatures'

const base = {
  ...overview,
  ...channels,
  ...accounts,
  ...resources,
  ...ops,
  ...settings,
}

export default {
  ...base,
  blindbox: balanceFeatures.blindbox,
  codeFormat: balanceFeatures.codeFormat,
  settings: {
    ...base.settings,
    ...balanceFeatures.settings,
    tabs: { ...base.settings.tabs, ...balanceFeatures.settings.tabs },
  },
}
