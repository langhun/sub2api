import { ref, watch, computed } from 'vue'

export type ThemeMode = 'light' | 'dark' | 'auto'
export type ThemeStyle = 'default' | 'apple'

const THEME_MODE_KEY = 'theme-mode'
const THEME_STYLE_KEY = 'theme-style'

// Get system preference
function getSystemTheme(): 'light' | 'dark' {
  if (typeof window === 'undefined') return 'light'
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

// Initialize from localStorage or defaults
const themeMode = ref<ThemeMode>(
  (localStorage.getItem(THEME_MODE_KEY) as ThemeMode) || 'auto'
)

const themeStyle = ref<ThemeStyle>(
  (localStorage.getItem(THEME_STYLE_KEY) as ThemeStyle) || 'default'
)

// Computed effective theme (resolves 'auto' to actual mode)
const effectiveMode = computed(() => {
  if (themeMode.value === 'auto') {
    return getSystemTheme()
  }
  return themeMode.value
})

function applyTheme() {
  const html = document.documentElement
  const mode = effectiveMode.value

  console.log('[useTheme] Applying theme:', {
    themeMode: themeMode.value,
    themeStyle: themeStyle.value,
    effectiveMode: mode,
  })

  // Apply dark class
  if (mode === 'dark') {
    html.classList.add('dark')
  } else {
    html.classList.remove('dark')
  }

  // Apply theme style
  html.setAttribute('data-theme', themeStyle.value)
  if (themeStyle.value === 'apple') {
    html.dataset.glass = 'true'
  } else {
    delete html.dataset.glass
  }

  console.log('[useTheme] DOM updated:', {
    'data-theme': html.getAttribute('data-theme'),
    'dark class': html.classList.contains('dark'),
  })

  // Save to localStorage
  localStorage.setItem(THEME_MODE_KEY, themeMode.value)
  localStorage.setItem(THEME_STYLE_KEY, themeStyle.value)
}

// Watch for changes
watch([themeMode, themeStyle], applyTheme, { immediate: true })

// Listen to system theme changes when in auto mode
if (typeof window !== 'undefined') {
  const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
  mediaQuery.addEventListener('change', () => {
    if (themeMode.value === 'auto') {
      applyTheme()
    }
  })
}

export function useTheme() {
  return {
    themeMode,
    themeStyle,
    effectiveMode,
    setThemeMode: (mode: ThemeMode) => {
      themeMode.value = mode
    },
    setThemeStyle: (style: ThemeStyle) => {
      themeStyle.value = style
    },
    toggleMode: () => {
      const current = effectiveMode.value
      themeMode.value = current === 'dark' ? 'light' : 'dark'
    },
  }
}
