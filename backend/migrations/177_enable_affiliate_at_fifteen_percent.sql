INSERT INTO settings (key, value, updated_at)
VALUES
    ('affiliate_enabled', 'true', NOW()),
    ('affiliate_rebate_rate', '15', NOW())
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value,
    updated_at = NOW();
