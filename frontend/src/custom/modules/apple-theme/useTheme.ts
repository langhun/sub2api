import { computed, ref, watch } from 'vue'

export type ThemeMode = 'light' | 'dark' | 'auto'
export type ThemeStyle = 'default' | 'apple'

type PublicThemeSettings = {
  enable_apple_theme?: boolean
  default_theme_mode?: ThemeMode
  theme_style?: 'glass' | 'classic'
}

const THEME_MODE_KEY = 'theme'

function getSystemTheme(): 'light' | 'dark' {
  if (typeof window === 'undefined') return 'light'
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

function readStoredMode(): ThemeMode | null {
  if (typeof window === 'undefined') return null
  const stored = window.localStorage.getItem(THEME_MODE_KEY)
  return stored === 'light' || stored === 'dark' || stored === 'auto' ? stored : null
}

function readDefaultMode(settings?: PublicThemeSettings): ThemeMode {
  const mode = settings?.default_theme_mode
  return mode === 'light' || mode === 'dark' || mode === 'auto' ? mode : 'auto'
}

function readThemeStyle(settings?: PublicThemeSettings): ThemeStyle {
  if (typeof settings?.enable_apple_theme === 'boolean') {
    return settings.enable_apple_theme ? 'apple' : 'default'
  }
  return settings?.theme_style === 'classic' ? 'default' : 'apple'
}

const injectedSettings = typeof window === 'undefined' ? undefined : window.__APP_CONFIG__
let defaultThemeMode = readDefaultMode(injectedSettings)
const themeMode = ref<ThemeMode>(readStoredMode() ?? defaultThemeMode)
const themeStyle = ref<ThemeStyle>(readThemeStyle(injectedSettings))

const effectiveMode = computed<'light' | 'dark'>(() => (
  themeMode.value === 'auto' ? getSystemTheme() : themeMode.value
))

function applyTheme(): void {
  if (typeof document === 'undefined') return
  const html = document.documentElement
  html.classList.toggle('dark', effectiveMode.value === 'dark')
  html.setAttribute('data-theme', themeStyle.value)
  if (themeStyle.value === 'apple') html.setAttribute('data-glass', 'true')
  else html.removeAttribute('data-glass')
}

watch([themeMode, themeStyle], applyTheme, { immediate: true })

if (typeof window !== 'undefined') {
  const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
  const onMediaChange = () => {
    if (themeMode.value === 'auto') applyTheme()
  }
  if (typeof mediaQuery.addEventListener === 'function') mediaQuery.addEventListener('change', onMediaChange)
  else mediaQuery.addListener?.(onMediaChange)
}

export function applyGlobalThemeSettings(settings: PublicThemeSettings): void {
  defaultThemeMode = readDefaultMode(settings)
  if (readStoredMode() === null) themeMode.value = defaultThemeMode
  themeStyle.value = readThemeStyle(settings)
  applyTheme()
}

export function useTheme() {
  return {
    themeMode,
    themeStyle,
    effectiveMode,
    setThemeMode: (mode: ThemeMode) => {
      themeMode.value = mode
      if (typeof window !== 'undefined') window.localStorage.setItem(THEME_MODE_KEY, mode)
    },
    toggleMode: () => {
      const mode = effectiveMode.value === 'dark' ? 'light' : 'dark'
      themeMode.value = mode
      if (typeof window !== 'undefined') window.localStorage.setItem(THEME_MODE_KEY, mode)
    },
  }
}
