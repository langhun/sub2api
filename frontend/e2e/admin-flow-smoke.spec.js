import { test, expect } from '@playwright/test'
import { bootstrapAdminPage } from './support/helpers.js'

function fieldContainer(page, label) {
  return page.locator('label.input-label', { hasText: label }).locator('..')
}

async function openPaymentSettings(page) {
  await page.goto('/admin/settings')
  await expect(page).toHaveURL(/\/admin\/settings$/)

  const paymentTab = page.locator('#settings-tab-payment')
  await paymentTab.click()
  await expect(paymentTab).toHaveAttribute('aria-selected', 'true')
  await expect(page.getByText('支付设置')).toBeVisible()
}

async function enablePaymentProviderManagement(page) {
  await openPaymentSettings(page)

  await page.getByText('启用支付').locator('xpath=ancestor::div[1]//button').click()
  await expect(page.getByRole('button', { name: '易支付' })).toBeVisible()
  await page.getByRole('button', { name: '易支付' }).click()
  await page.getByRole('button', { name: '保存设置' }).last().click()

  await expect(page.getByText('设置保存成功')).toBeVisible()
  await expect(page.getByRole('button', { name: '创建服务商' })).toBeVisible()
}

async function createEasyPayProvider(page, name) {
  await page.getByRole('button', { name: '创建服务商' }).click()
  await expect(page.getByRole('heading', { name: '创建服务商' })).toBeVisible()

  await fieldContainer(page, '服务商名称').locator('input').fill(name)
  await fieldContainer(page, 'PID').locator('input').fill('2088123412341234')
  await fieldContainer(page, 'PKey').locator('textarea').fill('admin-flow-smoke-secret')
  await fieldContainer(page, 'API 基础地址').locator('input').fill('https://pay.example.com')

  await page.getByRole('button', { name: '保存' }).click()

  await expect(page.getByRole('heading', { name: '创建服务商' })).toHaveCount(0)
  await expect(page.getByText(name, { exact: true })).toBeVisible()
}

test.beforeEach(async ({ page }) => {
  await bootstrapAdminPage(page)
})

test('管理台 smoke 串联 dashboard、payment provider 与 proxies 页面', async ({ page }) => {
  const providerName = '管理联动 Smoke 服务商'

  await page.goto('/admin/dashboard')
  await expect(page).toHaveURL(/\/admin\/dashboard$/)
  await expect(page.getByRole('heading', { name: '系统概览' })).toBeVisible()

  await enablePaymentProviderManagement(page)
  await createEasyPayProvider(page, providerName)

  await page.goto('/admin/proxies')
  await expect(page).toHaveURL(/\/admin\/proxies$/)
  await expect(page.getByRole('button', { name: '添加代理' })).toBeVisible()
  await expect(page.getByPlaceholder('搜索代理...')).toBeVisible()
  await expect(page.locator('[data-test="proxy-toolbar-pool"]')).toBeVisible()
})
