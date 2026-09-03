export type ThemeStyle = 'glass' | 'classic'

const THEME_STYLE_KEY = 'theme_style'

export function readThemeStyle(): ThemeStyle {
  try {
    return localStorage.getItem(THEME_STYLE_KEY) === 'classic' ? 'classic' : 'glass'
  } catch {
    return 'glass'
  }
}

export function applyThemeStyle(style: ThemeStyle): void {
  if (style === 'glass') document.documentElement.dataset.glass = 'true'
  else document.documentElement.removeAttribute('data-glass')
  try {
    localStorage.setItem(THEME_STYLE_KEY, style)
  } catch {
    // Ignore storage failures; the current session still follows the choice.
  }
  window.dispatchEvent(new CustomEvent('theme-style-change', { detail: style }))
}
