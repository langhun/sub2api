import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const source = readFileSync(resolve(dirname(fileURLToPath(import.meta.url)), '../AppHeader.vue'), 'utf8')

describe('AppHeader document listener lifecycle', () => {
  it('registers and removes the outside-click listener', () => {
    expect(source).toContain("onMounted(() => {\n  document.addEventListener('click', handleClickOutside)")
    expect(source).toContain("onBeforeUnmount(() => {\n  document.removeEventListener('click', handleClickOutside)")
  })
})
