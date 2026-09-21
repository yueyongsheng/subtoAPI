import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import PluginsView from '../PluginsView.vue'

const {
  listPlugins,
  uploadPlugin,
  enablePlugin,
  savePluginConfig,
  createUISession,
  stepUpRun,
} = vi.hoisted(() => ({
  listPlugins: vi.fn(),
  uploadPlugin: vi.fn(),
  enablePlugin: vi.fn(),
  savePluginConfig: vi.fn(),
  createUISession: vi.fn(),
  stepUpRun: vi.fn((action: () => Promise<unknown>) => action()),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    plugins: {
      list: listPlugins,
      upload: uploadPlugin,
      enable: enablePlugin,
      disable: vi.fn(),
      remove: vi.fn(),
      getConfig: vi.fn().mockResolvedValue({}),
      saveConfig: savePluginConfig,
      test: vi.fn().mockResolvedValue({ success: true, message: 'ok', latency_ms: 1 }),
      createUISession,
    },
  },
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
    showInfo: vi.fn(),
  }),
}))

vi.mock('@/composables/useStepUp', () => ({
  useStepUp: () => ({ run: stepUpRun }),
  isStepUpBlocked: () => false,
  isStepUpCancelled: () => false,
  stepUpBlockReason: () => '',
}))

vi.mock('vue-i18n', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue-i18n')>()),
  useI18n: () => ({ t: (key: string) => key }),
}))

const plugin = {
  id: 7,
  plugin_key: 'local.test.transport',
  name: 'Test Transport',
  version: '1.0.0',
  description: '',
  author: 'test',
  manifest: {
    schema_version: 1,
    id: 'local.test.transport',
    name: 'Test Transport',
    version: '1.0.0',
    requires: {
      sub2api: '>=0.1.0',
      plugin_protocol: 1,
      transport_api: 1,
      ui_bridge: 1,
    },
    capabilities: [],
    ui: { entrypoint: 'ui/index.html' },
  },
  binary_sha256: 'a'.repeat(64),
  signature_status: 'trusted' as const,
  state: 'disabled' as const,
  last_error: '',
  installed_at: '2026-08-22T00:00:00Z',
  updated_at: '2026-08-22T00:00:00Z',
  bindings: [
    {
      id: 1,
      plugin_id: 7,
      capability: 'openai.oauth.outbound_transport.v1',
      platform: 'openai',
      account_type: 'oauth',
      enabled: false,
      rollout_percent: 100,
    },
  ],
  compatibility: {
    compatible: true,
    tested: true,
    status: 'compatible' as const,
    message: '',
    current_sub2api_version: '0.1.0',
    required_sub2api_version: '>=0.1.0',
    recommended_sub2api_version: '0.1.0',
    plugin_protocol: 1,
    transport_api: 1,
    ui_bridge: 1,
  },
  runtime_healthy: false,
  runtime_message: '',
}

function mountView() {
  return mount(PluginsView, {
    attachTo: document.body,
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        BaseDialog: { name: 'BaseDialog', template: '<div><slot /></div>' },
        Icon: true,
        TotpStepUpDialog: true,
      },
    },
  })
}

describe('管理员插件页二次验证', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    stepUpRun.mockImplementation((action: () => Promise<unknown>) => action())
    listPlugins.mockResolvedValue([plugin])
    uploadPlugin.mockResolvedValue(plugin)
    enablePlugin.mockResolvedValue(plugin)
    savePluginConfig.mockResolvedValue({ enabled: true })
    createUISession.mockResolvedValue({
      url: '/api/v1/plugin-ui/token/index.html#bridge_token=bridge',
      bridge_token: 'bridge',
      ui_bridge_version: 1,
      expires_at: '2026-08-22T01:00:00Z',
    })
  })

  it('启用插件通过 step-up 控制器执行', async () => {
    const wrapper = mountView()
    await flushPromises()

    const button = wrapper.findAll('button').find((item) => item.text().includes('admin.plugins.enable'))
    expect(button).toBeDefined()
    await button!.trigger('click')
    await flushPromises()

    expect(stepUpRun).toHaveBeenCalledTimes(1)
    expect(enablePlugin).toHaveBeenCalledWith(7, 100, false)
    wrapper.unmount()
  })

  it('上传插件通过 step-up 控制器执行', async () => {
    const wrapper = mountView()
    await flushPromises()
    const input = wrapper.get('input[type="file"]')
    Object.defineProperty(input.element, 'files', {
      configurable: true,
      value: [new File(['plugin'], 'transport.s2plugin', { type: 'application/zip' })],
    })

    await input.trigger('change')
    await flushPromises()

    expect(stepUpRun).toHaveBeenCalledTimes(1)
    expect(uploadPlugin).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })
})

describe('插件配置加载生命周期', () => {
  let wrapper: ReturnType<typeof mountView>
  const session = (token: string) => ({
    url: `/api/v1/plugin-ui/${token}/index.html#bridge_token=${token}`,
    bridge_token: token,
    ui_bridge_version: 1,
    expires_at: '2099-01-01T00:00:00Z',
  })
  const open = async () => {
    await wrapper.findAll('button').find(b => b.text() === 'admin.plugins.configure')!.trigger('click')
    await flushPromises()
  }
  const ready = async (token: string, origin = 'null') => {
    window.dispatchEvent(new MessageEvent('message', {
      source: wrapper.get('iframe').element.contentWindow,
      origin,
      data: { source: 'sub2api-plugin-ui', bridge_token: token, type: 'sub2api.plugin.ready' },
    }))
    await flushPromises()
  }
  beforeEach(async () => {
    vi.clearAllMocks()
    vi.useFakeTimers()
    listPlugins.mockResolvedValue([plugin])
    createUISession.mockResolvedValue(session('current'))
    wrapper = mountView()
    await flushPromises()
  })
  afterEach(() => {
    wrapper.unmount()
    vi.useRealTimers()
  })
  it('静态文档 load 但脚本未就绪时超时退出加载，并允许新会话重试', async () => {
    await open()
    await wrapper.get('iframe').trigger('load')
    expect(wrapper.text()).toContain('admin.plugins.loadingUI')
    await vi.advanceTimersByTimeAsync(30_000)
    expect(wrapper.text()).not.toContain('admin.plugins.loadingUI')
    expect(wrapper.text()).toContain('admin.plugins.uiLoadTimeout')
    expect(wrapper.find('iframe').exists()).toBe(false)
    createUISession.mockResolvedValueOnce(session('retry'))
    await wrapper.get('[data-testid="plugin-ui-retry"]').trigger('click')
    await flushPromises()
    expect(wrapper.get('iframe').attributes('src')).toContain('/retry/')
    await ready('retry')
    expect(wrapper.text()).not.toContain('admin.plugins.loadingUI')
    expect(wrapper.text()).not.toContain('admin.plugins.uiLoadTimeout')
  })
  it('会话请求超时后忽略迟到响应', async () => {
    let resolve!: (value: ReturnType<typeof session>) => void
    createUISession.mockImplementationOnce(() => new Promise(r => { resolve = r }))
    await open()
    await vi.advanceTimersByTimeAsync(30_000)
    expect(wrapper.text()).toContain('admin.plugins.uiLoadTimeout')
    resolve(session('late'))
    await flushPromises()
    expect(wrapper.find('iframe').exists()).toBe(false)
  })
  it('关闭再打开时旧会话不得替换新会话', async () => {
    let resolve!: (value: ReturnType<typeof session>) => void
    createUISession.mockImplementationOnce(() => new Promise(r => { resolve = r }))
    await open()
    wrapper.findComponent({ name: 'BaseDialog' }).vm.$emit('close')
    await flushPromises()
    await open()
    resolve(session('stale'))
    await flushPromises()
    expect(wrapper.get('iframe').attributes('src')).toContain('/current/')
  })
  it('仅接受当前 iframe、opaque origin 和正确 token 的就绪消息', async () => {
    await open()
    window.dispatchEvent(new MessageEvent('message', {
      source: window,
      origin: 'null',
      data: { source: 'sub2api-plugin-ui', bridge_token: 'current', type: 'sub2api.plugin.ready' },
    }))
    await flushPromises()
    expect(wrapper.text()).toContain('admin.plugins.loadingUI')
    await ready('wrong')
    await ready('current', 'https://example.com')
    expect(wrapper.text()).toContain('admin.plugins.loadingUI')
    await ready('current')
    expect(wrapper.text()).not.toContain('admin.plugins.loadingUI')
    await vi.advanceTimersByTimeAsync(30_000)
    expect(wrapper.text()).not.toContain('admin.plugins.uiLoadTimeout')
    expect(wrapper.get('iframe').attributes('sandbox')).toBe('allow-scripts')
  })
  it('关闭配置取消尚未完成的会话请求和加载计时', async () => {
    createUISession.mockImplementationOnce(() => new Promise(() => {}))
    await open()
    const signal = createUISession.mock.calls[0][1] as AbortSignal
    expect(signal.aborted).toBe(false)
    wrapper.findComponent({ name: 'BaseDialog' }).vm.$emit('close')
    await flushPromises()
    expect(signal.aborted).toBe(true)
    await vi.advanceTimersByTimeAsync(30_000)
    expect(wrapper.text()).not.toContain('admin.plugins.uiLoadTimeout')
    expect(wrapper.find('iframe').exists()).toBe(false)
  })
})
