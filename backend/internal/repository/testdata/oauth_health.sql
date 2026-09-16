CREATE TEMP TABLE accounts (
 id bigint PRIMARY KEY,name text,platform text,type text,concurrency integer,status text,schedulable boolean,
 rate_limit_reset_at timestamptz,overload_until timestamptz,temp_unschedulable_until timestamptz,
 parent_account_id bigint,extra jsonb,deleted_at timestamptz,updated_at timestamptz
) ON COMMIT DROP;
CREATE TEMP TABLE account_groups(account_id bigint,group_id bigint) ON COMMIT DROP;
CREATE TEMP TABLE scheduler_outbox(event_type text,account_id bigint,group_id bigint,payload jsonb) ON COMMIT DROP;
CREATE TEMP TABLE ops_error_logs (
 id bigint,request_id text,account_id bigint,created_at timestamptz,is_count_tokens boolean DEFAULT false,
 status_code integer,upstream_status_code integer,upstream_error_message text,upstream_error_detail text,upstream_errors jsonb
) ON COMMIT DROP;
CREATE TEMP TABLE usage_logs(id bigint,request_id text,account_id bigint,created_at timestamptz,output_tokens integer,image_count integer,first_token_ms integer) ON COMMIT DROP;
INSERT INTO accounts(id,name,platform,type,concurrency,status,schedulable,extra) VALUES
 (1,'fixture-one','openai','oauth',10,'active',true,'{"unrelated":"preserved"}'),
 (2,'fixture-two','openai','oauth',8,'active',true,'{}'),
 (3,'api-pool','openai','apikey',99,'active',true,'{}'),
 (4,'ungrouped','anthropic','oauth',5,'active',true,'{}');
INSERT INTO account_groups VALUES (1,15),(1,16),(2,16),(3,15);
INSERT INTO usage_logs VALUES
 (1,'r1',1,'2026-09-16T03:58:00Z',20,0,1200),
 (2,'r2',1,'2026-09-16T03:59:00Z',20,0,2200),
 (3,'old',1,'2026-09-16T02:00:00Z',20,0,9000);
INSERT INTO ops_error_logs(id,request_id,account_id,created_at,upstream_status_code,upstream_error_message,upstream_errors) VALUES
 (1,'r1',2,'2026-09-16T03:58:00Z',503,'server_is_overloaded',
 '[{"account_id":1,"upstream_status_code":429},{"account_id":1,"upstream_status_code":429},{"account_id":2,"upstream_status_code":503}]'),
 (2,'r1',1,'2026-09-16T03:58:00Z',429,'rate limit',NULL),
 (3,'quota',1,'2026-09-16T03:59:00Z',429,'usage_limit_reached',
 '[{"account_id":1,"upstream_status_code":429,"upstream_response_body":"usage_limit_reached"}]'),
 (4,'local-only',1,'2026-09-16T03:59:00Z',NULL,NULL,'[]'),
 (5,'malformed-array',1,'2026-09-16T03:59:00Z',NULL,NULL,'{}'),
 (6,'r3',1,'2026-09-16T03:56:00Z',503,'overloaded','[]');
UPDATE ops_error_logs SET status_code=503 WHERE id=4;
