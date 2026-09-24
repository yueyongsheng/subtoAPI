import { apiClient } from '../client'

export const qualityAccountTypes = ['oauth', 'apikey', 'setup-token', 'bedrock'] as const
export type QualityAccountType = typeof qualityAccountTypes[number]
export const qualityProbeKeys = ['svg_html', 'reasoning_exact', 'english_knowledge', 'structured_output', 'custom'] as const
export type QualityProbeKey = typeof qualityProbeKeys[number]
export interface QualityCustomProbe { prompt: string; expected: string }

export interface OAuthQualityAccount {
  id: number
  name: string
  platform: string
  type: QualityAccountType
  group_ids: number[]
  status: string
  schedulable: boolean
}

export interface OAuthQualityProbeResult {
  key: string
  label: string
  status: 'passed' | 'review' | 'failed'
  summary: string
  output_preview?: string
  latency_ms?: number
}

export interface OAuthQualityAccountResult extends Omit<OAuthQualityAccount, 'status'> {
  status: 'passed' | 'review' | 'failed'
  summary: string
  passed: number
  total: number
  probes: OAuthQualityProbeResult[]
}

export interface OAuthQualityReport {
  checked_at: string
  group_id: number | null
  model_id: string
  account_types: QualityAccountType[]
  probe_keys: QualityProbeKey[]
  custom_probe?: QualityCustomProbe
  accounts: OAuthQualityAccountResult[]
}

export async function listOAuthQualityAccounts(groupID: number | null, accountTypes: QualityAccountType[]) {
  const { data } = await apiClient.post<{ accounts: OAuthQualityAccount[] }>(
    '/admin/accounts/oauth-quality/accounts',
    { group_id: groupID, account_types: accountTypes },
    { timeout: 30000 }
  )
  return data.accounts
}

export async function runOAuthQuality(groupID: number | null, accountIDs: number[], modelID: string, accountTypes: QualityAccountType[], probeKeys: QualityProbeKey[], customProbe?: QualityCustomProbe) {
  const { data } = await apiClient.post<OAuthQualityReport>(
    '/admin/accounts/oauth-quality/run',
    { group_id: groupID, account_ids: accountIDs, model_id: modelID, account_types: accountTypes, probe_keys: probeKeys, custom_probe: customProbe },
    { timeout: 15 * 60 * 1000 }
  )
  return data
}
