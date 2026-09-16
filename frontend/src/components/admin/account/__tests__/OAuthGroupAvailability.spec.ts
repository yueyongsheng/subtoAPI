import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { getOAuthGroupAvailability } from '@/api/admin/oauthAvailability'
import OAuthGroupAvailability from '../OAuthGroupAvailability.vue'

vi.mock('@/api/admin/oauthAvailability', () => ({ getOAuthGroupAvailability: vi.fn() }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
const snapshot = { queried_at: '2026-09-17T00:00:00Z', groups: [
  { group_id: 15, group_name: '企业pro', available_accounts: 8 },
  { group_id: 2, group_name: 'plus-quarantine', available_accounts: 0 }
] }

describe('OAuth group availability', () => {
  beforeEach(() => vi.mocked(getOAuthGroupAvailability).mockReset().mockResolvedValue(snapshot))
  afterEach(() => vi.useRealTimers())
  it('shows group counts including zero, time and the explicit scope', async () => {
    const w = mount(OAuthGroupAvailability, { props: { refreshKey: 0 } })
    await flushPromises()
    expect(w.get('[data-group-id="15"]').text()).toContain('企业pro8')
    expect(w.get('[data-group-id="2"] dd').text()).toMatch(/^0/)
    expect(w.text()).toContain('2026-09-17 08:00:00')
    expect(w.text()).toContain('oauthAvailability.hint')
    w.unmount()
  })
  it('does not poll and refreshes only with the statistics refresh key', async () => {
    vi.useFakeTimers()
    const w = mount(OAuthGroupAvailability, { props: { refreshKey: 0 } })
    await flushPromises()
    await vi.advanceTimersByTimeAsync(600000)
    expect(getOAuthGroupAvailability).toHaveBeenCalledTimes(1)
    vi.mocked(getOAuthGroupAvailability).mockResolvedValueOnce({ ...snapshot, groups: [{ ...snapshot.groups[0]!, available_accounts: 3 }] })
    await w.setProps({ refreshKey: 1 })
    await flushPromises()
    expect(w.get('dd').text()).toMatch(/^3/)
    expect(getOAuthGroupAvailability).toHaveBeenCalledTimes(2)
    w.unmount()
  })
  it('keeps the dated snapshot on failure and never presents a failed query as zero', async () => {
    const w = mount(OAuthGroupAvailability, { props: { refreshKey: 0 } })
    await flushPromises()
    vi.mocked(getOAuthGroupAvailability).mockRejectedValueOnce(new Error('network'))
    await w.setProps({ refreshKey: 1 })
    await flushPromises()
    expect(w.get('[role="alert"]').exists()).toBe(true)
    expect(w.get('[data-group-id="15"] dd').text()).toMatch(/^8/)
    expect(w.text()).toContain('2026-09-17 08:00:00')
    w.unmount()
  })
  it('shows the empty state and cancels requests on unmount', async () => {
    vi.mocked(getOAuthGroupAvailability).mockResolvedValueOnce({ ...snapshot, groups: [] })
    const w = mount(OAuthGroupAvailability, { props: { refreshKey: 0 } })
    await flushPromises()
    expect(w.text()).toContain('oauthAvailability.empty')
    expect(w.find('dd').exists()).toBe(false)
    const signal = vi.mocked(getOAuthGroupAvailability).mock.calls[0]?.[0]
    w.unmount()
    expect(signal?.aborted).toBe(true)
  })
})
