import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../AppLayout.vue')
const componentSource = readFileSync(componentPath, 'utf8')

describe('AppLayout', () => {
  it('does not pass a non-prop style attribute to the fragment sidebar', () => {
    expect(componentSource).toContain('<AppSidebar />')
    expect(componentSource).not.toContain('<AppSidebar style=')
  })
})
