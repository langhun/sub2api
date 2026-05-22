import { test as setup, expect } from '@playwright/test'
import { loginAsAdmin } from './support/helpers.js'

setup('authenticate admin user', async ({ page }) => {
  await loginAsAdmin(page)
  await expect(page.getByRole('heading', { name: '系统概览' })).toBeVisible()
  await page.context().storageState({ path: './e2e/.auth/admin.json' })
})
