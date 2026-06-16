-- 清理已废弃的旧 bank / finance 体系与旧版娱乐大厅共享表。
-- 当前现役余额以 users.balance 和 game_hall_* 为准。

DROP TABLE IF EXISTS financial_reconciliation_issues;
DROP TABLE IF EXISTS financial_reconciliation_runs;
DROP TABLE IF EXISTS financial_reversals;
DROP TABLE IF EXISTS financial_risk_events;
DROP VIEW IF EXISTS ledger_transaction_balances;
DROP TABLE IF EXISTS ledger_entries CASCADE;
DROP TABLE IF EXISTS transactions_log CASCADE;
DROP TABLE IF EXISTS ledger_accounts CASCADE;
DROP TABLE IF EXISTS exchange_orders CASCADE;
DROP TABLE IF EXISTS p2p_settlement_items CASCADE;
DROP TABLE IF EXISTS p2p_settlement_runs CASCADE;
DROP TABLE IF EXISTS p2p_investment_positions CASCADE;
DROP TABLE IF EXISTS p2p_loan_requests CASCADE;
DROP TABLE IF EXISTS debt_transfer_trades CASCADE;
DROP TABLE IF EXISTS debt_market_listings CASCADE;
DROP TABLE IF EXISTS bad_debt_cases CASCADE;
DROP TABLE IF EXISTS loans_contract CASCADE;
DROP TABLE IF EXISTS financial_audit_logs CASCADE;
DROP TABLE IF EXISTS users_bank_account CASCADE;

DROP TABLE IF EXISTS game_wallet_transactions CASCADE;
DROP TABLE IF EXISTS game_jackpot_transactions CASCADE;
DROP TABLE IF EXISTS game_ledger CASCADE;
DROP TABLE IF EXISTS game_wallets CASCADE;
DROP TABLE IF EXISTS game_jackpots CASCADE;

DROP TABLE IF EXISTS deleted_api_key_audits CASCADE;

DROP SCHEMA IF EXISTS finance_v2_20260605095822 CASCADE;
DROP SCHEMA IF EXISTS finance_v2_20260605095901 CASCADE;
