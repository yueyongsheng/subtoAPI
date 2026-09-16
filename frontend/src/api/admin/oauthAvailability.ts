import { apiClient } from '../client'

export interface OAuthGroupAvailabilityReport {
  queried_at: string
  groups: { group_id: number; group_name: string; available_accounts: number }[]
}

export async function getOAuthGroupAvailability(signal?: AbortSignal) {
  const { data } = await apiClient.get<OAuthGroupAvailabilityReport>('/admin/accounts/oauth-availability', { signal })
  return data
}
