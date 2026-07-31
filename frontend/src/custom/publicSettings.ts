import { activityPublicSettingsDefaults } from './modules/activity/publicSettings'
import { gameHallPublicSettingsDefaults } from './modules/game-hall/publicSettings'

/** Pure compatibility projection consumed by the shared application store. */
export const customPublicSettingsDefaults: Readonly<Record<string, boolean>> = {
  ...activityPublicSettingsDefaults,
  ...gameHallPublicSettingsDefaults,
}
