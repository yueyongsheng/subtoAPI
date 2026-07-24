INSERT INTO settings (key, value, updated_at)
VALUES ('affiliate_rebate_rate', '10', NOW())
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value,
    updated_at = NOW()
WHERE settings.value IN ('15', '15.00000000');
