import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { getOAuthGroupAvailability } from '@/api/admin/oauthAvailability'
import OAuthGroupAvailability from '../OAuthGroupAvailability.vue'
import HelpTooltip from '@/components/common/HelpTooltip.vue'

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
    expect(w.findComponent(HelpTooltip).props('content')).toContain('oauthAvailability.hint')
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
  it('shows the same fewest-first groups in the overview and popup without requesting again', async () => {
    const groups = [
      { group_id: 9, group_name: 'Pool nine', available_accounts: 2 },
      { group_id: 4, group_name: 'Pool four', available_accounts: 0 },
      { group_id: 2, group_name: 'Pool two', available_accounts: 2 },
      { group_id: 7, group_name: 'Pool seven', available_accounts: 1 },
    ]
    vi.mocked(getOAuthGroupAvailability).mockResolvedValueOnce({ ...snapshot, groups })
    const w = mount(OAuthGroupAvailability, { attachTo: document.body, props: { refreshKey: 0 } })
    try {
      await flushPromises()
      const order = ['Pool four', 'Pool seven', 'Pool two', 'Pool nine']
      expect(w.findAll('dt').map(row => row.text())).toEqual(order)
      const trigger = w.get<HTMLButtonElement>('button[aria-haspopup="dialog"]')
      await trigger.trigger('click')
      await flushPromises()
      const popup = document.querySelector<HTMLElement>('[role="dialog"]')!
      expect(Array.from(popup.querySelectorAll('dt'), row => row.textContent)).toEqual(order)
      expect(document.activeElement).toBe(popup)
      expect(trigger.attributes('aria-expanded')).toBe('true')
      popup.querySelector('dt')!.click()
      await flushPromises()
      expect(document.querySelector('[role="dialog"]')).toBe(popup)
      document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
      await flushPromises()
      expect(document.querySelector('[role="dialog"]')).toBeNull()
      expect(document.activeElement).toBe(trigger.element)
      await trigger.trigger('click')
      await flushPromises()
      document.body.click()
      await flushPromises()
      expect(document.querySelector('[role="dialog"]')).toBeNull()
      expect(getOAuthGroupAvailability).toHaveBeenCalledTimes(1)
      expect(groups.map(group => group.group_id)).toEqual([9, 4, 2, 7])
    } finally {
      w.unmount()
    }
  })
})
