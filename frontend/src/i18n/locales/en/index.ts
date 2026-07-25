import landing from './landing'
import common from './common'
import dashboard from './dashboard'
import batchImage from './batchImage'
import admin from './admin'
import misc from './misc'
import balanceFeatures from './balanceFeatures'
import { activityLocaleMessages } from '@/custom/modules/activity/locales'
import { gameHallLocaleMessages } from '@/custom/modules/game-hall/locales'
import { walletExtensionLocaleMessages } from '@/custom/modules/wallet-extension/locales'

export default {
  ...landing,
  ...common,
  ...dashboard,
  ...batchImage,
  admin,
  ...misc,
  ...balanceFeatures,
  ...activityLocaleMessages.en,
  ...gameHallLocaleMessages.en,
  ...walletExtensionLocaleMessages.en,
}
