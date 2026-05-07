-- 为当前 public schema 下的现存业务表补齐中文表注释。
-- 注意：历史 migration 已受 checksum 保护，本文件仅追加，不回改旧迁移。

COMMENT ON TABLE schema_migrations IS 'SQL 迁移执行记录表，记录已应用迁移文件、校验和与执行时间';
COMMENT ON TABLE atlas_schema_revisions IS 'Atlas 迁移版本记录表，用于兼容 Atlas 基线与版本状态管理';

COMMENT ON TABLE proxies IS '代理服务器配置表';
COMMENT ON TABLE groups IS '分组配置表，用于账号调度、计费与能力控制';
COMMENT ON TABLE users IS '用户账户表';
COMMENT ON TABLE accounts IS '上游账号表，保存第三方平台账号、凭据摘要与调度状态';
COMMENT ON TABLE api_keys IS '用户 API Key 表，用于访问网关';
COMMENT ON TABLE account_groups IS '账号-分组关联表';
COMMENT ON TABLE redeem_codes IS '兑换码表';
COMMENT ON TABLE usage_logs IS '用量明细日志表';
COMMENT ON TABLE user_subscriptions IS '用户订阅记录表';
COMMENT ON TABLE settings IS '系统设置表';
COMMENT ON TABLE user_allowed_groups IS '用户可用分组授权表';
COMMENT ON TABLE orphan_allowed_groups_audit IS '孤儿分组授权审计表，记录历史脏数据';
COMMENT ON TABLE user_attribute_definitions IS '用户属性定义表';
COMMENT ON TABLE user_attribute_values IS '用户属性取值表';
COMMENT ON TABLE billing_usage_entries IS '用量计费条目表，用于计费一致性核对与补偿';

COMMENT ON TABLE promo_codes IS '注册优惠码表';
COMMENT ON TABLE promo_code_usages IS '优惠码使用记录表';

COMMENT ON TABLE ops_error_logs IS 'Ops 错误日志表';
COMMENT ON TABLE ops_retry_attempts IS 'Ops 重试审计表';
COMMENT ON TABLE ops_system_metrics IS 'Ops 系统与请求指标快照表';
COMMENT ON TABLE ops_job_heartbeats IS 'Ops 后台任务心跳表';
COMMENT ON TABLE ops_alert_rules IS 'Ops 告警规则表';
COMMENT ON TABLE ops_alert_events IS 'Ops 告警事件表';
COMMENT ON TABLE ops_metrics_hourly IS 'Ops 每小时聚合指标表';
COMMENT ON TABLE ops_metrics_daily IS 'Ops 每日聚合指标表';
COMMENT ON TABLE usage_dashboard_hourly IS '用量仪表盘每小时聚合表';
COMMENT ON TABLE usage_dashboard_daily IS '用量仪表盘每日聚合表';
COMMENT ON TABLE usage_dashboard_hourly_users IS '用量仪表盘每小时活跃用户聚合表';
COMMENT ON TABLE usage_dashboard_daily_users IS '用量仪表盘每日活跃用户聚合表';
COMMENT ON TABLE usage_dashboard_aggregation_watermark IS '用量仪表盘聚合进度水位表';
COMMENT ON TABLE scheduler_outbox IS '调度器出站任务表';
COMMENT ON TABLE ops_alert_silences IS 'Ops 告警静默规则表';
COMMENT ON TABLE usage_cleanup_tasks IS '用量清理任务表';

COMMENT ON TABLE announcements IS '系统公告表';
COMMENT ON TABLE announcement_reads IS '公告已读记录表';
COMMENT ON TABLE user_group_rate_multipliers IS '用户-分组专属倍率与 RPM 覆写表';
COMMENT ON TABLE error_passthrough_rules IS '错误透传规则表';
COMMENT ON TABLE security_secrets IS '安全密钥与敏感配置存储表';
COMMENT ON TABLE ops_system_logs IS 'Ops 系统日志表';
COMMENT ON TABLE ops_system_log_cleanup_audits IS 'Ops 系统日志清理审计表';
COMMENT ON TABLE idempotency_records IS '幂等请求记录表';
COMMENT ON TABLE usage_billing_dedup IS '用量计费去重表';
COMMENT ON TABLE usage_billing_dedup_archive IS '用量计费去重归档表';
COMMENT ON TABLE tls_fingerprint_profiles IS 'TLS 指纹配置模板表';

COMMENT ON TABLE channels IS '渠道管理表，用于聚合分组与自定义定价';
COMMENT ON TABLE channel_groups IS '渠道-分组关联表';
COMMENT ON TABLE channel_model_pricing IS '渠道模型定价表';
COMMENT ON TABLE channel_pricing_intervals IS '渠道分层定价区间表';
COMMENT ON TABLE payment_orders IS '支付订单表';
COMMENT ON TABLE payment_audit_logs IS '支付审计日志表';
COMMENT ON TABLE subscription_plans IS '订阅套餐表';
COMMENT ON TABLE payment_provider_instances IS '支付提供方实例配置表';
COMMENT ON TABLE channel_account_stats_pricing_rules IS '渠道账号统计定价规则表';
COMMENT ON TABLE channel_account_stats_model_pricing IS '渠道账号统计模型定价表';
COMMENT ON TABLE channel_account_stats_pricing_intervals IS '渠道账号统计分层定价区间表';

COMMENT ON TABLE checkins IS '用户签到记录表';
COMMENT ON TABLE auth_identities IS '外部认证身份表，保存 OAuth、OIDC、微信等身份主体';
COMMENT ON TABLE auth_identity_channels IS '外部身份可用渠道表';
COMMENT ON TABLE pending_auth_sessions IS '待完成认证会话表';
COMMENT ON TABLE identity_adoption_decisions IS '外部身份归并决策表';
COMMENT ON TABLE auth_identity_migration_reports IS '外部身份迁移报告表，记录兼容迁移与安全检查结果';
COMMENT ON TABLE checkin_prize_items IS '签到盲盒奖项配置表';
COMMENT ON TABLE checkin_blindbox_records IS '签到盲盒开奖记录表';
COMMENT ON TABLE user_provider_default_grants IS '用户默认授权来源配置表';
COMMENT ON TABLE user_avatars IS '用户头像缓存表';
COMMENT ON TABLE channel_monitors IS '渠道可用性监控任务表';
COMMENT ON TABLE channel_monitor_histories IS '渠道监控历史记录表';
COMMENT ON TABLE channel_monitor_daily_rollups IS '渠道监控日聚合统计表';
COMMENT ON TABLE channel_monitor_aggregation_watermark IS '渠道监控聚合进度水位表';
COMMENT ON TABLE channel_monitor_request_templates IS '渠道监控请求模板表';
COMMENT ON TABLE user_affiliates IS '用户邀请返利信息表';
COMMENT ON TABLE model_pricings IS '模型标准定价表';
COMMENT ON TABLE user_affiliate_ledger IS '邀请返利资金流水表';
COMMENT ON TABLE balance_transfers IS '用户余额转账记录表';
COMMENT ON TABLE balance_redpackets IS '余额红包表';
COMMENT ON TABLE balance_redpacket_claims IS '余额红包领取记录表';
COMMENT ON TABLE content_moderation_logs IS '内容审核日志表';

COMMENT ON TABLE scheduled_test_plans IS '计划测试任务表';
COMMENT ON TABLE scheduled_test_results IS '计划测试结果表';

DO $$
DECLARE
    partition_name TEXT;
BEGIN
    FOR partition_name IN
        SELECT child.relname
        FROM pg_inherits i
        JOIN pg_class parent ON parent.oid = i.inhparent
        JOIN pg_class child ON child.oid = i.inhrelid
        JOIN pg_namespace ns ON ns.oid = child.relnamespace
        WHERE ns.nspname = 'public'
          AND parent.relname = 'usage_logs'
    LOOP
        EXECUTE format(
            'COMMENT ON TABLE %I IS %L',
            partition_name,
            '用量日志月分区表（按 UTC 月份切分）'
        );
    END LOOP;
END $$;
