import { test, expect } from '@playwright/test'
import { bootstrapAdminPage } from './support/helpers.js'

test.beforeEach(async ({ page }) => {
  await bootstrapAdminPage(page)
})

test('代理页展示关键操作并支持创建代理', async ({ page }) => {
  await page.goto('/admin/proxies')

  await expect(page).toHaveURL(/\/admin\/proxies$/)
  await expect(page.getByRole('button', { name: '添加代理' })).toBeVisible()
  await expect(page.getByPlaceholder('搜索代理...')).toBeVisible()
  await expect(page.locator('[data-test="proxy-toolbar-pool"]')).toBeVisible()

  await page.locator('[data-test="proxy-toolbar-batch-toggle"]').click()
  await expect(page.locator('[data-test="proxy-toolbar-batch-test"]')).toBeVisible()
  await expect(page.locator('[data-test="proxy-toolbar-batch-quality"]')).toBeVisible()

  await page.getByRole('button', { name: '添加代理' }).click()
  const dialog = page.getByRole('dialog', { name: '添加代理' })
  const form = dialog.locator('#create-proxy-form')
  await expect(dialog).toBeVisible()

  await form.getByPlaceholder('请输入代理名称').fill('e2e-proxy')
  await form.getByPlaceholder('例如 127.0.0.1').fill('10.0.0.8')
  await form.getByPlaceholder('例如 8080').fill('9001')
  await dialog.getByRole('button', { name: '创建' }).click()

  await expect(page.getByText('代理添加成功')).toBeVisible()
  await expect(page.getByText('e2e-proxy')).toBeVisible()
})

test('新建代理后列表立即可见并支持禁用状态切换', async ({ page }) => {
  const proxyName = 'e2e-proxy-toggle'

  await page.goto('/admin/proxies')

  await expect(page).toHaveURL(/\/admin\/proxies$/)
  await expect(page.getByRole('button', { name: '添加代理' })).toBeVisible()

  await page.getByRole('button', { name: '添加代理' }).click()
  const dialog = page.getByRole('dialog', { name: '添加代理' })
  const form = dialog.locator('#create-proxy-form')
  await expect(dialog).toBeVisible()

  await form.getByPlaceholder('请输入代理名称').fill(proxyName)
  await form.getByPlaceholder('例如 127.0.0.1').fill('10.0.0.9')
  await form.getByPlaceholder('例如 8080').fill('9002')
  await dialog.getByRole('button', { name: '创建' }).click()

  await expect(page.getByText('代理添加成功')).toBeVisible()
  await expect(page.getByText(proxyName, { exact: true })).toBeVisible()

  const proxyRow = page.locator('tbody tr', { has: page.getByText(proxyName, { exact: true }) }).first()
  await expect(proxyRow).toBeVisible()
  await expect(proxyRow.getByText('ACTIVE')).toBeVisible()

  await proxyRow.getByRole('button', { name: '更多' }).click()
  await page.getByRole('button', { name: '禁用代理' }).click()

  await expect(page.getByText('代理已禁用')).toBeVisible()
  await expect(proxyRow.getByText('INACTIVE')).toBeVisible()
  await expect(proxyRow.getByText(proxyName, { exact: true })).toBeVisible()
})
