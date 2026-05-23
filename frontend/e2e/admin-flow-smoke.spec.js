import { test, expect } from '@playwright/test'
import { bootstrapAdminPage } from './support/helpers.js'

async function openPaymentSettings(page) {
  await page.goto('/admin/settings')
  await expect(page).toHaveURL(/\/admin\/settings$/)

  const paymentTab = page.locator('#settings-tab-payment')
  await paymentTab.click()
  await expect(paymentTab).toHaveAttribute('aria-selected', 'true')
  await expect(page.getByRole('heading', { name: '支付设置' })).toBeVisible()
}

async function expectAccountsPageReachable(page) {
  await page.goto('/admin/accounts')
  await expect(page).toHaveURL(/\/admin\/accounts$/)
  await expect(page.getByRole('button', { name: '添加账号' })).toBeVisible()
  await expect(page.getByPlaceholder('搜索账号...')).toBeVisible()
}

async function expectGroupsPageReachable(page) {
  await page.goto('/admin/groups')
  await expect(page).toHaveURL(/\/admin\/groups$/)
  await expect(page.getByRole('button', { name: '创建分组' })).toBeVisible()
  await expect(page.getByPlaceholder('搜索分组...')).toBeVisible()
}

async function expectProxiesPageReachable(page) {
  await page.goto('/admin/proxies')
  await expect(page).toHaveURL(/\/admin\/proxies$/)
  await expect(page.getByRole('button', { name: '添加代理' })).toBeVisible()
  await expect(page.getByPlaceholder('搜索代理...')).toBeVisible()
  await expect(page.locator('[data-test="proxy-toolbar-pool"]')).toBeVisible()
}

test.beforeEach(async ({ page }) => {
  await bootstrapAdminPage(page)
})

test('管理台 smoke 串联跨页访问', async ({ page }) => {

  await page.goto('/dashboard')
  await expect(page).toHaveURL(/\/dashboard$/)
  await expect(page.getByRole('heading', { name: '系统概览' })).toBeVisible()

  await openPaymentSettings(page)
  await expect(page.locator('#settings-tab-payment')).toHaveAttribute('aria-selected', 'true')

  await expectAccountsPageReachable(page)
  await expectGroupsPageReachable(page)
  await expectProxiesPageReachable(page)
})
