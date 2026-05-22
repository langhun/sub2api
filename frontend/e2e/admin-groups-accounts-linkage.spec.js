import { test, expect } from '@playwright/test'
import { bootstrapAdminPage } from './support/helpers.js'

test.beforeEach(async ({ page }) => {
  await bootstrapAdminPage(page)
})

test('创建分组后可在账户页分组筛选中看到并按该分组过滤', async ({ page }) => {
  const groupName = 'E2E 跨页分组联动'
  const groupDescription = '验证创建分组后账户页筛选可见'

  await page.goto('/admin/groups')

  await expect(page).toHaveURL(/\/admin\/groups$/)
  await page.getByRole('button', { name: '创建分组' }).click()

  const dialog = page.getByRole('dialog', { name: '创建分组' })
  await expect(dialog).toBeVisible()

  await dialog.getByPlaceholder('请输入分组名称').fill(groupName)
  await dialog.getByPlaceholder('可选描述').fill(groupDescription)
  await dialog.getByRole('button', { name: '创建' }).click()

  await expect(page.getByText('分组创建成功')).toBeVisible()
  await expect(page.getByText(groupName, { exact: true })).toBeVisible()

  await page.goto('/admin/accounts')

  await expect(page).toHaveURL(/\/admin\/accounts$/)
  await expect(page.getByText('claude-main')).toBeVisible()
  await expect(page.getByText('ungrouped-openai')).toBeVisible()

  await page.getByTestId('account-more-filters-toggle').click()
  await expect(page.getByRole('button', { name: '收起高级筛选' })).toBeVisible()

  const groupSelect = page.getByRole('button', { name: '全部分组' }).first()
  await expect(groupSelect).toBeVisible()
  await groupSelect.click()

  const groupOption = page.getByRole('option', { name: groupName })
  await expect(groupOption).toBeVisible()
  await groupOption.click()

  await expect(page.getByText('已筛选 1 项')).toBeVisible()
  await expect(page.getByText(`分组：${groupName}`)).toBeVisible()
  await expect(page.getByText('claude-main')).toHaveCount(0)
  await expect(page.getByText('ungrouped-openai')).toHaveCount(0)
  await expect(page.getByText('暂无账号')).toBeVisible()
})

test('按新建分组筛选后清空筛选可恢复账户列表', async ({ page }) => {
  const groupName = 'E2E 清空筛选恢复'

  await page.goto('/admin/groups')
  await expect(page).toHaveURL(/\/admin\/groups$/)

  await page.getByRole('button', { name: '创建分组' }).click()
  const dialog = page.getByRole('dialog', { name: '创建分组' })
  await expect(dialog).toBeVisible()

  await dialog.getByPlaceholder('请输入分组名称').fill(groupName)
  await dialog.getByPlaceholder('可选描述').fill('验证账户页清空筛选恢复列表')
  await dialog.getByRole('button', { name: '创建' }).click()

  await expect(page.getByText('分组创建成功')).toBeVisible()
  await expect(page.getByText(groupName, { exact: true })).toBeVisible()

  await page.goto('/admin/accounts')
  await expect(page).toHaveURL(/\/admin\/accounts$/)
  await expect(page.getByText('claude-main')).toBeVisible()
  await expect(page.getByText('ungrouped-openai')).toBeVisible()

  await page.getByTestId('account-more-filters-toggle').click()
  await expect(page.getByRole('button', { name: '收起高级筛选' })).toBeVisible()

  const groupSelect = page.getByRole('button', { name: '全部分组' }).first()
  await expect(groupSelect).toBeVisible()
  await groupSelect.click()

  const groupOption = page.getByRole('option', { name: groupName })
  await expect(groupOption).toBeVisible()
  await groupOption.click()

  await expect(page.getByText(`分组：${groupName}`)).toBeVisible()
  await expect(page.getByText('已筛选 1 项')).toBeVisible()
  await expect(page.getByText('暂无账号')).toBeVisible()

  await page.getByRole('button', { name: '清空筛选' }).click()

  await expect(page.getByText(`分组：${groupName}`)).toHaveCount(0)
  await expect(page.getByText('已筛选 1 项')).toHaveCount(0)
  await expect(page.getByRole('button', { name: '全部分组' }).first()).toBeVisible()
  await expect(page.getByText('claude-main')).toBeVisible()
  await expect(page.getByText('ungrouped-openai')).toBeVisible()
  await expect(page.getByText('暂无账号')).toHaveCount(0)
})
