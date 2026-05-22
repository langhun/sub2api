import { test, expect } from '@playwright/test'
import { bootstrapAdminPage } from './support/helpers.js'

function fieldContainer(page, label) {
  return page.locator('label.input-label', { hasText: label }).locator('..')
}

test.beforeEach(async ({ page }) => {
  await bootstrapAdminPage(page)
})

test('设置页支持支付配置与服务商创建删除流程', async ({ page }) => {
  await page.goto('/admin/settings')

  await expect(page).toHaveURL(/\/admin\/settings$/)

  const paymentTab = page.locator('#settings-tab-payment')
  await paymentTab.click()
  await expect(paymentTab).toHaveAttribute('aria-selected', 'true')

  await expect(page.getByText('支付设置')).toBeVisible()

  await page.getByText('启用支付').locator('xpath=ancestor::div[1]//button').click()
  await expect(page.getByRole('button', { name: '易支付' })).toBeVisible()

  await page.getByRole('button', { name: '易支付' }).click()

  const saveButton = page.getByRole('button', { name: '保存设置' }).last()
  await saveButton.click()

  await expect(page.getByText('设置保存成功')).toBeVisible()
  await expect(page.getByRole('button', { name: '创建服务商' })).toBeVisible()
  await expect(page.getByText('暂无服务商')).toBeVisible()

  await page.getByRole('button', { name: '创建服务商' }).click()

  await expect(page.getByRole('heading', { name: '创建服务商' })).toBeVisible()
  await fieldContainer(page, '服务商名称').locator('input').fill('易支付 E2E')
  await fieldContainer(page, 'PID').locator('input').fill('2088123412341234')
  await fieldContainer(page, 'PKey').locator('textarea').fill('e2e-easypay-secret')
  await fieldContainer(page, 'API 基础地址').locator('input').fill('https://pay.example.com')

  await page.getByRole('button', { name: '保存' }).click()

  await expect(page.getByRole('heading', { name: '创建服务商' })).toHaveCount(0)
  await expect(page.getByText('易支付 E2E')).toBeVisible()
  await expect(page.getByText('易支付')).toBeVisible()
  await expect(page.getByRole('button', { name: '支付宝' })).toBeVisible()
  await expect(page.getByRole('button', { name: '微信支付' })).toBeVisible()

  await page.getByRole('button', { name: '删除' }).last().click()
  await expect(page.getByText('确定要删除此服务商吗？')).toBeVisible()
  await page.getByRole('button', { name: '删除' }).last().click()

  await expect(page.getByText('删除成功')).toBeVisible()
  await expect(page.getByText('易支付 E2E')).toHaveCount(0)
  await expect(page.getByText('暂无服务商')).toBeVisible()
})
