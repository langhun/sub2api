-- 为当前 public schema 下的现存业务表补齐中文表注释。
-- 注意：历史 migration 已受 checksum 保护，本文件仅追加，不回改旧迁移。

DO $$
DECLARE
    item RECORD;
BEGIN
    FOR item IN
        SELECT * FROM (VALUES
        ('schema_migrations', 'SQL 迁移执行记录表，记录已应用迁移文件、校验和与执行时间'),
        ('atlas_schema_revisions', 'Atlas 迁移版本记录表，用于兼容 Atlas 基线与版本状态管理'),
        ('proxies', '代理服务器配置表'),
        ('groups', '分组配置表，用于账号调度、计费与能力控制'),
        ('users', '用户账户表'),
        ('accounts', '上游账号表，保存第三方平台账号、凭据摘要与调度状态'),
        ('api_keys', '用户 API Key 表，用于访问网关'),
        ('account_groups', '账号-分组关联表'),
        ('redeem_codes', '兑换码表'),
        ('usage_logs', '用量明细日志表'),
        ('user_subscriptions', '用户订阅记录表'),
        ('settings', '系统设置表'),
        ('user_allowed_groups', '用户可用分组授权表'),
        ('orphan_allowed_groups_audit', '孤儿分组授权审计表，记录历史脏数据'),
        ('user_attribute_definitions', '用户属性定义表'),
        ('user_attribute_values', '用户属性取值表'),
        ('billing_usage_entries', '用量计费条目表，用于计费一致性核对与补偿'),
        ('promo_codes', '注册优惠码表'),
        ('promo_code_usages', '优惠码使用记录表'),
        ('ops_error_logs', 'Ops 错误日志表'),
        ('ops_retry_attempts', 'Ops 重试审计表'),
        ('ops_system_metrics', 'Ops 系统与请求指标快照表'),
        ('ops_job_heartbeats', 'Ops 后台任务心跳表'),
        ('ops_alert_rules', 'Ops 告警规则表'),
        ('ops_alert_events', 'Ops 告警事件表'),
        ('ops_metrics_hourly', 'Ops 每小时聚合指标表'),
        ('ops_metrics_daily', 'Ops 每日聚合指标表'),
        ('usage_dashboard_hourly', '用量仪表盘每小时聚合表'),
        ('usage_dashboard_daily', '用量仪表盘每日聚合表'),
        ('usage_dashboard_hourly_users', '用量仪表盘每小时活跃用户聚合表'),
        ('usage_dashboard_daily_users', '用量仪表盘每日活跃用户聚合表'),
        ('usage_dashboard_aggregation_watermark', '用量仪表盘聚合进度水位表'),
        ('scheduler_outbox', '调度器出站任务表'),
        ('ops_alert_silences', 'Ops 告警静默规则表'),
        ('usage_cleanup_tasks', '用量清理任务表'),
        ('announcements', '系统公告表'),
        ('announcement_reads', '公告已读记录表'),
        ('user_group_rate_multipliers', '用户-分组专属倍率与 RPM 覆写表'),
        ('error_passthrough_rules', '错误透传规则表'),
        ('security_secrets', '安全密钥与敏感配置存储表'),
        ('ops_system_logs', 'Ops 系统日志表'),
        ('ops_system_log_cleanup_audits', 'Ops 系统日志清理审计表'),
        ('idempotency_records', '幂等请求记录表'),
        ('usage_billing_dedup', '用量计费去重表'),
        ('usage_billing_dedup_archive', '用量计费去重归档表'),
        ('tls_fingerprint_profiles', 'TLS 指纹配置模板表'),
        ('channels', '渠道管理表，用于聚合分组与自定义定价'),
        ('channel_groups', '渠道-分组关联表'),
        ('channel_model_pricing', '渠道模型定价表'),
        ('channel_pricing_intervals', '渠道分层定价区间表'),
        ('payment_orders', '支付订单表'),
        ('payment_audit_logs', '支付审计日志表'),
        ('subscription_plans', '订阅套餐表'),
        ('payment_provider_instances', '支付提供方实例配置表'),
        ('channel_account_stats_pricing_rules', '渠道账号统计定价规则表'),
        ('channel_account_stats_model_pricing', '渠道账号统计模型定价表'),
        ('channel_account_stats_pricing_intervals', '渠道账号统计分层定价区间表'),
        ('checkins', '用户签到记录表'),
        ('auth_identities', '外部认证身份表，保存 OAuth、OIDC、微信等身份主体'),
        ('auth_identity_channels', '外部身份可用渠道表'),
        ('pending_auth_sessions', '待完成认证会话表'),
        ('identity_adoption_decisions', '外部身份归并决策表'),
        ('auth_identity_migration_reports', '外部身份迁移报告表，记录兼容迁移与安全检查结果'),
        ('checkin_prize_items', '签到盲盒奖项配置表'),
        ('checkin_blindbox_records', '签到盲盒开奖记录表'),
        ('user_provider_default_grants', '用户默认授权来源配置表'),
        ('user_avatars', '用户头像缓存表'),
        ('channel_monitors', '渠道可用性监控任务表'),
        ('channel_monitor_histories', '渠道监控历史记录表'),
        ('channel_monitor_daily_rollups', '渠道监控日聚合统计表'),
        ('channel_monitor_aggregation_watermark', '渠道监控聚合进度水位表'),
        ('channel_monitor_request_templates', '渠道监控请求模板表'),
        ('user_affiliates', '用户邀请返利信息表'),
        ('model_pricings', '模型标准定价表'),
        ('user_affiliate_ledger', '邀请返利资金流水表'),
        ('balance_transfers', '用户余额转账记录表'),
        ('balance_redpackets', '余额红包表'),
        ('balance_redpacket_claims', '余额红包领取记录表'),
        ('content_moderation_logs', '内容审核日志表'),
        ('scheduled_test_plans', '计划测试任务表'),
        ('scheduled_test_results', '计划测试结果表')
        ) AS comments(table_name, comment_text)
    LOOP
        IF to_regclass(format('public.%I', item.table_name)) IS NOT NULL THEN
            EXECUTE format(
                'COMMENT ON TABLE public.%I IS %L',
                item.table_name,
                item.comment_text
            );
        END IF;
    END LOOP;
END $$;

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
