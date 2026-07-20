import { describe, expect, it } from 'vitest'
import zhAudit from '@/i18n/locales/zh/admin/audit'
import { formatAuditAction, formatAuditActorRole } from '../auditLogActionLabel'

const actionSegments = zhAudit.audit.actionSegments as Record<string, string>
const roles = zhAudit.audit.roles as Record<string, string>
const actionPrefix = 'admin.audit.actionSegments.'
const rolePrefix = 'admin.audit.roles.'
const messageFromKey = (key: string) => {
  if (key.startsWith(actionPrefix)) return actionSegments[key.slice(actionPrefix.length)]
  if (key.startsWith(rolePrefix)) return roles[key.slice(rolePrefix.length)]
  return undefined
}
const hasTranslation = (key: string) => messageFromKey(key) !== undefined
const translate = (key: string) => messageFromKey(key) ?? key

describe('formatAuditAction', () => {
  it.each([
    ['auth.validate_invitation_code.create', '认证 · 校验邀请码 · 创建'],
    ['auth.oauth.pending.exchange.create', '认证 · OAuth 授权 · 待处理 · 兑换 · 创建'],
    ['admin.accounts.test.create', '账号 · 测试 · 创建'],
    ['admin.accounts.today_stats.batch.create', '账号 · 今日统计 · 批量 · 创建'],
    ['admin.accounts.upstream_billing_probe.batch.create', '账号 · 上游计费探测 · 批量 · 创建'],
    ['admin.dashboard.api_keys_usage.create', '仪表盘 · API 密钥用量 · 创建'],
    ['admin.accounts.models.sync_upstream.create', '账号 · 模型 · 同步上游 · 创建']
  ])('translates route-derived action %s', (action, expected) => {
    expect(formatAuditAction(action, translate, hasTranslation)).toBe(expected)
  })

  it('composes translated tokens for new snake-case route segments', () => {
    expect(formatAuditAction('admin.accounts.quota_window_reset.create', translate, hasTranslation))
      .toBe('账号 · 配额窗口重置 · 创建')
  })

  it('keeps unknown route tokens visible for diagnostics', () => {
    expect(formatAuditAction('admin.accounts.future_action.create', translate, hasTranslation))
      .toBe('账号 · future 操作 · 创建')
  })
})

describe('formatAuditActorRole', () => {
  it.each([
    ['admin', '管理员'],
    ['user', '用户'],
    ['operator', 'operator']
  ])('formats actor role %s', (role, expected) => {
    expect(formatAuditActorRole(role, translate, hasTranslation)).toBe(expected)
  })
})
