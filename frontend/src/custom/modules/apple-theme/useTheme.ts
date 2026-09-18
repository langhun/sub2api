import { computed, ref, watch } from 'vue'

export type ThemeMode = 'light' | 'dark' | 'auto' | 'time'
export type ThemeStyle = 'default' | 'apple'

type PublicThemeSettings = {
  enable_apple_theme?: boolean
  default_theme_mode?: ThemeMode
  theme_style?: 'glass' | 'classic'
}

const THEME_MODE_KEY = 'theme'
const DAY_START_HOUR = 7
const NIGHT_START_HOUR = 19

function getSystemTheme(): 'light' | 'dark' {
  if (typeof window === 'undefined') return 'light'
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

export function getTimeBasedTheme(date: Date): 'light' | 'dark' {
  const hour = date.getHours()
  return hour >= DAY_START_HOUR && hour < NIGHT_START_HOUR ? 'light' : 'dark'
}

function readStoredMode(): ThemeMode | null {
  if (typeof window === 'undefined') return null
  const stored = window.localStorage.getItem(THEME_MODE_KEY)
  return stored === 'light' || stored === 'dark' || stored === 'auto' || stored === 'time' ? stored : null
}

function readDefaultMode(settings?: PublicThemeSettings): ThemeMode {
  const mode = settings?.default_theme_mode
  return mode === 'light' || mode === 'dark' || mode === 'auto' || mode === 'time' ? mode : 'auto'
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
const systemTheme = ref<'light' | 'dark'>(getSystemTheme())
const browserTime = ref(Date.now())
let timeRefreshTimer: number | undefined

const effectiveMode = computed<'light' | 'dark'>(() => {
  if (themeMode.value === 'auto') return systemTheme.value
  if (themeMode.value === 'time') return getTimeBasedTheme(new Date(browserTime.value))
  return themeMode.value
})

function applyTheme(): void {
  if (typeof document === 'undefined') return
  const html = document.documentElement
  html.classList.toggle('dark', effectiveMode.value === 'dark')
  html.setAttribute('data-theme', themeStyle.value)
  if (themeStyle.value === 'apple') html.setAttribute('data-glass', 'true')
  else html.removeAttribute('data-glass')
}

function refreshBrowserTime(): void {
  browserTime.value = Date.now()
}

watch([themeMode, themeStyle, effectiveMode], applyTheme, { immediate: true })

if (typeof window !== 'undefined') {
  const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
  const onMediaChange = (event: MediaQueryListEvent) => {
    systemTheme.value = event.matches ? 'dark' : 'light'
  }
  if (typeof mediaQuery.addEventListener === 'function') mediaQuery.addEventListener('change', onMediaChange)
  else mediaQuery.addListener?.(onMediaChange)

  const scheduleTimeRefresh = () => {
    if (timeRefreshTimer !== undefined) window.clearTimeout(timeRefreshTimer)

    const now = new Date()
    const nextChange = new Date(now)
    nextChange.setHours(getTimeBasedTheme(now) === 'light' ? NIGHT_START_HOUR : DAY_START_HOUR, 0, 0, 0)
    if (nextChange <= now) nextChange.setDate(nextChange.getDate() + 1)

    timeRefreshTimer = window.setTimeout(() => {
      refreshBrowserTime()
      applyTheme()
      scheduleTimeRefresh()
    }, nextChange.getTime() - now.getTime() + 25)
  }

  document.addEventListener('visibilitychange', () => {
    if (!document.hidden) {
      refreshBrowserTime()
      applyTheme()
      scheduleTimeRefresh()
    }
  })
  scheduleTimeRefresh()
}

export function applyGlobalThemeSettings(settings: PublicThemeSettings): void {
  defaultThemeMode = readDefaultMode(settings)
  refreshBrowserTime()
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
      if (mode === 'time') refreshBrowserTime()
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
