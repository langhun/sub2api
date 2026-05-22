import { test, expect } from '@playwright/test'
import { bootstrapAdminPage } from './support/helpers.js'

function fieldContainer(page, label) {
  return page.locator('label.input-label', { hasText: label }).locator('..')
}

function providerSwitch(page, label) {
  return page.locator('label').filter({ hasText: new RegExp(`^${label}$`) }).getByRole('switch')
}

function providerCard(page, name) {
  return page
    .getByText(name, { exact: true })
    .locator('xpath=ancestor::div[.//button[normalize-space()="编辑"] and .//button[normalize-space()="删除"]][1]')
}

function paymentTypeBadge(page, providerName, label) {
  return providerCard(page, providerName).getByRole('button', { name: label })
}

async function expectPaymentTypeBadgeSelected(page, providerName, label, selected) {
  const badge = paymentTypeBadge(page, providerName, label)
  if (selected) {
    await expect(badge).toHaveClass(/bg-primary-500/)
    await expect(badge).toHaveClass(/text-white/)
    return
  }

  await expect(badge).toHaveClass(/bg-gray-100/)
  await expect(badge).toHaveClass(/text-gray-400/)
}

async function togglePaymentTypeBadge(page, providerName, label, expectedSupportedTypes) {
  const updateRequestPromise = page.waitForRequest((request) => {
    const pathname = new URL(request.url()).pathname
    return request.method() === 'PUT' && /\/api\/v1\/admin\/payment\/providers\/\d+$/.test(pathname)
  })

  await paymentTypeBadge(page, providerName, label).click()

  const updateRequest = await updateRequestPromise
  expect(updateRequest.postDataJSON()).toMatchObject({
    supported_types: expectedSupportedTypes,
  })
}

async function openPaymentSettings(page) {
  await page.goto('/admin/settings')
  await expect(page).toHaveURL(/\/admin\/settings$/)

  const paymentTab = page.locator('#settings-tab-payment')
  await paymentTab.click()
  await expect(paymentTab).toHaveAttribute('aria-selected', 'true')
  await expect(page.getByText('支付设置')).toBeVisible()
}

async function enablePaymentAndOpenProviderManagement(page) {
  await openPaymentSettings(page)

  await page.getByText('启用支付').locator('xpath=ancestor::div[1]//button').click()
  await expect(page.getByRole('button', { name: '易支付' })).toBeVisible()
  await page.getByRole('button', { name: '易支付' }).click()

  await page.getByRole('button', { name: '保存设置' }).last().click()

  await expect(page.getByText('设置保存成功')).toBeVisible()
  await expect(page.getByRole('button', { name: '创建服务商' })).toBeVisible()
  await expect(page.getByText('暂无服务商')).toBeVisible()
}

async function createEasyPayProvider(page, name = '易支付 E2E') {
  await page.getByRole('button', { name: '创建服务商' }).click()

  await expect(page.getByRole('heading', { name: '创建服务商' })).toBeVisible()
  await fieldContainer(page, '服务商名称').locator('input').fill(name)
  await fieldContainer(page, 'PID').locator('input').fill('2088123412341234')
  await fieldContainer(page, 'PKey').locator('textarea').fill('e2e-easypay-secret')
  await fieldContainer(page, 'API 基础地址').locator('input').fill('https://pay.example.com')

  await page.getByRole('button', { name: '保存' }).click()

  await expect(page.getByRole('heading', { name: '创建服务商' })).toHaveCount(0)
  const card = providerCard(page, name)
  await expect(card).toBeVisible()
  await expect(page.getByText('易支付')).toBeVisible()
  await expect(card.getByRole('button', { name: '支付宝' })).toBeVisible()
  await expect(card.getByRole('button', { name: '微信支付' })).toBeVisible()
}

test.beforeEach(async ({ page }) => {
  await bootstrapAdminPage(page)
})

test('设置页支持支付配置与服务商创建删除流程', async ({ page }) => {
  await enablePaymentAndOpenProviderManagement(page)
  await createEasyPayProvider(page)

  await page.getByRole('button', { name: '删除' }).last().click()
  await expect(page.getByText('确定要删除此服务商吗？')).toBeVisible()
  await page.getByRole('button', { name: '删除' }).last().click()

  await expect(page.getByText('删除成功')).toBeVisible()
  await expect(page.getByText('易支付 E2E')).toHaveCount(0)
  await expect(page.getByText('暂无服务商')).toBeVisible()
})

test('设置页支持编辑已创建服务商并刷新列表名称', async ({ page }) => {
  await enablePaymentAndOpenProviderManagement(page)
  await createEasyPayProvider(page, '易支付 原始名称')

  await page.getByRole('button', { name: '编辑' }).click()
  await expect(page.getByRole('heading', { name: '编辑服务商' })).toBeVisible()

  const nameInput = fieldContainer(page, '服务商名称').locator('input')
  await expect(nameInput).toHaveValue('易支付 原始名称')
  await nameInput.fill('易支付 已编辑')

  await page.getByRole('button', { name: '保存' }).click()

  await expect(page.getByRole('heading', { name: '编辑服务商' })).toHaveCount(0)
  await expect(page.getByText('易支付 已编辑')).toBeVisible()
  await expect(page.getByText('易支付 原始名称')).toHaveCount(0)
})

test('设置页支持切换服务商启用状态', async ({ page }) => {
  await enablePaymentAndOpenProviderManagement(page)
  await createEasyPayProvider(page, '易支付 开关测试')

  const enabledSwitch = providerSwitch(page, '启用')
  await expect(enabledSwitch).toHaveAttribute('aria-checked', 'true')

  await enabledSwitch.click()
  await expect(enabledSwitch).toHaveAttribute('aria-checked', 'false')

  await enabledSwitch.click()
  await expect(enabledSwitch).toHaveAttribute('aria-checked', 'true')
})

test('设置页支持切换退款与用户退款开关', async ({ page }) => {
  await enablePaymentAndOpenProviderManagement(page)
  await createEasyPayProvider(page, '易支付 退款测试')

  const refundSwitch = providerSwitch(page, '允许退款')
  await expect(refundSwitch).toHaveAttribute('aria-checked', 'false')
  await expect(page.locator('label').filter({ hasText: /^允许用户退款$/ })).toHaveCount(0)

  await refundSwitch.click()
  await expect(refundSwitch).toHaveAttribute('aria-checked', 'true')

  const userRefundSwitch = providerSwitch(page, '允许用户退款')
  await expect(userRefundSwitch).toHaveAttribute('aria-checked', 'false')

  await userRefundSwitch.click()
  await expect(userRefundSwitch).toHaveAttribute('aria-checked', 'true')

  await refundSwitch.click()
  await expect(refundSwitch).toHaveAttribute('aria-checked', 'false')
  await expect(page.locator('label').filter({ hasText: /^允许用户退款$/ })).toHaveCount(0)
})

test('设置页支持切换服务商支付方式 badge 并在刷新后保持状态', async ({ page }) => {
  const providerName = '易支付 支付方式测试'
  await enablePaymentAndOpenProviderManagement(page)
  await createEasyPayProvider(page, providerName)

  await expectPaymentTypeBadgeSelected(page, providerName, '支付宝', true)
  await expectPaymentTypeBadgeSelected(page, providerName, '微信支付', true)

  await togglePaymentTypeBadge(page, providerName, '支付宝', ['wxpay'])
  await expectPaymentTypeBadgeSelected(page, providerName, '支付宝', false)
  await expectPaymentTypeBadgeSelected(page, providerName, '微信支付', true)

  await togglePaymentTypeBadge(page, providerName, '支付宝', ['wxpay', 'alipay'])
  await expectPaymentTypeBadgeSelected(page, providerName, '支付宝', true)
  await expectPaymentTypeBadgeSelected(page, providerName, '微信支付', true)

  await togglePaymentTypeBadge(page, providerName, '微信支付', ['alipay'])
  await expectPaymentTypeBadgeSelected(page, providerName, '支付宝', true)
  await expectPaymentTypeBadgeSelected(page, providerName, '微信支付', false)

  await page.reload()
  await openPaymentSettings(page)
  await expect(page.getByText(providerName)).toBeVisible()
  await expectPaymentTypeBadgeSelected(page, providerName, '支付宝', true)
  await expectPaymentTypeBadgeSelected(page, providerName, '微信支付', false)
})
