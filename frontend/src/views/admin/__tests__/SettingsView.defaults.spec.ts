import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../SettingsView.vue')
const componentSource = readFileSync(componentPath, 'utf8')

describe('SettingsView defaults', () => {
  it('normalizes an omitted backend mode setting to a boolean', () => {
    expect(componentSource).toContain('form.backend_mode_enabled = Boolean(settings.backend_mode_enabled)')
  })
})
