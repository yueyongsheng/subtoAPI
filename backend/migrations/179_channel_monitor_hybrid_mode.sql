-- Add group-scoped hybrid channel monitoring. Existing monitors remain active.

ALTER TABLE channel_monitors
    ADD COLUMN IF NOT EXISTS mode VARCHAR(16) NOT NULL DEFAULT 'active',
    ADD COLUMN IF NOT EXISTS group_id BIGINT,
    ADD COLUMN IF NOT EXISTS probe_api_key_id BIGINT,
    ADD COLUMN IF NOT EXISTS last_ping_latency_ms INT,
    ADD COLUMN IF NOT EXISTS last_ping_at TIMESTAMPTZ;

ALTER TABLE channel_monitors
    DROP CONSTRAINT IF EXISTS channel_monitors_mode_check;
ALTER TABLE channel_monitors
    ADD CONSTRAINT channel_monitors_mode_check CHECK (mode IN ('active', 'hybrid'));

ALTER TABLE channel_monitors
    DROP CONSTRAINT IF EXISTS channel_monitors_group_id_fkey;
ALTER TABLE channel_monitors
    ADD CONSTRAINT channel_monitors_group_id_fkey
        FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE SET NULL;

ALTER TABLE channel_monitors
    DROP CONSTRAINT IF EXISTS channel_monitors_probe_api_key_id_fkey;
ALTER TABLE channel_monitors
    ADD CONSTRAINT channel_monitors_probe_api_key_id_fkey
        FOREIGN KEY (probe_api_key_id) REFERENCES api_keys(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_channel_monitors_mode_group
    ON channel_monitors (mode, group_id) WHERE enabled = TRUE;
CREATE INDEX IF NOT EXISTS idx_channel_monitors_probe_api_key
    ON channel_monitors (probe_api_key_id) WHERE probe_api_key_id IS NOT NULL;

ALTER TABLE channel_monitor_histories
    ADD COLUMN IF NOT EXISTS source VARCHAR(20) NOT NULL DEFAULT 'active_probe',
    ADD COLUMN IF NOT EXISTS bucket_start TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS sample_count INT NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS success_count INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS failure_count INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS recovered_error_count INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS slow_count INT NOT NULL DEFAULT 0;

ALTER TABLE channel_monitor_histories
    DROP CONSTRAINT IF EXISTS channel_monitor_histories_source_check;
ALTER TABLE channel_monitor_histories
    ADD CONSTRAINT channel_monitor_histories_source_check
        CHECK (source IN ('active_probe', 'real_traffic'));

ALTER TABLE channel_monitor_histories
    DROP CONSTRAINT IF EXISTS channel_monitor_histories_counts_check;
ALTER TABLE channel_monitor_histories
    ADD CONSTRAINT channel_monitor_histories_counts_check CHECK (
        sample_count >= 0 AND success_count >= 0 AND failure_count >= 0
        AND recovered_error_count >= 0 AND slow_count >= 0
    );

CREATE UNIQUE INDEX IF NOT EXISTS idx_channel_monitor_histories_source_bucket
    ON channel_monitor_histories (monitor_id, model, source, bucket_start);
