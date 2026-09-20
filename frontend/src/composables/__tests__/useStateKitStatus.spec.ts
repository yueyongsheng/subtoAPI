import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import * as plugins from '@/api/admin/plugins'
import { parseStateKitTickets, useStateKitStatus } from '../useStateKitStatus'

vi.mock('@/api/admin/plugins', () => ({ list: vi.fn(), status: vi.fn() }))
const plugin = { id: 1, plugin_key: 'io.github.wangyunjeff.sub2api-state-kit', state: 'enabled' }
const ticket = { account_id: 184, model: 'gpt-6-astra', state: 'ready', remaining_seconds: 120, expires_at: '2026-09-20T10:02:00Z', last_error: '', attempts: 1 }
const response = (rows: unknown[] = [ticket]) => ({ healthy: true, message: '', status_json: JSON.stringify({ host_ready: true, tickets: rows }) })

describe('STATE status lifecycle', () => {
  let wrapper: VueWrapper | undefined
  let service: ReturnType<typeof useStateKitStatus>
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ['Date', 'performance', 'setInterval', 'clearInterval'] })
    vi.setSystemTime(new Date('2026-09-20T10:00:00Z'))
    vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('visible')
    vi.mocked(plugins.list).mockReset().mockResolvedValue([plugin as plugins.PluginInstallation])
    vi.mocked(plugins.status).mockReset().mockResolvedValue(response())
  })
  afterEach(() => { wrapper?.unmount(); wrapper = undefined; vi.restoreAllMocks(); vi.useRealTimers() })
  async function start() {
    wrapper = mount(defineComponent({ setup() { service = useStateKitStatus(); return () => null } }))
    await flushPromises()
  }
  it('uses one shared snapshot for multiple accounts and models and only reads status', async () => {
    vi.mocked(plugins.status).mockResolvedValue(response([ticket, { ...ticket, model: 'gpt-5.6-sol' }, { ...ticket, account_id: 185 }]))
    await start()
    expect(service.ticketsByAccount.value[184]).toHaveLength(2)
    expect(service.ticketsByAccount.value[185]).toHaveLength(1)
    await vi.advanceTimersByTimeAsync(29_000)
    expect(service.elapsedSeconds.value).toBe(29)
    expect(plugins.status).toHaveBeenCalledTimes(1)
    await vi.advanceTimersByTimeAsync(1000)
    expect(plugins.status).toHaveBeenCalledTimes(2)
  })
  it('marks saved rows unavailable after failure, rate limits retries and recovers', async () => {
    await start()
    vi.mocked(plugins.status).mockRejectedValue(new Error('request failed'))
    await vi.advanceTimersByTimeAsync(30_000)
    expect(service.availability.value).toBe('unavailable')
    expect(service.ticketsByAccount.value[184]).toHaveLength(1)
    await vi.advanceTimersByTimeAsync(29_000)
    expect(plugins.status).toHaveBeenCalledTimes(2)
    vi.mocked(plugins.status).mockResolvedValue(response())
    await vi.advanceTimersByTimeAsync(1000)
    expect(service.availability.value).toBe('healthy')
    expect(plugins.status).toHaveBeenCalledTimes(3)
  })
  it('does not reset old countdowns while refreshing and marks slow snapshots stale', async () => {
    await start()
    vi.mocked(plugins.status).mockImplementation(() => new Promise(() => {}))
    await vi.advanceTimersByTimeAsync(46_000)
    expect(service.elapsedSeconds.value).toBe(46)
    expect(service.availability.value).toBe('stale')
    expect(plugins.status).toHaveBeenCalledTimes(2)
  })
  it('stops hidden-page polling, refreshes on return and aborts on unmount', async () => {
    await start()
    vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('hidden')
    document.dispatchEvent(new Event('visibilitychange'))
    await vi.advanceTimersByTimeAsync(60_000)
    expect(plugins.status).toHaveBeenCalledTimes(1)
    expect(service.availability.value).toBe('stale')
    vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('visible')
    document.dispatchEvent(new Event('visibilitychange'))
    await flushPromises()
    expect(plugins.status).toHaveBeenCalledTimes(2)
    vi.mocked(plugins.status).mockImplementation(() => new Promise(() => {}))
    void service.refresh()
    await flushPromises()
    const signal = vi.mocked(plugins.status).mock.calls.at(-1)?.[1]
    wrapper?.unmount(); wrapper = undefined
    expect(signal?.aborted).toBe(true)
    await vi.advanceTimersByTimeAsync(60_000)
    expect(plugins.status).toHaveBeenCalledTimes(3)
  })
  it('distinguishes disabled, removed and unreadable plugins without retaining a healthy badge', async () => {
    await start()
    vi.mocked(plugins.list).mockResolvedValue([{ ...plugin, state: 'disabled' } as plugins.PluginInstallation])
    await service.refresh()
    expect(service.availability.value).toBe('disabled')
    expect(plugins.status).toHaveBeenCalledTimes(1)
    vi.mocked(plugins.list).mockResolvedValue([])
    await service.refresh()
    expect(service.availability.value).toBe('absent')
    expect(service.ticketsByAccount.value).toEqual({})
    vi.mocked(plugins.list).mockRejectedValue(new Error('offline'))
    await service.refresh()
    expect(service.availability.value).toBe('unavailable')
  })
  it('rejects malformed, duplicate, oversized and unhealthy snapshots', async () => {
    expect(() => parseStateKitTickets('{')).toThrow()
    expect(() => parseStateKitTickets(response([{ ...ticket, remaining_seconds: 7200 }]).status_json)).toThrow()
    expect(() => parseStateKitTickets(response([ticket, ticket]).status_json)).toThrow()
    expect(() => parseStateKitTickets(response([{ ...ticket, model: '<script>' }]).status_json)).toThrow()
    expect(() => parseStateKitTickets(' '.repeat(2 * 1024 * 1024 + 1))).toThrow()
    vi.mocked(plugins.status).mockResolvedValue({ ...response(), healthy: false })
    await start()
    expect(service.availability.value).toBe('unavailable')
  })
  it('keeps only display fields and does not render arbitrary upstream errors', () => {
    const rows = parseStateKitTickets(response([{ ...ticket, last_error: 'sensitive arbitrary text', state_value: 'private', proxy: 'private' }]).status_json)
    expect(rows[0]?.lastError).toBe('unknown')
    expect(JSON.stringify(rows)).not.toMatch(/sensitive|private/)
  })
})
