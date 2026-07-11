-- Add user-facing entry and leaderboard switches without changing current behavior.
INSERT INTO settings (key, value, updated_at) VALUES
    ('usage_query_enabled', 'true', NOW()),
    ('leaderboard_enabled', 'true', NOW()),
    ('leaderboard_balance_enabled', 'true', NOW()),
    ('leaderboard_consumption_enabled', 'true', NOW()),
    ('leaderboard_checkin_enabled', 'true', NOW()),
    ('leaderboard_include_admin', 'false', NOW())
ON CONFLICT (key) DO NOTHING;
