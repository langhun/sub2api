import { expect } from '@playwright/test'
import { ADMIN_EMAIL, ADMIN_PASSWORD } from './fixtures.js'
import { mockCommonAppRoutes } from './mock-api.js'

export async function bootstrapAdminPage(page, options = {}) {
  const state = await mockCommonAppRoutes(page, options)
  await page.addInitScript(() => {
    window.localStorage.setItem('sub2api_locale', 'zh')
    window.localStorage.setItem('admin_guide_guest_user_v4_interactive', 'true')
    window.localStorage.setItem('admin_guide_1_admin_v4_interactive', 'true')
    window.localStorage.setItem('user_guide_guest_user_v4_interactive', 'true')
    window.localStorage.setItem('user_guide_1_admin_v4_interactive', 'true')
  })
  return state
}

export async function loginAsAdmin(page, options = {}) {
  const state = await bootstrapAdminPage(page, options)
  await page.goto('/login')
  await page.locator('#email').fill(ADMIN_EMAIL)
  await page.locator('#password').fill(ADMIN_PASSWORD)
  await page.getByRole('button', { name: '登录' }).click()
  await expect(page).toHaveURL(/\/(admin\/dashboard|dashboard)$/)
  return state
}
