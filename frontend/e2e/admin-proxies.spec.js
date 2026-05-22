import { test, expect } from '@playwright/test'
import { bootstrapAdminPage } from './support/helpers.js'

async function createProxy(page, { name, host, port }) {
  await page.getByRole('button', { name: '添加代理' }).click()
  const dialog = page.getByRole('dialog', { name: '添加代理' })
  const form = dialog.locator('#create-proxy-form')

  await expect(dialog).toBeVisible()
  await form.getByPlaceholder('请输入代理名称').fill(name)
  await form.getByPlaceholder('例如 127.0.0.1').fill(host)
  await form.getByPlaceholder('例如 8080').fill(String(port))
  await dialog.getByRole('button', { name: '创建' }).click()
}

function proxyRow(page, name) {
  return page.locator('tbody tr').filter({ hasText: name }).first()
}

async function expectProxyStatus(page, name, statusText) {
  const row = proxyRow(page, name)
  await expect(row).toBeVisible()
  await expect(row.getByText(statusText, { exact: true })).toBeVisible()
  return row
}

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

  await createProxy(page, { name: 'e2e-proxy', host: '10.0.0.8', port: 9001 })

  await expect(page.getByText('代理添加成功')).toBeVisible()
  await expect(page.getByText('e2e-proxy')).toBeVisible()
})

test('创建代理后刷新页面仍可见且停用状态在刷新后保持一致', async ({ page }) => {
  const candidate = {
    name: 'e2e-proxy-persist',
    host: '10.0.0.9',
    port: 9002,
  }

  await page.goto('/admin/proxies')

  await expect(page).toHaveURL(/\/admin\/proxies$/)
  await expect(page.getByRole('button', { name: '添加代理' })).toBeVisible()
  await expect(page.getByPlaceholder('搜索代理...')).toBeVisible()

  await createProxy(page, candidate)

  await expect(page.getByText('代理添加成功')).toBeVisible()
  await expect(page.getByText(candidate.name, { exact: true })).toBeVisible()
  await expectProxyStatus(page, candidate.name, '正常')

  await page.reload()
  await expect(page).toHaveURL(/\/admin\/proxies$/)
  await expect(page.getByPlaceholder('搜索代理...')).toBeVisible()
  await expectProxyStatus(page, candidate.name, '正常')

  const createdRow = proxyRow(page, candidate.name)
  await createdRow.getByRole('button', { name: '更多' }).click()
  await expect(page.getByRole('button', { name: '停用代理' })).toBeVisible()
  await page.getByRole('button', { name: '停用代理' }).click()

  await expect(page.getByText('代理已停用')).toBeVisible()
  await expectProxyStatus(page, candidate.name, '停用')

  await page.reload()
  await expect(page).toHaveURL(/\/admin\/proxies$/)
  await expectProxyStatus(page, candidate.name, '停用')

  const disabledRow = proxyRow(page, candidate.name)
  await disabledRow.getByRole('button', { name: '更多' }).click()
  await expect(page.getByRole('button', { name: '启用代理' })).toBeVisible()
})
