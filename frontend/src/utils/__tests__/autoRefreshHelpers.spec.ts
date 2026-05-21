import { describe, it, expect } from 'vitest'
import {
  shouldSkipAutoRefresh,
  calculateSilentWindowCountdown,
  type AutoRefreshSkipConditions
} from '../autoRefreshHelpers'

describe('autoRefreshHelpers', () => {
  describe('shouldSkipAutoRefresh', () => {
    const baseConditions: AutoRefreshSkipConditions = {
      autoRefreshEnabled: true,
      documentHidden: false,
      loading: false,
      autoRefreshFetching: false,
      isAnyModalOpen: false,
      menuShow: false,
      showAccountToolsDropdown: false,
      showAutoRefreshDropdown: false,
      inSilentWindow: false
    }

    it('should return false when all conditions allow refresh', () => {
      expect(shouldSkipAutoRefresh(baseConditions)).toBe(false)
    })

    it('should skip when autoRefreshEnabled is false', () => {
      expect(shouldSkipAutoRefresh({
        ...baseConditions,
        autoRefreshEnabled: false
      })).toBe(true)
    })

    it('should skip when document is hidden', () => {
      expect(shouldSkipAutoRefresh({
        ...baseConditions,
        documentHidden: true
      })).toBe(true)
    })

    it('should skip when loading', () => {
      expect(shouldSkipAutoRefresh({
        ...baseConditions,
        loading: true
      })).toBe(true)
    })

    it('should skip when autoRefreshFetching', () => {
      expect(shouldSkipAutoRefresh({
        ...baseConditions,
        autoRefreshFetching: true
      })).toBe(true)
    })

    it('should skip when any modal is open', () => {
      expect(shouldSkipAutoRefresh({
        ...baseConditions,
        isAnyModalOpen: true
      })).toBe(true)
    })

    it('should skip when menu is shown', () => {
      expect(shouldSkipAutoRefresh({
        ...baseConditions,
        menuShow: true
      })).toBe(true)
    })

    it('should skip when account tools dropdown is shown', () => {
      expect(shouldSkipAutoRefresh({
        ...baseConditions,
        showAccountToolsDropdown: true
      })).toBe(true)
    })

    it('should skip when auto refresh dropdown is shown', () => {
      expect(shouldSkipAutoRefresh({
        ...baseConditions,
        showAutoRefreshDropdown: true
      })).toBe(true)
    })

    it('should skip when in silent window', () => {
      expect(shouldSkipAutoRefresh({
        ...baseConditions,
        inSilentWindow: true
      })).toBe(true)
    })

    it('should skip when multiple conditions are true', () => {
      expect(shouldSkipAutoRefresh({
        ...baseConditions,
        loading: true,
        isAnyModalOpen: true,
        menuShow: true
      })).toBe(true)
    })
  })

  describe('calculateSilentWindowCountdown', () => {
    it('should return positive countdown when silentUntil is in the future', () => {
      const now = 1000000
      const silentUntil = 1005000 // 5 seconds in the future
      expect(calculateSilentWindowCountdown(silentUntil, now)).toBe(5)
    })

    it('should return 0 when silentUntil is in the past', () => {
      const now = 1000000
      const silentUntil = 995000 // 5 seconds in the past
      expect(calculateSilentWindowCountdown(silentUntil, now)).toBe(0)
    })

    it('should return 0 when silentUntil equals now', () => {
      const now = 1000000
      const silentUntil = 1000000
      expect(calculateSilentWindowCountdown(silentUntil, now)).toBe(0)
    })

    it('should round up partial seconds', () => {
      const now = 1000000
      const silentUntil = 1000500 // 0.5 seconds in the future
      expect(calculateSilentWindowCountdown(silentUntil, now)).toBe(1)
    })

    it('should handle large time differences', () => {
      const now = 1000000
      const silentUntil = 1060000 // 60 seconds in the future
      expect(calculateSilentWindowCountdown(silentUntil, now)).toBe(60)
    })
  })
})

