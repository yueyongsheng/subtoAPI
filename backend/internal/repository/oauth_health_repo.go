package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

func NewOAuthHealthRepository(client *dbent.Client, db *sql.DB, cache service.SchedulerCache) service.OAuthHealthRepository {
	return newAccountRepositoryWithSQL(client, db, cache)
}

const oauthHealthAccountSQL = `
SELECT a.id,a.name,a.platform,a.concurrency,a.status,a.schedulable,
 a.rate_limit_reset_at,a.overload_until,a.temp_unschedulable_until,a.parent_account_id,
 EXISTS(SELECT 1 FROM accounts child WHERE child.parent_account_id=a.id AND child.deleted_at IS NULL),
 COALESCE(a.extra->'oauth_health','null'::jsonb) raw_health,
 COALESCE(a.extra->>'oauth_health_manual_concurrency','') manual_concurrency_value,
 ARRAY(SELECT ag.group_id FROM account_groups ag WHERE ag.account_id=a.id ORDER BY ag.group_id)
FROM accounts a WHERE a.deleted_at IS NULL AND a.type='oauth'
 AND ($1::bigint IS NULL OR ($1=-1 AND NOT EXISTS(SELECT 1 FROM account_groups ag WHERE ag.account_id=a.id))
 OR EXISTS(SELECT 1 FROM account_groups ag WHERE ag.account_id=a.id AND ag.group_id=$1))
ORDER BY a.id LIMIT 2001`

func (r *accountRepository) ListOAuthHealthAccounts(ctx context.Context, groupID *int64) ([]service.OAuthHealthAccount, error) {
	rows, err := r.sql.QueryContext(ctx, oauthHealthAccountSQL, groupID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	result := []service.OAuthHealthAccount{}
	for rows.Next() {
		var a service.OAuthHealthAccount
		if err := rows.Scan(&a.ID, &a.Name, &a.Platform, &a.Concurrency, &a.Status, &a.Schedulable,
			&a.RateLimitResetAt, &a.OverloadUntil, &a.TempUnschedulableUntil, &a.ParentAccountID, &a.HasChildren,
			&a.RawHealth, &a.ManualConcurrencyValue, pq.Array(&a.GroupIDs)); err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	if len(result) > 2000 {
		return nil, errors.New("select a smaller OAuth account group (maximum 2000)")
	}
	return result, rows.Err()
}

// Correlate by account AND gateway request. Both the upstream-events array and
// terminal upstream error can describe the same attempt. Neither local 429/503
// nor the number of retries is an account failure count. Attributed attempts
// override terminal metadata even outside the selected scope; a final account
// may have succeeded after failover. Use attempt time and exclude monitor keys.
const oauthHealthObservationSQL = `
WITH scope AS MATERIALIZED (
 SELECT * FROM jsonb_to_recordset($1::jsonb) AS x(id bigint,since timestamptz,history_since timestamptz)
), logs AS MATERIALIZED (
 SELECT o.* FROM ops_error_logs o
 WHERE o.created_at >= (SELECT min(COALESCE(history_since,since)) FROM scope) AND o.created_at < $2
 AND o.is_count_tokens=FALSE
 AND NOT EXISTS(SELECT 1 FROM channel_monitors m WHERE m.probe_api_key_id=o.api_key_id)
), attempts AS MATERIALIZED (
 SELECT o.id log_id,e,
 CASE WHEN e->>'at_unix_ms' ~ '^[0-9]{1,13}$'
 THEN to_timestamp((e->>'at_unix_ms')::double precision/1000) ELSE o.created_at END at
 FROM logs o CROSS JOIN LATERAL jsonb_array_elements(
 CASE WHEN jsonb_typeof(o.upstream_errors)='array' THEN o.upstream_errors ELSE '[]'::jsonb END) e
 WHERE e->>'account_id' ~ '^[1-9][0-9]{0,17}$'
 AND e->>'upstream_status_code' ~ '^[45][0-9]{2}$'
), events AS (
 SELECT s.id,COALESCE(NULLIF(o.request_id,''),'error:'||o.id::text) req,
 o.model, a.at created_at, e->>'upstream_status_code' code,
 lower(COALESCE(e->>'message','')||' '||COALESCE(e->>'detail','')||' '||COALESCE(e->>'upstream_response_body','')) msg
 FROM logs o JOIN attempts a ON a.log_id=o.id
 JOIN scope s ON e->>'account_id'=s.id::text AND a.at>=COALESCE(s.history_since,s.since) AND a.at<$2
 UNION ALL
 SELECT s.id,COALESCE(NULLIF(o.request_id,''),'error:'||o.id::text),o.model,o.created_at,
 o.upstream_status_code::text,lower(COALESCE(o.upstream_error_message,'')||' '||COALESCE(o.upstream_error_detail,''))
 FROM logs o JOIN scope s ON o.account_id=s.id AND o.created_at>=COALESCE(s.history_since,s.since)
 WHERE o.upstream_status_code>=400
 AND NOT EXISTS(SELECT 1 FROM attempts a WHERE a.log_id=o.id)
), classified AS (
 SELECT *,code='429' AND msg ~ '(insufficient_quota|usage_limit_reached|quota_exceeded|quota exhausted|quota exceeded|weekly.{0,20}limit|daily.{0,20}limit)' quota
 FROM events WHERE code IN ('401','403','429','500','502','503','504')
), requests AS (
 SELECT id,req,max(created_at) at,
 bool_or(code='429' AND NOT quota) rate_limited,bool_or(quota) quota,
 bool_or(code='503') overloaded,bool_or(code IN ('401','403')) auth,
 bool_or(code IN ('500','502','504')) other_error
 FROM classified c WHERE c.created_at>=(SELECT s.since FROM scope s WHERE s.id=c.id) GROUP BY id,req
), usage AS MATERIALIZED (
 SELECT s.id,COALESCE(NULLIF(u.request_id,''),'usage:'||u.id::text) req,
 (u.output_tokens>0 OR u.image_count>0) has_output,u.first_token_ms,u.model,u.created_at
 FROM usage_logs u JOIN scope s ON s.id=u.account_id AND u.created_at>=s.since
 WHERE u.created_at >= (SELECT min(since) FROM scope) AND u.created_at < $2
 AND NOT EXISTS(SELECT 1 FROM channel_monitors m WHERE m.probe_api_key_id=u.api_key_id)
), observed AS (
 SELECT id,req FROM requests UNION SELECT id,req FROM usage
), counts AS (
 SELECT id,count(*) observed FROM observed GROUP BY id
), errors AS (
 SELECT id,count(*) FILTER(WHERE rate_limited) rate_limited,
 count(*) FILTER(WHERE quota) quota,count(*) FILTER(WHERE overloaded) overloaded,
 count(*) FILTER(WHERE auth) auth,count(*) FILTER(WHERE other_error) other_error,
 count(*) FILTER(WHERE rate_limited OR overloaded) pressure,
 count(DISTINCT date_trunc('minute',at)) FILTER(WHERE rate_limited OR overloaded) pressure_minutes,
 max(at) FILTER(WHERE rate_limited OR overloaded) latest_pressure_at
 FROM requests GROUP BY id
), outputs AS (
 SELECT id,count(DISTINCT req) FILTER(WHERE has_output) outputs,
 count(DISTINCT date_trunc('minute',created_at)) FILTER(WHERE has_output) output_minutes,
 percentile_cont(0.95) WITHIN GROUP(ORDER BY first_token_ms) FILTER(WHERE first_token_ms>=0) p95
 FROM usage GROUP BY id
), model_keys AS (
 SELECT id,COALESCE(model,'') model FROM usage UNION SELECT id,COALESCE(model,'') FROM classified
), model_stats AS (
 SELECT k.id,jsonb_agg(jsonb_build_object('model',k.model,
 'output_requests',(SELECT count(DISTINCT u.req) FROM usage u WHERE u.id=k.id AND COALESCE(u.model,'')=k.model AND u.has_output),
 'error_requests',(SELECT count(DISTINCT c.req) FROM classified c JOIN scope s ON s.id=c.id WHERE c.id=k.id AND COALESCE(c.model,'')=k.model AND c.created_at>=s.since),
 'had_error',EXISTS(SELECT 1 FROM classified c WHERE c.id=k.id AND COALESCE(c.model,'')=k.model)) ORDER BY k.model) models
 FROM model_keys k GROUP BY k.id
)
SELECT s.id,COALESCE(c.observed,0),COALESCE(u.outputs,0),COALESCE(e.rate_limited,0),COALESCE(e.quota,0),
 COALESCE(e.overloaded,0),COALESCE(e.auth,0),COALESCE(e.other_error,0),COALESCE(e.pressure,0),
 COALESCE(e.pressure_minutes,0),e.latest_pressure_at,u.p95,COALESCE(u.output_minutes,0),COALESCE(m.models,'[]'::jsonb)
FROM scope s LEFT JOIN counts c ON c.id=s.id LEFT JOIN errors e ON e.id=s.id LEFT JOIN outputs u ON u.id=s.id LEFT JOIN model_stats m ON m.id=s.id`

func (r *accountRepository) ObserveOAuthHealth(ctx context.Context, scope []service.OAuthHealthObservationScope, end time.Time) (map[int64]service.OAuthHealthStats, error) {
	result := make(map[int64]service.OAuthHealthStats, len(scope))
	if len(scope) == 0 {
		return result, nil
	}
	for i := range scope {
		if scope[i].HistorySince.IsZero() {
			scope[i].HistorySince = scope[i].Since
		}
	}
	data, err := json.Marshal(scope)
	if err != nil {
		return nil, err
	}
	rows, err := r.sql.QueryContext(ctx, oauthHealthObservationSQL, string(data), end)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id int64
		var s service.OAuthHealthStats
		var models []byte
		if err := rows.Scan(&id, &s.ObservedRequests, &s.OutputRequests, &s.RateLimitedRequests, &s.QuotaRequests,
			&s.OverloadedRequests, &s.AuthRequests, &s.OtherErrorRequests, &s.PressureRequests, &s.PressureMinutes,
			&s.LatestPressureAt, &s.P95FirstTokenMS, &s.OutputMinutes, &models); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(models, &s.Models); err != nil {
			return nil, err
		}
		result[id] = s
	}
	return result, rows.Err()
}

// CAS guards the report, concurrency and scheduling state. JSONB merge preserves
// tokens and unrelated extra fields. A concurrency change and its durable
// scheduler notification commit in the same statement.
const saveOAuthHealthSQL = `
WITH updated AS (
 UPDATE accounts a SET extra=jsonb_set(COALESCE(a.extra,'{}'::jsonb),'{oauth_health}',$1::jsonb,true),
 concurrency=$2,updated_at=NOW(),
 temp_unschedulable_until=CASE WHEN $16::timestamptz IS NULL THEN a.temp_unschedulable_until ELSE $16 END,
 temp_unschedulable_reason=CASE WHEN $16::timestamptz IS NULL THEN a.temp_unschedulable_reason ELSE 'oauth_health_cooldown' END
 WHERE a.id=$3 AND a.deleted_at IS NULL AND a.type='oauth' AND a.concurrency=$4
 AND COALESCE(a.extra->'oauth_health','null'::jsonb)=$5::jsonb
 AND a.status=$6 AND a.schedulable=$7
 AND a.rate_limit_reset_at IS NOT DISTINCT FROM $8::timestamptz
 AND a.overload_until IS NOT DISTINCT FROM $9::timestamptz
 AND a.temp_unschedulable_until IS NOT DISTINCT FROM $10::timestamptz
 AND ARRAY(SELECT ag.group_id FROM account_groups ag WHERE ag.account_id=a.id ORDER BY ag.group_id)=$11::bigint[]
 AND ($12::bigint IS NULL OR ($12=-1 AND NOT EXISTS(SELECT 1 FROM account_groups ag WHERE ag.account_id=a.id))
 OR EXISTS(SELECT 1 FROM account_groups ag WHERE ag.account_id=a.id AND ag.group_id=$12))
 AND a.parent_account_id IS NOT DISTINCT FROM $14::bigint
 AND EXISTS(SELECT 1 FROM accounts child WHERE child.parent_account_id=a.id AND child.deleted_at IS NULL)=$15
 AND COALESCE(a.extra->>'oauth_health_manual_concurrency','')=$17
 AND ($16::timestamptz IS NULL OR (
  cardinality($18::bigint[])>0 AND NOT EXISTS(
   SELECT 1 FROM unnest($11::bigint[]) g(group_id) CROSS JOIN jsonb_array_elements_text($1::jsonb->'cooldown_models') m(model)
   WHERE NOT EXISTS(SELECT 1 FROM accounts b JOIN account_groups bg ON bg.account_id=b.id
    WHERE b.id=ANY($18::bigint[]) AND bg.group_id=g.group_id AND b.deleted_at IS NULL AND b.status='active' AND b.schedulable
    AND b.platform=a.platform AND b.type='oauth' AND b.parent_account_id IS NULL
    AND NOT EXISTS(SELECT 1 FROM accounts child WHERE child.parent_account_id=b.id AND child.deleted_at IS NULL)
    AND (b.rate_limit_reset_at IS NULL OR b.rate_limit_reset_at<=NOW()) AND (b.overload_until IS NULL OR b.overload_until<=NOW())
    AND (b.temp_unschedulable_until IS NULL OR b.temp_unschedulable_until<=NOW())
    AND EXISTS(SELECT 1 FROM usage_logs u WHERE u.account_id=b.id AND u.model=m.model AND u.created_at>NOW()-interval '5 minutes' AND u.output_tokens>0
     AND NOT EXISTS(SELECT 1 FROM channel_monitors cm WHERE cm.probe_api_key_id=u.api_key_id)))
  )))
 RETURNING a.id
), notified AS (
 INSERT INTO scheduler_outbox(event_type,account_id,group_id,payload)
 SELECT $13,updated.id,NULL,NULL FROM updated WHERE $2<>$4 OR $16::timestamptz IS NOT NULL RETURNING account_id
)
SELECT id FROM updated`

func (r *accountRepository) SaveOAuthHealth(ctx context.Context, a service.OAuthHealthAccount, h *service.OAuthHealth, target int, groupID *int64) (bool, error) {
	data, err := json.Marshal(h)
	if err != nil {
		return false, err
	}
	raw := string(a.RawHealth)
	if raw == "" {
		raw = "null"
	}
	var cooldown *time.Time
	if h.LastChange != nil && h.LastChange.Action == "cooldown" && h.LastChange.At.Equal(h.CheckedAt) && h.CooldownUntil != nil && (a.TempUnschedulableUntil == nil || !a.TempUnschedulableUntil.Equal(*h.CooldownUntil)) {
		cooldown = h.CooldownUntil
	}
	rows, err := r.sql.QueryContext(ctx, saveOAuthHealthSQL, string(data), target, a.ID, a.Concurrency, raw, a.Status, a.Schedulable,
		a.RateLimitResetAt, a.OverloadUntil, a.TempUnschedulableUntil, pq.Array(a.GroupIDs), groupID, service.SchedulerOutboxEventAccountChanged, a.ParentAccountID, a.HasChildren, cooldown, a.ManualConcurrencyValue, pq.Array(h.CooldownAlternatives))
	if err != nil {
		return false, err
	}
	changed := false
	for rows.Next() {
		changed = true
	}
	rowErr := rows.Err()
	closeErr := rows.Close()
	if rowErr != nil {
		return false, rowErr
	}
	if closeErr != nil {
		return false, closeErr
	}
	if changed && (target != a.Concurrency || cooldown != nil) {
		r.syncSchedulerAccountSnapshot(ctx, a.ID)
	}
	return changed, nil
}

// Require recent successful alternatives for every group/model affected by the
// account cooldown. The write repeats eligibility checks to guard races.
func (r *accountRepository) OAuthHealthCooldownAlternatives(ctx context.Context, a service.OAuthHealthAccount, models []string, now time.Time, excluded []int64) ([]int64, error) {
	if len(a.GroupIDs) == 0 || len(models) == 0 {
		return nil, nil
	}
	excluded = append(append([]int64{}, excluded...), a.ID)
	rows, err := r.sql.QueryContext(ctx, oauthHealthCooldownAlternativesSQL, pq.Array(a.GroupIDs), pq.Array(models), a.Platform, pq.Array(excluded), now)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	ids := []int64{}
	for rows.Next() {
		var g int64
		var model string
		var matches pq.Int64Array
		if err := rows.Scan(&g, &model, &matches); err != nil {
			return nil, err
		}
		if len(matches) == 0 {
			return nil, nil
		}
		ids = append(ids, matches...)
	}
	return ids, rows.Err()
}

const oauthHealthCooldownAlternativesSQL = `SELECT g.group_id,m.model,COALESCE(array_agg(DISTINCT b.id) FILTER(WHERE b.id IS NOT NULL),'{}'::bigint[])
 FROM unnest($1::bigint[]) g(group_id) CROSS JOIN unnest($2::text[]) m(model)
 LEFT JOIN (accounts b JOIN account_groups bg ON bg.account_id=b.id) ON bg.group_id=g.group_id
 AND b.platform=$3 AND b.type='oauth' AND b.deleted_at IS NULL AND b.status='active' AND b.schedulable AND NOT(b.id=ANY($4::bigint[]))
 AND b.parent_account_id IS NULL
 AND NOT EXISTS(SELECT 1 FROM accounts child WHERE child.parent_account_id=b.id AND child.deleted_at IS NULL)
 AND (b.rate_limit_reset_at IS NULL OR b.rate_limit_reset_at<=$5) AND (b.overload_until IS NULL OR b.overload_until<=$5)
 AND (b.temp_unschedulable_until IS NULL OR b.temp_unschedulable_until<=$5)
 AND EXISTS(SELECT 1 FROM usage_logs u WHERE u.account_id=b.id AND u.model=m.model AND u.created_at>=$5::timestamptz-interval '5 minutes' AND u.created_at<$5 AND u.output_tokens>0
 AND NOT EXISTS(SELECT 1 FROM channel_monitors cm WHERE cm.probe_api_key_id=u.api_key_id))
 GROUP BY g.group_id,m.model`
