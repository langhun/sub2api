import { test, expect } from '@playwright/test'
import { bootstrapAdminPage } from './support/helpers.js'

test.beforeEach(async ({ page }) => {
  await bootstrapAdminPage(page)
})

test('设置页可加载并保存系统设置', async ({ page }) => {
  await page.goto('/admin/settings')

  await expect(page).toHaveURL(/\/admin\/settings$/)
  await expect(page.locator('#settings-tab-general')).toBeVisible()
  await expect(page.locator('#settings-tab-gateway')).toBeVisible()

  await page.locator('#settings-tab-gateway').click()
  await expect(page.locator('#settings-tab-gateway')).toHaveAttribute('aria-selected', 'true')

  await page.locator('#settings-tab-general').click()
  await page.getByPlaceholder('Sub2API').fill('Sub2API E2E')

  const saveButton = page.getByRole('button', { name: '保存设置' }).last()
  await saveButton.click()

  await expect(page.getByText('设置保存成功')).toBeVisible()
})
