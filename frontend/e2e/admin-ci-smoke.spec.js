import { test, expect } from '@playwright/test'
import { bootstrapAdminPage } from './support/helpers.js'

test.beforeEach(async ({ page }) => {
  await bootstrapAdminPage(page)
})

test('CI smoke covers admin dashboard and key routes', async ({ page }) => {
  await page.goto('/dashboard')
  await expect(page).toHaveURL(/\/dashboard$/)
  await expect(page.getByRole('heading', { name: '系统概览' })).toBeVisible()

  await page.goto('/admin/settings')
  await expect(page).toHaveURL(/\/admin\/settings$/)
  const paymentTab = page.locator('#settings-tab-payment')
  await paymentTab.click()
  await expect(paymentTab).toHaveAttribute('aria-selected', 'true')
  await expect(page.getByRole('heading', { name: '支付设置' })).toBeVisible()

  await page.goto('/admin/accounts')
  await expect(page).toHaveURL(/\/admin\/accounts$/)
  await expect(page.getByRole('button', { name: '添加账号' })).toBeVisible()
  await expect(page.getByPlaceholder('搜索账号...')).toBeVisible()
})
