-- 140_add_scheduled_test_failure_actions.sql
-- Add scheduled test failure actions and richer execution result fields.

ALTER TABLE scheduled_test_plans
    ADD COLUMN IF NOT EXISTS delete_on_confirmed_401 BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS switch_group_from_id BIGINT NULL REFERENCES groups(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS switch_group_to_id BIGINT NULL REFERENCES groups(id) ON DELETE SET NULL;

ALTER TABLE scheduled_test_results
    ADD COLUMN IF NOT EXISTS http_status_code INT NULL,
    ADD COLUMN IF NOT EXISTS attempt_no SMALLINT NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS action_taken VARCHAR(64) NOT NULL DEFAULT '';
