INSERT INTO settings (key, value, updated_at)
VALUES ('affiliate_rebate_rate', '5', NOW())
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value,
    updated_at = NOW()
WHERE settings.value ~ '^10([.]0+)?$';
