import { afterEach, describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import postcss from 'postcss'

const themeCss = readFileSync(resolve(__dirname, '../theme.css'), 'utf8')
const inlineCodeRules: postcss.Rule[] = []
const transparentTextSurfaceRules: postcss.Rule[] = []
postcss.parse(themeCss).walkRules((rule) => {
  if (rule.selector.includes('code:not(pre code)')) inlineCodeRules.push(rule)
  if (rule.selector.includes('.dimension-cell') && rule.selector.includes('.summary-value')) {
    transparentTextSurfaceRules.push(rule)
  }
})

afterEach(() => {
  document.documentElement.removeAttribute('data-theme')
  document.documentElement.removeAttribute('data-glass')
  document.documentElement.classList.remove('dark')
  document.body.innerHTML = ''
})

describe('glass inline code surfaces', () => {
  it.each([
    ['data-theme', 'apple', false],
    ['data-theme', 'apple', true],
    ['data-glass', '', false],
    ['data-glass', '', true],
  ] as const)('keeps %s=%s dark=%s text transparent without changing code blocks', (attribute, value, dark) => {
    document.documentElement.setAttribute(attribute, value)
    document.documentElement.classList.toggle('dark', dark)
    document.body.innerHTML = `
      <div class="card"><code class="sm:flex-1">LOCAL2026</code></div>
      <table><tbody><tr><td><code class="code">sk-example</code></td></tr></tbody></table>
      <article class="docs-markdown"><p><code>command</code></p><pre><code>block</code></pre></article>
      <span class="badge">Active</span>
    `

    for (const code of document.querySelectorAll('code:not(pre code)')) {
      const matches = inlineCodeRules.filter(rule => code.matches(rule.selector))
      expect(matches).toHaveLength(1)
      const backgrounds: postcss.Declaration[] = []
      matches[0]!.walkDecls('background', declaration => { backgrounds.push(declaration) })
      expect(backgrounds).toHaveLength(1)
      expect(backgrounds[0]!.value).toBe('transparent')
      expect(backgrounds[0]!.important).toBe(true)
    }
    for (const element of document.querySelectorAll('pre, pre code, .badge')) {
      expect(inlineCodeRules.some(rule => element.matches(rule.selector))).toBe(false)
    }
  })

  it('leaves the default theme untouched', () => {
    document.documentElement.dataset.theme = 'default'
    document.body.innerHTML = '<code>example</code>'
    const code = document.querySelector('code')!
    expect(inlineCodeRules.some(rule => code.matches(rule.selector))).toBe(false)
  })
})

describe('glass monitor text surfaces', () => {
  it('keeps monitor dimensions and summary values transparent', () => {
    expect(transparentTextSurfaceRules).toHaveLength(1)
    const backgrounds: postcss.Declaration[] = []
    transparentTextSurfaceRules[0]!.walkDecls('background', declaration => { backgrounds.push(declaration) })
    expect(backgrounds).toHaveLength(1)
    expect(backgrounds[0]!.value).toBe('transparent')
    expect(backgrounds[0]!.important).toBe(true)

    const backgroundColors: postcss.Declaration[] = []
    transparentTextSurfaceRules[0]!.walkDecls('background-color', declaration => { backgroundColors.push(declaration) })
    expect(backgroundColors).toHaveLength(1)
    expect(backgroundColors[0]!.value).toBe('transparent')
    expect(backgroundColors[0]!.important).toBe(true)
  })
})
