import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { applyGlobalThemeSettings, getTimeBasedTheme, useTheme } from '../useTheme'

describe('useTheme', () => {
  beforeEach(() => {
    localStorage.clear()
    document.documentElement.className = ''
    document.documentElement.removeAttribute('data-glass')
    document.documentElement.removeAttribute('data-theme')
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('uses server settings for the global theme style and ignores local overrides', () => {
    localStorage.setItem('theme-style', 'default')

    applyGlobalThemeSettings({ enable_apple_theme: true, default_theme_mode: 'auto' })
    expect(useTheme().themeStyle.value).toBe('apple')
    expect(document.documentElement.dataset.glass).toBe('true')

    applyGlobalThemeSettings({ enable_apple_theme: false, default_theme_mode: 'auto' })
    expect(useTheme().themeStyle.value).toBe('default')
    expect(document.documentElement.hasAttribute('data-glass')).toBe(false)
  })

  it('supports the legacy server theme_style without writing a style preference', () => {
    localStorage.removeItem('theme-style')

    applyGlobalThemeSettings({ theme_style: 'classic' })

    expect(useTheme().themeStyle.value).toBe('default')
    expect(localStorage.getItem('theme-style')).toBeNull()
  })

  it('uses the server default mode until the user chooses a local mode', () => {
    applyGlobalThemeSettings({ enable_apple_theme: true, default_theme_mode: 'dark' })
    expect(useTheme().effectiveMode.value).toBe('dark')

    useTheme().setThemeMode('light')
    applyGlobalThemeSettings({ enable_apple_theme: true, default_theme_mode: 'dark' })

    expect(useTheme().effectiveMode.value).toBe('light')
    expect(localStorage.getItem('theme')).toBe('light')
  })

  it('uses the browser local time for time mode', () => {
    expect(getTimeBasedTheme(new Date(2026, 8, 5, 6, 59))).toBe('dark')
    expect(getTimeBasedTheme(new Date(2026, 8, 5, 7, 0))).toBe('light')
    expect(getTimeBasedTheme(new Date(2026, 8, 5, 18, 59))).toBe('light')
    expect(getTimeBasedTheme(new Date(2026, 8, 5, 19, 0))).toBe('dark')
  })

  it('accepts the time-based server default', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date(2026, 8, 5, 20, 0))

    applyGlobalThemeSettings({ enable_apple_theme: true, default_theme_mode: 'time' })
    expect(useTheme().themeMode.value).toBe('time')
    expect(useTheme().effectiveMode.value).toBe('dark')
    expect(document.documentElement.classList.contains('dark')).toBe(true)

    useTheme().setThemeMode('time')
    expect(localStorage.getItem('theme')).toBe('time')
  })
})
