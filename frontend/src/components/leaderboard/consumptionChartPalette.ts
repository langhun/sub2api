const BASE_CHART_COLORS = [
  '#3b82f6',
  '#10b981',
  '#f59e0b',
  '#ef4444',
  '#8b5cf6',
  '#ec4899',
  '#14b8a6',
  '#f97316',
  '#6366f1',
  '#84cc16',
  '#06b6d4',
  '#a855f7',
]

const EXTRA_HUES = [205, 160, 32, 350, 272, 326, 188, 22, 243, 88, 195, 279]

export function createConsumptionLeaderboardPalette(count: number): string[] {
  if (count <= 0) {
    return []
  }

  if (count <= BASE_CHART_COLORS.length) {
    return BASE_CHART_COLORS.slice(0, count)
  }

  const colors = [...BASE_CHART_COLORS]
  for (let index = BASE_CHART_COLORS.length; index < count; index += 1) {
    const hue = EXTRA_HUES[index % EXTRA_HUES.length]
    const saturation = index % 2 === 0 ? 72 : 66
    const lightness = index % 3 === 0 ? 56 : 50
    colors.push(`hsl(${hue} ${saturation}% ${lightness}%)`)
  }
  return colors
}
