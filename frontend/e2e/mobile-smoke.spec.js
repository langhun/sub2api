import { test, expect } from '@playwright/test'
import { bootstrapAdminPage } from './support/helpers.js'

test('移动端后台 smoke: 打开菜单并进入代理管理', async ({ page }) => {
  await bootstrapAdminPage(page)
  await page.goto('/admin/dashboard')

  await expect(page.getByRole('heading', { name: '系统概览' })).toBeVisible()
  await page.getByRole('button', { name: 'Toggle Menu' }).click()
  await page.getByRole('link', { name: '代理管理' }).click()

  await expect(page).toHaveURL(/\/admin\/proxies$/)
  await expect(page.getByRole('button', { name: '添加代理' })).toBeVisible()
})
