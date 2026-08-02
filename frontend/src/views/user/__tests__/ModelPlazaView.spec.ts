import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import ModelPlazaView from '../ModelPlazaView.vue'

const { getCatalog } = vi.hoisted(() => ({
  getCatalog: vi.fn(),
}))

const messages: Record<string, string> = {
  'modelPlaza.searchPlaceholder': '搜索模型...',
  'modelPlaza.filtersLabel': '模型价格筛选',
  'modelPlaza.platformLabel': '平台',
  'modelPlaza.modelLabel': '模型',
  'modelPlaza.groupLabel': '计费分组',
  'modelPlaza.serviceTierLabel': '服务档位',
  'modelPlaza.modelCount': '模型数量',
  'modelPlaza.priceUnit': 'USD / 1M Token',
  'modelPlaza.customRate': '已应用你的专属倍率',
  'modelPlaza.peakRate': '高峰倍率',
  'modelPlaza.pricingTableLabel': '模型价格对照表',
  'modelPlaza.groupPriceDescription': '实付价格已包含当前分组倍率',
  'modelPlaza.actualPrice': '实付价格',
  'modelPlaza.basePrice': '基础价格',
  'modelPlaza.discountRate': '计费倍率',
  'modelPlaza.baseInput': '基础输入',
  'modelPlaza.baseOutput': '基础输出',
  'modelPlaza.baseCacheRead': '基础缓存读取',
  'modelPlaza.cacheWriteShort': '写入',
  'modelPlaza.cacheReadShort': '读取',
  'modelPlaza.longCachePrice': '长上下文读取',
  'modelPlaza.retry': '重新加载',
  'modelPlaza.loadError': '模型价格加载失败',
  'modelPlaza.emptyTitle': '未找到可用模型',
  'modelPlaza.emptyDescription': '请调整搜索条件',
  'modelPlaza.tiers.standard': 'Standard',
  'modelPlaza.tiers.fast': 'Fast',
  'modelPlaza.groups.plus': 'ChatGPT Plus 分组',
  'modelPlaza.groups.pro': 'ChatGPT Pro 分组',
  'modelPlaza.columns.model': '模型',
  'modelPlaza.columns.input': '输入',
  'modelPlaza.columns.output': '输出',
  'modelPlaza.columns.cacheWrite': '缓存写入',
  'modelPlaza.columns.cacheRead': '缓存读取',
  'modelPlaza.columns.cache': '缓存',
  'modelPlaza.columns.context': '长上下文',
  'modelPlaza.longContextRule': '长上下文规则',
  'common.refresh': '刷新',
}

vi.mock('@/api/modelPlaza', () => ({
  default: { getCatalog },
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
    }),
  }
})

const catalog = {
  currency: 'USD',
  unit: 'per_million_tokens' as const,
  groups: [
    {
      id: 10,
      name: 'Plus',
      platform: 'openai' as const,
      rate_multiplier: 0.12,
      is_custom_rate: false,
      peak_rate_enabled: false,
      peak_start: '',
      peak_end: '',
      peak_rate_multiplier: 0,
    },
    {
      id: 20,
      name: 'Pro',
      platform: 'openai' as const,
      rate_multiplier: 0.18,
      is_custom_rate: true,
      peak_rate_enabled: false,
      peak_start: '',
      peak_end: '',
      peak_rate_multiplier: 0,
    },
  ],
  models: [
    {
      name: 'gpt-5.6-sol',
      platform: 'openai' as const,
      base_standard: { input: 5, output: 30, cache_write: 6.25, cache_read: 0.5 },
      base_fast: { input: 10, output: 60, cache_write: 12.5, cache_read: 1 },
      long_context_threshold: 272000,
      long_context_input_multiplier: 2,
      long_context_output_multiplier: 1.5,
      prices: [
        {
          group_id: 10,
          standard: { input: 0.6, output: 3.6, cache_write: 0.75, cache_read: 0.06 },
          fast: { input: 1.2, output: 7.2, cache_write: 1.5, cache_read: 0.12 },
        },
        {
          group_id: 20,
          standard: { input: 0.9, output: 5.4, cache_write: 1.125, cache_read: 0.09 },
          fast: { input: 1.8, output: 10.8, cache_write: 2.25, cache_read: 0.18 },
        },
      ],
    },
    {
      name: 'gpt-5.6-luna',
      platform: 'openai' as const,
      base_standard: { input: 0.2, output: 1.2, cache_write: 0.25, cache_read: 0.02 },
      base_fast: { input: 0.4, output: 2.4, cache_write: 0.5, cache_read: 0.04 },
      long_context_threshold: 272000,
      long_context_input_multiplier: 2,
      long_context_output_multiplier: 1.5,
      prices: [
        {
          group_id: 10,
          standard: { input: 0.024, output: 0.144, cache_write: 0.03, cache_read: 0.0024 },
          fast: { input: 0.048, output: 0.288, cache_write: 0.06, cache_read: 0.0048 },
        },
      ],
    },
  ],
}

function mountView() {
  return mount(ModelPlazaView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        Icon: true,
        PlatformIcon: true,
      },
    },
  })
}

describe('ModelPlazaView', () => {
  beforeEach(() => {
    getCatalog.mockReset()
    getCatalog.mockResolvedValue(catalog)
  })

  it('shows the default group Standard prices returned by billing', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(getCatalog).toHaveBeenCalledOnce()
    expect(wrapper.text()).toContain('gpt-5.6-sol')
    expect(wrapper.text()).toContain('$0.60')
    expect(wrapper.text()).toContain('$3.60')
    expect(wrapper.text()).toContain('$0.024')
    expect(wrapper.text()).toContain('基础价格')
    expect(wrapper.text()).toContain('$5.00')
    expect(wrapper.text()).toContain('$30.00')
  })

  it('switches between Fast and user group prices without another request', async () => {
    const wrapper = mountView()
    await flushPromises()

    const fastButton = wrapper.findAll('button').find(button => button.text().includes('Fast'))
    expect(fastButton).toBeDefined()
    await fastButton!.trigger('click')
    expect(wrapper.text()).toContain('$1.20')
    expect(wrapper.text()).toContain('$7.20')

    const proButton = wrapper.findAll('button').find(button => button.text().includes('Pro'))
    expect(proButton).toBeDefined()
    await proButton!.trigger('click')

    expect(wrapper.text()).toContain('$1.80')
    expect(wrapper.text()).toContain('$10.80')
    expect(wrapper.text()).toContain('已应用你的专属倍率')
    expect(getCatalog).toHaveBeenCalledOnce()
  })

  it('filters models locally and renders a useful empty state', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('input[type="search"]').setValue('luna')
    expect(wrapper.text()).toContain('gpt-5.6-luna')
    expect(wrapper.text()).not.toContain('gpt-5.6-sol')

    await wrapper.get('input[type="search"]').setValue('missing-model')
    expect(wrapper.text()).toContain('未找到可用模型')
  })

  it('keeps the previous catalog visible when refresh fails', async () => {
    const wrapper = mountView()
    await flushPromises()

    getCatalog.mockRejectedValueOnce(new Error('network down'))
    const refreshButton = wrapper.findAll('button').find(button => button.attributes('title') === '刷新')
    await refreshButton!.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('gpt-5.6-sol')
    expect(wrapper.get('[role="alert"]').text()).not.toBe('')
  })
})
