-- Extend the upstream Channel Monitor V2 platform allow-list without
-- replacing operator-managed settings. Existing entries keep their enabled
-- state and model selection; only missing platforms are appended.
WITH desired(platform, ordinal) AS (
    VALUES
        ('kimi', 1),
        ('zhipu', 2),
        ('deepseek', 3)
),
missing AS (
    SELECT
        config.id,
        desired.ordinal,
        jsonb_build_object(
            'platform', desired.platform,
            'enabled', TRUE,
            'models', '[]'::jsonb
        ) AS item
    FROM channel_monitor_v2_config AS config
    CROSS JOIN desired
    WHERE config.id = 1
      AND NOT EXISTS (
          SELECT 1
          FROM jsonb_array_elements(config.platforms) AS existing(item)
          WHERE lower(btrim(existing.item ->> 'platform')) = desired.platform
      )
),
additions AS (
    SELECT id, jsonb_agg(item ORDER BY ordinal) AS items
    FROM missing
    GROUP BY id
)
UPDATE channel_monitor_v2_config AS config
SET
    platforms = config.platforms || additions.items,
    version = config.version + 1,
    updated_at = NOW()
FROM additions
WHERE config.id = additions.id;
