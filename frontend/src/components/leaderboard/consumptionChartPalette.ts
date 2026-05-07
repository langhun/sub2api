const GOLDEN_ANGLE = 137.508
const BASE_SATURATION = 72
const BASE_LIGHTNESS = 50

export function createConsumptionLeaderboardPalette(count: number): string[] {
  if (count <= 0) {
    return []
  }

  return Array.from({ length: count }, (_, index) => {
    const hue = Math.round((index * GOLDEN_ANGLE) % 360)
    const saturation = Math.max(58, BASE_SATURATION - (index % 3) * 6)
    const lightness = Math.min(64, BASE_LIGHTNESS + (index % 2) * 8)
    return `hsl(${hue} ${saturation}% ${lightness}%)`
  })
}
