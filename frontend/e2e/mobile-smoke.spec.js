import { test, expect } from '@playwright/test'
import { bootstrapAdminPage } from './support/helpers.js'

test('移动端后台 smoke: 打开菜单并进入代理管理', async ({ page }) => {
  await bootstrapAdminPage(page)
  await page.goto('/dashboard')

  if (await page.locator('#email').count()) {
    await page.locator('#email').fill('admin@example.com')
    await page.locator('#password').fill('Passw0rd!')
    await page.getByRole('button', { name: '登录' }).click()
  }

  await expect(page).toHaveURL(/\/(admin\/dashboard|dashboard)$/)
  await page.getByRole('button', { name: 'Toggle Menu' }).click()
  await page.getByRole('link', { name: '代理管理' }).click()

  await expect(page).toHaveURL(/\/admin\/proxies$/)
})
