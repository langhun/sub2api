import { activityPublicSettingsDefaults } from './modules/activity/publicSettings'
import { gameHallPublicSettingsDefaults } from './modules/game-hall/publicSettings'
import { rechargePromotionPublicSettingsDefaults } from './modules/recharge-promotions/publicSettings'

/** Pure compatibility projection consumed by the shared application store. */
export const customPublicSettingsDefaults: Readonly<Record<string, unknown>> = {
  enable_apple_theme: true,
  default_theme_mode: 'auto',
  theme_style: 'glass',
  ...activityPublicSettingsDefaults,
  ...gameHallPublicSettingsDefaults,
  ...rechargePromotionPublicSettingsDefaults,
}
