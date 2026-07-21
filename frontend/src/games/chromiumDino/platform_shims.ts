export function assert(condition: unknown, message = 'Chromium Dino assertion failed'): asserts condition {
  if (!condition) throw new Error(message)
}

export const HIDDEN_CLASS = 'hidden'

const strings: Record<string, string> = {
  dinoGameA11yAriaLabel: 'Dino game',
  dinoGameA11yDescription: 'Press space to play',
  dinoGameA11yGameOver: 'Game over',
  dinoGameA11yHighScore: 'High score $1',
  dinoGameA11yJump: 'Jump',
  dinoGameA11ySpeedToggle: 'Slow speed',
  dinoGameA11yStartGame: 'Game started',
}

export const loadTimeData = {
  getString: (key: string) => strings[key] ?? '',
  getValue: (key: string) => strings[key] ?? '',
  valueExists: (_key: string) => false,
}
