-- Convert the platform from 1 CNY = 1 USD balance to the GJX-style
-- 1 CNY = 25 USD balance unit. Historical audit rows are deliberately kept
-- unchanged; only live monetary state that can still be spent or released is
-- converted.
DO $$
DECLARE
    current_multiplier NUMERIC;
    standard_channel_id BIGINT;
BEGIN
    IF EXISTS (SELECT 1 FROM settings WHERE key = '1to25_balance_cutover_v1') THEN
        RETURN;
    END IF;

    SELECT CASE
               WHEN value ~ '^1([.]0+)?$' THEN value::NUMERIC
               ELSE NULL
           END
      INTO current_multiplier
      FROM settings
     WHERE key = 'BALANCE_RECHARGE_MULTIPLIER'
     FOR UPDATE;

    -- A missing row uses the application's former 1x default. Any other
    -- explicit value indicates an unexpected unit and must stop the cutover.
    IF NOT FOUND THEN
        current_multiplier := 1;
    END IF;
    IF current_multiplier IS DISTINCT FROM 1 THEN
        RAISE EXCEPTION '1:25 balance conversion requires the current recharge multiplier to be 1';
    END IF;

    IF EXISTS (
        SELECT 1
          FROM payment_orders
         WHERE status IN ('PENDING', 'PAID', 'RECHARGING', 'REFUND_REQUESTED', 'REFUNDING', 'REFUND_PENDING')
    ) THEN
        RAISE EXCEPTION '1:25 balance conversion requires no unsettled payment orders';
    END IF;

    UPDATE users
       SET balance = ROUND(balance * 25, 8),
           frozen_balance = ROUND(frozen_balance * 25, 8),
           total_recharged = ROUND(total_recharged * 25, 8),
           balance_notify_threshold = CASE
               WHEN balance_notify_threshold IS NULL THEN NULL
               ELSE ROUND(balance_notify_threshold * 25, 8)
           END;

    UPDATE user_affiliates
       SET aff_quota = ROUND(aff_quota * 25, 8),
           aff_frozen_quota = ROUND(aff_frozen_quota * 25, 8),
           aff_history_quota = ROUND(aff_history_quota * 25, 8),
           updated_at = NOW();

    -- Frozen accrual rows will later be released into aff_quota, so pending
    -- amounts use the new unit. Settled affiliate ledger history stays intact.
    UPDATE user_affiliate_ledger
       SET amount = ROUND(amount * 25, 8),
           updated_at = NOW()
     WHERE action = 'accrue'
       AND frozen_until IS NOT NULL;

    UPDATE redeem_codes
       SET value = ROUND(value * 25, 8)
     WHERE type = 'balance'
       AND status = 'unused';

    UPDATE promo_codes
       SET bonus_amount = ROUND(bonus_amount * 25, 8),
           updated_at = NOW()
     WHERE status = 'active'
       AND (max_uses = 0 OR used_count < max_uses);

    UPDATE settings
       SET value = to_char(ROUND(value::NUMERIC * 25, 8), 'FM999999999999990.00000000'),
           updated_at = NOW()
     WHERE key = 'balance_low_notify_threshold'
       AND value ~ '^-?[0-9]+([.][0-9]+)?$';

    -- All configured signup sources share the same post-cutover grant. Sources
    -- with grant_on_signup disabled retain that flag and therefore do not issue
    -- the stored amount until explicitly enabled.
    INSERT INTO settings (key, value, updated_at)
    VALUES
        ('default_balance', '5.00000000', NOW()),
        ('auth_source_default_email_balance', '5.00000000', NOW()),
        ('auth_source_default_linuxdo_balance', '5.00000000', NOW()),
        ('auth_source_default_oidc_balance', '5.00000000', NOW()),
        ('auth_source_default_wechat_balance', '5.00000000', NOW()),
        ('auth_source_default_github_balance', '5.00000000', NOW()),
        ('auth_source_default_google_balance', '5.00000000', NOW()),
        ('auth_source_default_dingtalk_balance', '5.00000000', NOW()),
        ('BALANCE_RECHARGE_MULTIPLIER', '25.00000000', NOW())
    ON CONFLICT (key) DO UPDATE
       SET value = EXCLUDED.value,
           updated_at = NOW();

    UPDATE groups
       SET rate_multiplier = 1.0000,
           updated_at = NOW()
     WHERE name = 'plus-quarantine'
       AND deleted_at IS NULL;

    UPDATE groups
       SET rate_multiplier = 1.8800,
           updated_at = NOW()
     WHERE name = 'pro-quarantine'
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

        -- Price columns remain NULL so all clients use the single bundled and
        -- runtime-overridden price source. These rows are the paid allowlist.
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
        '1to25_balance_cutover_v1',
        json_build_object(
            'cutover_at', NOW(),
            'old_multiplier', 1,
            'new_multiplier', 25,
            'balance_multiplier', 25,
            'signup_bonus', 5
        )::TEXT,
        NOW()
    );
END
$$;
