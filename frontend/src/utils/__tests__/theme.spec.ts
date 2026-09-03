import { afterEach, beforeEach, describe, expect, it } from 'vitest'

import { applyThemeStyle, readThemeStyle } from '@/utils/theme'

describe('theme style', () => {
  beforeEach(() => {
    delete window.__APP_CONFIG__
    localStorage.clear()
    document.documentElement.removeAttribute('data-glass')
  })

  afterEach(() => {
    delete window.__APP_CONFIG__
    localStorage.clear()
    document.documentElement.removeAttribute('data-glass')
  })

  it('reads the server-injected global theme and ignores local overrides', () => {
    localStorage.setItem('theme_style', 'classic')
    expect(readThemeStyle()).toBe('glass')

    window.__APP_CONFIG__ = { theme_style: 'classic' }
    expect(readThemeStyle()).toBe('classic')
  })

  it('applies the global theme without persisting a user override', () => {
    applyThemeStyle('glass')
    expect(document.documentElement.dataset.glass).toBe('true')
    expect(localStorage.getItem('theme_style')).toBeNull()

    applyThemeStyle('classic')
    expect(document.documentElement.hasAttribute('data-glass')).toBe(false)
    expect(localStorage.getItem('theme_style')).toBeNull()
  })
})
