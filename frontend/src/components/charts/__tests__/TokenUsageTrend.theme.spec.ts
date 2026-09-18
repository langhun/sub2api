import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const source = readFileSync(
  resolve(dirname(fileURLToPath(import.meta.url)), '../TokenUsageTrend.vue'),
  'utf8',
)

describe('TokenUsageTrend theme binding', () => {
  it('uses the reactive theme mode instead of reading the DOM once', () => {
    expect(source).toContain("import { useTheme } from '@/custom/modules/apple-theme/useTheme'")
    expect(source).toContain("effectiveMode.value === 'dark'")
    expect(source).not.toContain("document.documentElement.classList.contains('dark')")
  })
})
