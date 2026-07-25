import landing from './landing'
import common from './common'
import dashboard from './dashboard'
import batchImage from './batchImage'
import admin from './admin'
import misc from './misc'
import balanceFeatures from './balanceFeatures'
import { gameHallLocaleMessages } from '@/custom/modules/game-hall/locales'

export default {
  ...landing,
  ...common,
  ...dashboard,
  ...batchImage,
  admin,
  ...misc,
  ...balanceFeatures,
  ...gameHallLocaleMessages.zh,
}
