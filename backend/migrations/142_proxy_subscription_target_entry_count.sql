ALTER TABLE proxy_subscription_sources
    ADD COLUMN IF NOT EXISTS target_entry_count INTEGER NOT NULL DEFAULT 1;
