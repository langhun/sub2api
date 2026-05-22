import { test, expect } from '@playwright/test'
import { bootstrapAdminPage } from './support/helpers.js'

async function acceptConfirmAfterClick(page, trigger) {
  const dialogPromise = page.waitForEvent('dialog')
  await trigger.click()
  const dialog = await dialogPromise
  await dialog.accept()
}

test.beforeEach(async ({ page }) => {
  await bootstrapAdminPage(page)
})

test('设置页支持管理员 API Key 的创建重建与删除', async ({ page }) => {
  await page.goto('/admin/settings')

  await expect(page).toHaveURL(/\/admin\/settings$/)

  const securityTab = page.locator('#settings-tab-security')
  await securityTab.click()
  await expect(securityTab).toHaveAttribute('aria-selected', 'true')
  await expect(page.getByText('尚未配置管理员 API Key')).toBeVisible()

  await page.getByRole('button', { name: '创建密钥' }).click()

  await expect(page.getByText('新的管理员 API Key 已生成')).toBeVisible()
  await expect(page.getByText('sk-admin-e2e-0001-secret')).toBeVisible()
  await expect(page.getByRole('button', { name: '复制密钥' })).toBeVisible()

  await page.reload()
  await expect(page).toHaveURL(/\/admin\/settings$/)

  const reloadedSecurityTab = page.locator('#settings-tab-security')
  await reloadedSecurityTab.click()
  await expect(reloadedSecurityTab).toHaveAttribute('aria-selected', 'true')
  await expect(page.getByText('sk-admin-e...cret')).toBeVisible()
  await expect(page.getByText('sk-admin-e2e-0001-secret')).toHaveCount(0)
  await expect(page.getByRole('button', { name: '重新生成' })).toBeVisible()
  await expect(page.getByRole('button', { name: '删除' })).toBeVisible()

  await acceptConfirmAfterClick(
    page,
    page.getByRole('button', { name: '重新生成' }),
  )
  await expect(page.getByText('新的管理员 API Key 已生成')).toBeVisible()
  await expect(page.getByText('sk-admin-e2e-0002-secret')).toBeVisible()

  await acceptConfirmAfterClick(
    page,
    page.getByRole('button', { name: '删除' }),
  )
  await expect(page.getByText('管理员 API Key 已删除')).toBeVisible()
  await expect(page.getByText('尚未配置管理员 API Key')).toBeVisible()
  await expect(page.getByRole('button', { name: '创建密钥' })).toBeVisible()

  await page.reload()
  await expect(page).toHaveURL(/\/admin\/settings$/)
  await page.locator('#settings-tab-security').click()
  await expect(page.getByText('尚未配置管理员 API Key')).toBeVisible()
})
