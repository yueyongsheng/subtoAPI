-- Convert the production platform from 1 CNY = 25 USD to 1 CNY = 1 USD.
--
-- This migration is deliberately guarded by the legacy multiplier. Other
-- installations that did not use the 25x balance unit are left untouched.
DO $$
DECLARE
    legacy_multiplier NUMERIC;
    standard_channel_id BIGINT;
BEGIN
    IF EXISTS (SELECT 1 FROM settings WHERE key = '1to1_balance_cutover_v1') THEN
        RETURN;
    END IF;

    SELECT CASE
               WHEN value ~ '^25([.]0+)?$' THEN value::NUMERIC
               ELSE NULL
           END
      INTO legacy_multiplier
      FROM settings
     WHERE key = 'BALANCE_RECHARGE_MULTIPLIER'
     FOR UPDATE;

    IF legacy_multiplier IS DISTINCT FROM 25 THEN
        RETURN;
    END IF;

    IF EXISTS (
        SELECT 1
          FROM payment_orders
         WHERE status IN ('PENDING', 'PAID', 'RECHARGING', 'REFUND_REQUESTED', 'REFUNDING', 'REFUND_PENDING')
    ) THEN
        RAISE EXCEPTION '1:1 balance conversion requires no unsettled payment orders';
    END IF;

    UPDATE users
       SET balance = ROUND(balance / 25, 8),
           frozen_balance = ROUND(frozen_balance / 25, 8),
           total_recharged = ROUND(total_recharged / 25, 8),
           balance_notify_threshold = CASE
               WHEN balance_notify_threshold IS NULL THEN NULL
               ELSE ROUND(balance_notify_threshold / 25, 8)
           END;

    UPDATE user_affiliates
       SET aff_quota = ROUND(aff_quota / 25, 8),
           aff_frozen_quota = ROUND(aff_frozen_quota / 25, 8),
           aff_history_quota = ROUND(aff_history_quota / 25, 8),
           updated_at = NOW();

    -- Frozen accrual rows will later be released into aff_quota, so their
    -- pending amounts must use the new unit too. Settled ledger history stays unchanged.
    UPDATE user_affiliate_ledger
       SET amount = ROUND(amount / 25, 8),
           updated_at = NOW()
     WHERE action = 'accrue'
       AND frozen_until IS NOT NULL;

    UPDATE redeem_codes
       SET value = ROUND(value / 25, 8)
     WHERE type = 'balance'
       AND status = 'unused';

    UPDATE promo_codes
       SET bonus_amount = ROUND(bonus_amount / 25, 8),
           updated_at = NOW()
     WHERE status = 'active'
       AND (max_uses = 0 OR used_count < max_uses);

    UPDATE settings
       SET value = to_char(ROUND(value::NUMERIC / 25, 8), 'FM999999999999990.00000000'),
           updated_at = NOW()
     WHERE key IN (
         'auth_source_default_email_balance',
         'auth_source_default_linuxdo_balance',
         'auth_source_default_oidc_balance',
         'auth_source_default_wechat_balance',
         'auth_source_default_github_balance',
         'auth_source_default_google_balance',
         'auth_source_default_dingtalk_balance',
         'balance_low_notify_threshold'
     )
       AND value ~ '^-?[0-9]+([.][0-9]+)?$';

    INSERT INTO settings (key, value, updated_at)
    VALUES
        ('default_balance', '0.10000000', NOW()),
        ('BALANCE_RECHARGE_MULTIPLIER', '1.00000000', NOW())
    ON CONFLICT (key) DO UPDATE
       SET value = EXCLUDED.value,
           updated_at = NOW();

    UPDATE groups
       SET rate_multiplier = 1.4000,
           updated_at = NOW()
     WHERE name = 'plus-quarantine'
       AND deleted_at IS NULL;

    UPDATE groups
       SET rate_multiplier = 1.8800,
           updated_at = NOW()
     WHERE name = 'pro-quarantine'
       AND deleted_at IS NULL;

    UPDATE accounts
       SET rate_multiplier = 0.1000,
           updated_at = NOW()
     WHERE id IN (22, 23, 24, 25)
       AND deleted_at IS NULL;

    UPDATE accounts
       SET rate_multiplier = 0.1000,
           updated_at = NOW()
     WHERE id = 13
       AND deleted_at IS NULL;

    UPDATE accounts
       SET rate_multiplier = 0.1500,
           updated_at = NOW()
     WHERE id = 14
       AND deleted_at IS NULL;

    UPDATE accounts
       SET rate_multiplier = 0.0700,
           updated_at = NOW()
     WHERE id = 26
       AND deleted_at IS NULL;

    SELECT id
      INTO standard_channel_id
      FROM channels
     WHERE name = 'Standard Price'
     ORDER BY id
     LIMIT 1;

    IF standard_channel_id IS NOT NULL THEN
        UPDATE channels
           SET restrict_models = TRUE,
               billing_model_source = 'channel_mapped',
               apply_pricing_to_account_stats = FALSE,
               updated_at = NOW()
         WHERE id = standard_channel_id;

        DELETE FROM channel_model_pricing
         WHERE channel_id = standard_channel_id
           AND platform = 'openai';

        -- Keep token price columns NULL so the bundled official price source
        -- supplies standard/priority, cache and long-context prices. These
        -- exact rows are the paid-model allowlist, not flat price overrides.
        INSERT INTO channel_model_pricing (
            channel_id, platform, models, billing_mode, created_at, updated_at
        ) VALUES
            (
                standard_channel_id, 'openai',
                '["gpt-5.6-sol","gpt-5.5","codex-auto-review"]'::JSONB, 'token',
                NOW(), NOW()
            ),
            (
                standard_channel_id, 'openai',
                '["gpt-5.6-terra","gpt-5.4"]'::JSONB, 'token',
                NOW(), NOW()
            ),
            (
                standard_channel_id, 'openai',
                '["gpt-5.6-luna"]'::JSONB, 'token',
                NOW(), NOW()
            );
    END IF;

    INSERT INTO settings (key, value, updated_at)
    VALUES (
        '1to1_balance_cutover_v1',
        json_build_object(
            'cutover_at', NOW(),
            'old_multiplier', 25,
            'new_multiplier', 1,
            'balance_divisor', 25
        )::TEXT,
        NOW()
    );
END
$$;
