import { apiClient } from '../client'

export interface OAuthHealthChange {
  at: string
  before: number
  after: number
  action: 'reduce' | 'restore' | 'increase' | 'rollback' | 'cooldown'
  actor_id: number
}
export interface OAuthHealth {
  policy_version?: number
  action?: 'reduce' | 'increase' | 'rollback' | 'cooldown' | ''
  observation_started_at?: string
  hold_until?: string
  cooldown_until?: string
  manual_concurrency?: boolean
  required_models?: string[]
  account_id: number
  checked_at: string
  window_start: string
  window_end: string
  status: string
  reason: string
  concurrency: number
  recommended_concurrency: number
  stats: {
    current_concurrency?: number
    output_minutes?: number
    models?: { model: string; output_requests: number; error_requests: number; had_error: boolean }[]
    observed_requests: number
    output_requests: number
    rate_limited_requests: number
    quota_requests: number
    overloaded_requests: number
    auth_requests: number
    other_error_requests: number
    pressure_requests: number
    pressure_minutes: number
    latest_pressure_at?: string
    p95_first_token_ms?: number
  }
  last_change?: OAuthHealthChange
  history?: OAuthHealthChange[]
}
export interface OAuthHealthAccount {
  id: number
  name: string
  platform: string
  group_ids: number[]
  concurrency: number
  schedulable: boolean
  health: OAuthHealth | null
  outcome?: string
}
export interface OAuthHealthReport {
  checked_at: string
  group_id: number | null
  accounts: OAuthHealthAccount[]
}
export async function checkOAuthHealth(groupID: number | null) {
  const { data } = await apiClient.post<OAuthHealthReport>('/admin/accounts/oauth-health/check', { group_id: groupID }, { timeout: 30000 })
  return data
}
export async function adjustOAuthHealth(groupID: number | null, accounts: OAuthHealthAccount[], restore = false) {
  const { data } = await apiClient.post<OAuthHealthReport>('/admin/accounts/oauth-health/adjust', {
    group_id: groupID,
    accounts: accounts.map(a => ({ id: a.id, checked_at: a.health?.checked_at })),
    restore
  }, { timeout: 30000 })
  return data
}

export function readOAuthHealth(value: unknown): OAuthHealth | null {
  if (!value || typeof value !== 'object') return null
  const h = value as Partial<OAuthHealth>
  return typeof h.checked_at === 'string' && Number.isFinite(Date.parse(h.checked_at)) &&
    typeof h.status === 'string' && typeof h.reason === 'string' && h.stats &&
    typeof h.concurrency === 'number' ? h as OAuthHealth : null
}

export function oauthHealthTone(status: string) {
  if (['stable', 'recovered'].includes(status)) return 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
  if (['rate_limited', 'quota_limited', 'overloaded'].includes(status)) return 'bg-amber-50 text-amber-800 dark:bg-amber-900/30 dark:text-amber-300'
  if (['auth_error', 'upstream_error'].includes(status)) return 'bg-red-50 text-red-700 dark:bg-red-900/30 dark:text-red-300'
  return 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'
}
