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

  it('lets the title use the space left by header actions without self-relative clipping', () => {
    expect(source).toContain('class="flex min-w-0 flex-1 items-center gap-2 sm:gap-4"')
    expect(source).toContain('class="hidden min-w-0 flex-1 lg:block"')
    expect(source).toContain('class="flex min-w-0 shrink-0 items-center justify-end gap-1 overflow-visible sm:gap-3"')
    expect(source).not.toContain('max-w-[45%]')
  })
})
