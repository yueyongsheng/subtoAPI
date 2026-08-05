//go:build integration

package repository

import (
	"context"
	"database/sql"
	"strconv"
	"testing"

	projectmigrations "github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
)

func TestMigration182ConvertsOnlyLiveMonetaryStateAndIsIdempotent(t *testing.T) {
	ctx := context.Background()
	tx := testTx(t)

	_, err := tx.ExecContext(ctx, `
SET LOCAL search_path TO pg_temp, public;
CREATE TEMP TABLE settings (key TEXT PRIMARY KEY, value TEXT NOT NULL, updated_at TIMESTAMPTZ);
CREATE TEMP TABLE payment_orders (status TEXT NOT NULL);
CREATE TEMP TABLE users (
    balance NUMERIC NOT NULL,
    frozen_balance NUMERIC NOT NULL,
    total_recharged NUMERIC NOT NULL,
    balance_notify_threshold NUMERIC NULL
);
CREATE TEMP TABLE user_affiliates (
    aff_quota NUMERIC NOT NULL,
    aff_frozen_quota NUMERIC NOT NULL,
    aff_history_quota NUMERIC NOT NULL,
    updated_at TIMESTAMPTZ
);
CREATE TEMP TABLE user_affiliate_ledger (
    action TEXT NOT NULL,
    amount NUMERIC NOT NULL,
    frozen_until TIMESTAMPTZ NULL,
    updated_at TIMESTAMPTZ
);
CREATE TEMP TABLE redeem_codes (type TEXT NOT NULL, status TEXT NOT NULL, value NUMERIC NOT NULL);
CREATE TEMP TABLE promo_codes (
    status TEXT NOT NULL,
    max_uses INTEGER NOT NULL,
    used_count INTEGER NOT NULL,
    bonus_amount NUMERIC NOT NULL,
    updated_at TIMESTAMPTZ
);
CREATE TEMP TABLE groups (
    name TEXT NOT NULL,
    rate_multiplier NUMERIC NOT NULL,
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ NULL
);
CREATE TEMP TABLE channels (
    id BIGINT PRIMARY KEY,
    name TEXT NOT NULL,
    restrict_models BOOLEAN NOT NULL,
    billing_model_source TEXT NOT NULL,
    apply_pricing_to_account_stats BOOLEAN NOT NULL,
    updated_at TIMESTAMPTZ
);
CREATE TEMP TABLE channel_model_pricing (
    channel_id BIGINT NOT NULL,
    platform TEXT NOT NULL,
    models JSONB NOT NULL,
    billing_mode TEXT NOT NULL,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
);

INSERT INTO settings (key, value) VALUES
    ('BALANCE_RECHARGE_MULTIPLIER', '1.00000000'),
    ('1to1_balance_cutover_v1', '{"preserved":true}'),
    ('default_balance', '0.10000000'),
    ('auth_source_default_email_balance', '0.10000000'),
    ('balance_low_notify_threshold', '-0.60000000');
INSERT INTO users VALUES
    (30.08, 1.20, 90.47, -0.60),
    (-0.60, 0, 0, NULL);
INSERT INTO user_affiliates VALUES (2, 3, 4, NOW());
INSERT INTO user_affiliate_ledger VALUES
    ('accrue', 1.25, NOW() + INTERVAL '1 day', NOW()),
    ('accrue', 2.00, NULL, NOW()),
    ('transfer', 3.00, NULL, NOW());
INSERT INTO redeem_codes VALUES
    ('balance', 'unused', 8),
    ('balance', 'used', 9),
    ('subscription', 'unused', 10);
INSERT INTO promo_codes VALUES
    ('active', 0, 0, 6, NOW()),
    ('active', 1, 1, 7, NOW()),
    ('disabled', 0, 0, 8, NOW());
INSERT INTO groups VALUES
    ('plus-quarantine', 0.12, NOW(), NULL),
    ('pro-quarantine', 0.18, NOW(), NULL);
INSERT INTO channels VALUES (1, 'Standard Price', FALSE, 'default', TRUE, NOW());
`)
	require.NoError(t, err)

	migrationSQL, err := projectmigrations.FS.ReadFile("182_convert_balances_to_gjx_twenty_five.sql")
	require.NoError(t, err)
	require.NoError(t, execMigration182InTransaction(ctx, tx, string(migrationSQL)))
	// Re-running the SQL body must stop on the marker without multiplying again.
	require.NoError(t, execMigration182InTransaction(ctx, tx, string(migrationSQL)))

	assertTempNumericRows(t, tx, "SELECT balance, frozen_balance, total_recharged, balance_notify_threshold FROM users ORDER BY balance DESC", [][]any{
		{752.0, 30.0, 2261.75, -15.0},
		{-15.0, 0.0, 0.0, nil},
	})
	assertTempNumericRows(t, tx, "SELECT aff_quota, aff_frozen_quota, aff_history_quota FROM user_affiliates", [][]any{{50.0, 75.0, 100.0}})
	assertTempNumericRows(t, tx, "SELECT amount FROM user_affiliate_ledger ORDER BY amount", [][]any{{2.0}, {3.0}, {31.25}})
	assertTempNumericRows(t, tx, "SELECT value FROM redeem_codes ORDER BY value", [][]any{{9.0}, {10.0}, {200.0}})
	assertTempNumericRows(t, tx, "SELECT bonus_amount FROM promo_codes ORDER BY bonus_amount", [][]any{{7.0}, {8.0}, {150.0}})

	var multiplier, defaultBalance, emailBalance, lowThreshold, oldMarker, newMarker string
	require.NoError(t, tx.QueryRowContext(ctx, `
SELECT
    MAX(value) FILTER (WHERE key = 'BALANCE_RECHARGE_MULTIPLIER'),
    MAX(value) FILTER (WHERE key = 'default_balance'),
    MAX(value) FILTER (WHERE key = 'auth_source_default_email_balance'),
    MAX(value) FILTER (WHERE key = 'balance_low_notify_threshold'),
    MAX(value) FILTER (WHERE key = '1to1_balance_cutover_v1'),
    MAX(value) FILTER (WHERE key = '1to25_balance_cutover_v1')
FROM settings
`).Scan(&multiplier, &defaultBalance, &emailBalance, &lowThreshold, &oldMarker, &newMarker))
	require.Equal(t, "25.00000000", multiplier)
	require.Equal(t, "5.00000000", defaultBalance)
	require.Equal(t, "5.00000000", emailBalance)
	require.Equal(t, "-15.00000000", lowThreshold)
	require.Equal(t, `{"preserved":true}`, oldMarker)
	require.Contains(t, newMarker, `"balance_multiplier" : 25`)

	assertTempNumericRows(t, tx, "SELECT rate_multiplier FROM groups ORDER BY name", [][]any{{1.0}, {1.88}})
	var pricingRows int
	require.NoError(t, tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM channel_model_pricing").Scan(&pricingRows))
	require.Equal(t, 3, pricingRows)
}

func execMigration182InTransaction(ctx context.Context, tx *sql.Tx, migrationSQL string) error {
	_, err := tx.ExecContext(ctx, migrationSQL)
	return err
}

func assertTempNumericRows(t *testing.T, tx *sql.Tx, query string, expected [][]any) {
	t.Helper()
	rows, err := tx.QueryContext(context.Background(), query)
	require.NoError(t, err)
	defer rows.Close()

	columns, err := rows.Columns()
	require.NoError(t, err)
	actual := make([][]any, 0, len(expected))
	for rows.Next() {
		raw := make([]any, len(columns))
		dest := make([]any, len(columns))
		for i := range raw {
			dest[i] = &raw[i]
		}
		require.NoError(t, rows.Scan(dest...))
		converted := make([]any, len(raw))
		for i, value := range raw {
			if value == nil {
				converted[i] = nil
				continue
			}
			parsed, err := strconv.ParseFloat(string(value.([]byte)), 64)
			require.NoError(t, err)
			converted[i] = parsed
		}
		actual = append(actual, converted)
	}
	require.NoError(t, rows.Err())
	require.Equal(t, expected, actual)
}
