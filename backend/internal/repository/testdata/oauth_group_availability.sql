CREATE TEMP TABLE groups(id bigint PRIMARY KEY,name text,status text,sort_order integer,deleted_at timestamptz) ON COMMIT DROP;
CREATE TEMP TABLE accounts (
 id bigint PRIMARY KEY,type text,status text,schedulable boolean,expires_at timestamptz,auto_pause_on_expired boolean DEFAULT true,
 rate_limit_reset_at timestamptz,overload_until timestamptz,temp_unschedulable_until timestamptz,parent_account_id bigint,deleted_at timestamptz,extra jsonb
) ON COMMIT DROP;
CREATE TEMP TABLE account_groups(account_id bigint,group_id bigint) ON COMMIT DROP;
INSERT INTO groups VALUES (10,'Enterprise Pro','active',2,NULL),(20,'Shared','active',1,NULL),(30,'Unavailable','active',3,NULL),
 (40,'Disabled','disabled',4,NULL),(50,'API only','active',5,NULL),(60,'Deleted','active',6,'2026-09-16T00:00:00Z');
INSERT INTO accounts(id,type,status,schedulable) SELECT id,'oauth','active',true FROM generate_series(1,20) id;
UPDATE accounts SET extra='{"oauth_health":{"status":"overloaded"}}' WHERE id=1;
UPDATE accounts SET schedulable=false WHERE id=2;
UPDATE accounts SET status='error' WHERE id=3;
UPDATE accounts SET temp_unschedulable_until='2026-09-17T01:00:00Z' WHERE id IN (4,17);
UPDATE accounts SET rate_limit_reset_at='2026-09-17T01:00:00Z' WHERE id=5;
UPDATE accounts SET overload_until='2026-09-17T01:00:00Z' WHERE id=6;
UPDATE accounts SET expires_at='2026-09-16T23:00:00Z' WHERE id IN (7,8);
UPDATE accounts SET auto_pause_on_expired=false WHERE id=8;
UPDATE accounts SET type='apikey' WHERE id=9;
UPDATE accounts SET type='setup-token' WHERE id=10;
UPDATE accounts SET deleted_at='2026-09-16T23:00:00Z' WHERE id=11;
UPDATE accounts SET temp_unschedulable_until='2026-09-17T00:00:00Z' WHERE id=13;
UPDATE accounts SET rate_limit_reset_at='2026-09-17T00:00:00Z' WHERE id=14;
UPDATE accounts SET overload_until='2026-09-17T00:00:00Z' WHERE id=15;
UPDATE accounts SET parent_account_id=3 WHERE id=18;
UPDATE accounts SET parent_account_id=5 WHERE id=19;
UPDATE accounts SET expires_at='2026-09-17T00:00:00Z' WHERE id=20;
INSERT INTO account_groups SELECT id,10 FROM accounts WHERE id NOT IN (12,16,17);
INSERT INTO account_groups VALUES(1,10),(1,20),(17,20),(17,30),(16,40),(9,50),(1,60);
