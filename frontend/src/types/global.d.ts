import type { PublicSettings } from '@/types'

declare global {
  interface ErrorPageController {
    resetEasterEggHighScore(): void
    trackEasterEgg(): void
    updateEasterEggHighScore(score: number): void
  }

  interface Window {
    __APP_CONFIG__?: PublicSettings
    errorPageController?: ErrorPageController
    initializeEasterEggHighScore?: (highScore: number) => void
  }
}

export {}
