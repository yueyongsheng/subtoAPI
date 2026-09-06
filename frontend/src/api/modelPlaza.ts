/**
 * Model Plaza API（公开端点，可匿名访问）
 * 以分组为中心的模型价目：分组信息 + 模型渠道定价 + LiteLLM 官方参考价。
 * 带 token 请求时后端会额外返回专属分组与用户专属倍率。
 */

import { apiClient } from './client'
import type { UserPricingInterval, UserSupportedModelPricing } from './channels'

/** 官方参考价（USD per token，与计费目录同源；字段缺失 = 目录未覆盖）。 */
export interface PlazaOfficialPricing {
  input_price: number | null
  output_price: number | null
  /** 5m 缓存写入（= LiteLLM cache_creation）。 */
  cache_write_price: number | null
  /** 1h 缓存写入（LiteLLM cache_creation_above_1hr），多数模型缺失。 */
  cache_write_1h_price?: number | null
  cache_read_price: number | null
  /** 官方长上下文阶梯（多档模型才有），不受分组开关影响。 */
  intervals?: UserPricingInterval[]
}

/**
 * 多档时的计价基准：
 * - whole_request：整单按所在档单价计价（目录阶梯、渠道区间）；
 * - marginal：仅超出阈值的部分按该档单价计价（平台旧规则）。
 */
export type PlazaLongContextBasis = 'whole_request' | 'marginal'

/** 分时倍率时段：配置时区当天 [start_time, end_time) 内整单实付乘 multiplier。 */
export interface PlazaTimePricingPeriod {
  start_time: string
  end_time: string
  multiplier: number
}

/** 计费会生效的分时倍率（仅倍率 ≠ 1 的时段，已按开始时间升序）。 */
export interface PlazaTimePricing {
  /** IANA 时区名，如 Asia/Shanghai。 */
  timezone: string
  /** true 时时段仅周一至周五生效，周末整天按标准价计费。 */
  weekdays_only?: boolean
  periods: PlazaTimePricingPeriod[]
}

export interface PlazaModel {
  name: string
  platform: string
  /** 实收口径的展示定价：多档时 intervals 为各档绝对单价（已由计费服务折算）；均为标准时段价。 */
  pricing: UserSupportedModelPricing | null
  official_pricing: PlazaOfficialPricing | null
  /** 仅多档模型返回。 */
  long_context_basis?: PlazaLongContextBasis
  /** 仅配置了分时倍率的模型返回。 */
  time_pricing?: PlazaTimePricing
}

export interface ModelPlazaGroup {
  id: number
  name: string
  description: string
  platform: string
  /** 'standard' | 'subscription' */
  subscription_type: string
  rate_multiplier: number
  /** 登录且管理员为该用户配了专属倍率时返回；生效倍率 = user_rate ?? rate_multiplier。 */
  user_rate_multiplier?: number
  peak_rate_enabled: boolean
  peak_start: string
  peak_end: string
  peak_rate_multiplier: number
  is_exclusive: boolean
  /** 生图独立倍率：true 时图片计费模型的实付倍率取 image_rate_multiplier，不取分组/专属倍率。 */
  image_rate_independent: boolean
  image_rate_multiplier: number
  /** 分组是否启用长上下文阶梯计费；false 时实付列只展示最低档，官方阶梯仅供参考。 */
  long_context_pricing_enabled: boolean
  models: PlazaModel[]
}

export interface ModelPlazaResponse {
  /** 管理员配置的全局价格说明（Markdown）。 */
  description: string
  groups: ModelPlazaGroup[]
}

export interface ModelPlazaTierPrice {
  input: number
  output: number
  cache_write: number
  cache_read: number
  group_id?: number
}

export interface ModelPlazaModel {
  name: string
  platform: string
  prices: Array<ModelPlazaTierPrice & { group_id: number; standard: ModelPlazaTierPrice; fast: ModelPlazaTierPrice }>
  base_standard: ModelPlazaTierPrice
  base_fast: ModelPlazaTierPrice
  long_context_threshold: number
  long_context_input_multiplier: number
  long_context_output_multiplier: number
}

export interface ModelPlazaCatalog {
  currency: string
  unit: string
  groups: Array<ModelPlazaGroup & { is_custom_rate?: boolean }>
  models: ModelPlazaModel[]
}

function tierPrice(input: number | null, output: number | null, write: number | null, read: number | null): ModelPlazaTierPrice {
  return { input: input ?? 0, output: output ?? 0, cache_write: write ?? 0, cache_read: read ?? 0 }
}

function toLegacyCatalog(response: ModelPlazaResponse): ModelPlazaCatalog {
  const byModel = new Map<string, ModelPlazaModel>()
  for (const group of response.groups) {
    for (const model of group.models) {
      const key = `${model.platform}:${model.name}`
      const official = model.official_pricing
      const actual = model.pricing
      const standard = tierPrice(actual?.input_price ?? official?.input_price ?? null, actual?.output_price ?? official?.output_price ?? null, actual?.cache_write_price ?? official?.cache_write_price ?? null, actual?.cache_read_price ?? official?.cache_read_price ?? null)
      const fast = tierPrice(standard.input * 2, standard.output * 2, standard.cache_write * 2, standard.cache_read * 2)
      const base = tierPrice(official?.input_price ?? standard.input / Math.max(group.rate_multiplier, 1), official?.output_price ?? standard.output / Math.max(group.rate_multiplier, 1), official?.cache_write_price ?? standard.cache_write / Math.max(group.rate_multiplier, 1), official?.cache_read_price ?? standard.cache_read / Math.max(group.rate_multiplier, 1))
      const current = byModel.get(key) ?? {
        name: model.name,
        platform: model.platform,
        prices: [],
        base_standard: base,
        base_fast: tierPrice(base.input * 2, base.output * 2, base.cache_write * 2, base.cache_read * 2),
        long_context_threshold: 0,
        long_context_input_multiplier: 1,
        long_context_output_multiplier: 1,
      }
      current.prices.push({ group_id: group.id, ...standard, standard, fast })
      byModel.set(key, current)
    }
  }
  return { currency: 'USD', unit: 'per_million_tokens', groups: response.groups.map(group => ({ ...group, is_custom_rate: Boolean(group.user_rate_multiplier) })), models: [...byModel.values()] }
}

/** 获取模型广场数据。开关未启用时后端返回 404。 */
export async function getModelPlaza(options?: { signal?: AbortSignal }): Promise<ModelPlazaResponse> {
  const { data } = await apiClient.get<ModelPlazaResponse>('/model-plaza', {
    signal: options?.signal
  })
  return data
}

export async function getCatalog(options?: { signal?: AbortSignal }): Promise<ModelPlazaCatalog> {
  return toLegacyCatalog(await getModelPlaza(options))
}

export const modelPlazaAPI = { getModelPlaza, getCatalog }

export default modelPlazaAPI
