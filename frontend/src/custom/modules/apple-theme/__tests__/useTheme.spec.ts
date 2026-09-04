import { describe, it, expect } from 'vitest'
import { useTheme } from '../useTheme'

describe('useTheme', () => {
  it('should initialize with default values', () => {
    const { themeMode, themeStyle } = useTheme()

    expect(['light', 'dark', 'auto']).toContain(themeMode.value)
    expect(['default', 'apple']).toContain(themeStyle.value)
  })

  it('should toggle between light and dark modes', () => {
    const { themeMode, toggleMode, effectiveMode } = useTheme()

    const initialMode = effectiveMode.value
    toggleMode()
    const newMode = effectiveMode.value

    expect(newMode).not.toBe(initialMode)
    expect(['light', 'dark']).toContain(newMode)
  })

  it('should set theme mode correctly', () => {
    const { themeMode, setThemeMode } = useTheme()

    setThemeMode('dark')
    expect(themeMode.value).toBe('dark')

    setThemeMode('light')
    expect(themeMode.value).toBe('light')

    setThemeMode('auto')
    expect(themeMode.value).toBe('auto')
  })

  it('should set theme style correctly', () => {
    const { themeStyle, setThemeStyle } = useTheme()

    setThemeStyle('apple')
    expect(themeStyle.value).toBe('apple')

    setThemeStyle('default')
    expect(themeStyle.value).toBe('default')
  })
})
