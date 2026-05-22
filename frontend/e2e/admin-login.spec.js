import { test, expect } from '@playwright/test'
import { bootstrapAdminPage } from './support/helpers.js'

test('管理员可从登录页进入后台控制台', async ({ page }) => {
  await bootstrapAdminPage(page)

  await page.goto('/login')

  await expect(page.getByRole('heading', { name: '欢迎回来' })).toBeVisible()
  await page.locator('#email').fill('admin@example.com')
  await page.locator('#password').fill('Passw0rd!')
  await page.getByRole('button', { name: '登录' }).click()

  await expect(page).toHaveURL(/\/admin\/dashboard$/)
  await expect(page.getByText('系统概览')).toBeVisible()
  await expect(page.getByRole('heading', { name: '系统概览' })).toBeVisible()
})
