export type ThemeStyle = 'glass' | 'classic'

export function readThemeStyle(): ThemeStyle {
  return typeof window !== 'undefined' && window.__APP_CONFIG__?.theme_style === 'classic'
    ? 'classic'
    : 'glass'
}

export function applyThemeStyle(style: ThemeStyle): void {
  if (style === 'glass') document.documentElement.dataset.glass = 'true'
  else document.documentElement.removeAttribute('data-glass')
}
