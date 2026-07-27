declare global {
  interface ErrorPageController {
    resetEasterEggHighScore(): void
    trackEasterEgg(): void
    updateEasterEggHighScore(score: number): void
  }

  interface Window {
    errorPageController?: ErrorPageController
    initializeEasterEggHighScore?: (highScore: number) => void
  }
}

export {}
