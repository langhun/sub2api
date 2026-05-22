import { test, expect } from '@playwright/test'
import { bootstrapAdminPage } from './support/helpers.js'

test.beforeEach(async ({ page }) => {
  await bootstrapAdminPage(page)
})

test('账户页支持搜索、展开高级筛选并清空筛选', async ({ page }) => {
  await page.goto('/admin/accounts')

  await expect(page).toHaveURL(/\/admin\/accounts$/)
  await expect(page.getByRole('button', { name: '添加账号' })).toBeVisible()
  await expect(page.getByPlaceholder('搜索账号...')).toBeVisible()
  await expect(page.getByText('claude-main')).toBeVisible()
  await expect(page.getByText('ungrouped-openai')).toBeVisible()

  await page.getByPlaceholder('搜索账号...').fill('claude-main')
  await expect(page.getByText('claude-main')).toBeVisible()
  await expect(page.getByText('ungrouped-openai')).toHaveCount(0)
  await expect(page.getByText('已筛选 1 项')).toBeVisible()

  await page.getByTestId('account-more-filters-toggle').click()
  await expect(page.getByRole('button', { name: '收起高级筛选' })).toBeVisible()
  await expect(page.getByText('全部分组').first()).toBeVisible()

  await page.getByRole('button', { name: '清空筛选' }).click()
  await expect(page.getByText('已筛选 1 项')).toHaveCount(0)
  await expect(page.getByText('claude-main')).toBeVisible()
  await expect(page.getByText('ungrouped-openai')).toBeVisible()
})
