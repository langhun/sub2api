import { describe, expect, it } from 'vitest'
import { formatAuditAction } from '../actionLabels'

const capturedProductionActions = `
admin.accounts.batch_delete.create
admin.accounts.bulk_update.create
admin.accounts.check_mixed_channel.create
admin.accounts.create
admin.accounts.delete
admin.accounts.duplicate.create
admin.accounts.import.codex_session.create
admin.accounts.models.sync_upstream.create
admin.accounts.models.sync_upstream_preview.create
admin.accounts.recover_state.create
admin.accounts.refresh.create
admin.accounts.schedulable.create
admin.accounts.test.create
admin.accounts.today_stats.batch.create
admin.accounts.upstream_billing_probe.batch.create
admin.accounts.upstream_billing_probe.settings.update
admin.accounts.usage.batch.create
admin.announcements.create
admin.api_keys.update
admin.backups.s3_config.test.create
admin.backups.schedule.update
admin.channel_monitor_v2.config.update
admin.channel_monitors.run.create
admin.compliance.accept.create
admin.dashboard.users_usage.create
admin.docs.revisions.review.create
admin.gemini.oauth.auth_url.create
admin.groups.rate_multipliers.update
admin.openai.accounts.quota.refresh.create
admin.payment.orders.refund.create
admin.proxies.quality_check.create
admin.redeem_codes.generate.create
admin.settings.email_templates.update
admin.settings.send_test_email.create
admin.settings.web_search_emulation.update
admin.support.tickets.messages.create
admin.user_attributes.batch.create
announcements.read.create
auth.forgot_password.create
auth.logout.create
auth.oauth.pending.send_verify_code.create
auth.passkey.login.begin.create
auth.validate_invitation_code.create
checkin.luck.create
custom.admin.openai.accounts.refresh_token.create
custom.admin.support_contacts.qr.delete
docs.revisions.create
keys.create
security.account_abuse_owner_cleanup
security.disabled_account_api_key_enforcement
security.registration_abuse_cleanup
security.registration_abuse_followup_cleanup
support.tickets.reopen.create
user.account_bindings.email.send_code.create
user.aff.transfer.create
user.passkeys.register.finish.create
user.totp.disable.create
user.totp.enable.create
user.totp.setup.create
`.trim().split('\n')

describe('formatAuditAction', () => {
  it.each([
    ['admin.accounts.today_stats.batch.create', '批量创建账号今日统计'],
    ['usage.dashboard.api_keys_usage.create', '创建 API 密钥用量记录'],
    ['auth.send_verify_code.create', '发送验证码'],
  ])('translates known action %s', (action, label) => {
    expect(formatAuditAction(action)).toEqual({ label })
  })

  it('derives Chinese labels from known resources and verbs', () => {
    expect(formatAuditAction('admin.users.update')).toEqual({ label: '更新用户' })
    expect(formatAuditAction('admin.support.tickets.delete')).toEqual({ label: '删除客服工单' })
  })

  it.each([
    'admin.accounts.batch_delete.create',
    'admin.accounts.models.sync_upstream_preview.create',
    'admin.backups.s3_config.test.create',
    'admin.channel_monitor_v2.config.update',
    'admin.docs.revisions.review.create',
    'admin.openai.accounts.quota.refresh.create',
    'admin.payment.orders.refund.create',
    'admin.settings.rate_limit_429_cooldown.update',
    'admin.support.tickets.messages.create',
    'auth.oauth.pending.send_verify_code.create',
    'auth.passkey.login.begin.create',
    'checkin.luck.create',
    'custom.admin.support_contacts.qr.delete',
    'user.account_bindings.email.send_code.create',
  ])('keeps captured production action %s Chinese', (action) => {
    expect(formatAuditAction(action).label).not.toBe('未识别操作')
  })

  it.each(capturedProductionActions)('translates every captured production action %s', (action) => {
    expect(formatAuditAction(action).label).not.toBe('未识别操作')
  })

  it('uses a Chinese fallback and keeps unknown source code only in its tooltip', () => {
    expect(formatAuditAction('upstream.future.operation')).toEqual({
      label: '未识别操作',
      title: 'upstream.future.operation',
    })
  })
})
