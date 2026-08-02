import { apiClient } from './client'
import type { GroupPlatform } from '@/types'

export interface ModelPlazaTierPrice {
  input: number
  output: number
  cache_write: number
  cache_read: number
}

export interface ModelPlazaGroupPrice {
  group_id: number
  standard: ModelPlazaTierPrice
  fast: ModelPlazaTierPrice
}

export interface ModelPlazaModel {
  name: string
  platform: GroupPlatform
  prices: ModelPlazaGroupPrice[]
  long_context_threshold: number
  long_context_input_multiplier: number
  long_context_output_multiplier: number
}

export interface ModelPlazaGroup {
  id: number
  name: string
  platform: GroupPlatform
  rate_multiplier: number
  is_custom_rate: boolean
  peak_rate_enabled: boolean
  peak_start: string
  peak_end: string
  peak_rate_multiplier: number
}

export interface ModelPlazaCatalog {
  currency: string
  unit: 'per_million_tokens'
  groups: ModelPlazaGroup[]
  models: ModelPlazaModel[]
}

export async function getCatalog(options?: { signal?: AbortSignal }): Promise<ModelPlazaCatalog> {
  const { data } = await apiClient.get<ModelPlazaCatalog>('/models/catalog', {
    signal: options?.signal,
  })
  return data
}

export const modelPlazaAPI = { getCatalog }

export default modelPlazaAPI
