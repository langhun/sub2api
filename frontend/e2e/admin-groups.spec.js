import { test, expect } from '@playwright/test'
import { bootstrapAdminPage } from './support/helpers.js'

test.beforeEach(async ({ page }) => {
  await bootstrapAdminPage(page)
})

test('分组页支持创建新分组并展示在列表中', async ({ page }) => {
  await page.goto('/admin/groups')

  await expect(page).toHaveURL(/\/admin\/groups$/)
  await expect(page.getByRole('button', { name: '创建分组' })).toBeVisible()
  await expect(page.getByPlaceholder('搜索分组...')).toBeVisible()

  await page.getByRole('button', { name: '创建分组' }).click()

  const dialog = page.getByRole('dialog', { name: '创建分组' })
  await expect(dialog).toBeVisible()

  await dialog.getByPlaceholder('请输入分组名称').fill('E2E Anthropic 组')
  await dialog.getByPlaceholder('可选描述').fill('用于 Playwright E2E 创建流程')
  await dialog.getByRole('button', { name: '创建' }).click()

  await expect(page.getByText('分组创建成功')).toBeVisible()
  await expect(page.getByText('E2E Anthropic 组')).toBeVisible()
})
