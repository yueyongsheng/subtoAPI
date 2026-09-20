import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import AccountStateTickets from '../AccountStateTickets.vue'
import type { StateKitTicket } from '@/composables/useStateKitStatus'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('@/utils/format', () => ({ formatDateTime: (value: string | Date) => String(value) }))
const ticket: StateKitTicket = { accountId: 184, model: 'gpt-6-astra', state: 'ready', remainingSeconds: 120, expiresAt: '2026-09-20T10:02:00Z', lastError: '', attempts: 1 }
const props = { tickets: [ticket], availability: 'healthy' as const, elapsedSeconds: 10, queriedAt: Date.parse('2026-09-20T10:00:00Z') }

describe('STATE account display', () => {
  it('shows configured models and counts down to expired without a green zero', async () => {
    const w = mount(AccountStateTickets, { props })
    expect(w.text()).toContain('STATE · astra')
    expect(w.text()).toContain('1m50s')
    expect(w.get('[data-state="ready"]').classes()).toContain('text-emerald-600')
    await w.setProps({ elapsedSeconds: 120 })
    expect(w.find('[data-state="ready"]').exists()).toBe(false)
    expect(w.get('[data-state="expired"]').classes()).not.toContain('text-emerald-600')
    expect(w.text()).not.toContain('0m00s')
    w.unmount()
  })
  it('preserves a usable ticket during renewal but stops claiming usability after expiry', async () => {
    const w = mount(AccountStateTickets, { props: { ...props, tickets: [{ ...ticket, state: 'renewing' }] } })
    expect(w.get('[data-state="renewing"]').text()).toContain('1m50s')
    await w.setProps({ elapsedSeconds: 130 })
    expect(w.get('[data-state="harvesting"]').exists()).toBe(true)
    w.unmount()
  })
  it('shows a renewal failure with the remaining usable lifetime and mapped error', () => {
    const w = mount(AccountStateTickets, { props: { ...props, tickets: [{ ...ticket, lastError: 'upstream_forbidden' }] } })
    expect(w.text()).toContain('renewalFailed')
    expect(w.text()).toContain('1m50s')
    expect(w.get('[data-state="ready"]').classes()).toContain('text-amber-600')
    expect(w.get('[title]').attributes('title')).toContain('errors.upstream_forbidden')
    w.unmount()
  })
  it('hides old green countdowns on query failure, stale data or a disabled plugin', async () => {
    const w = mount(AccountStateTickets, { props })
    for (const availability of ['unavailable', 'stale', 'disabled'] as const) {
      await w.setProps({ availability })
      expect(w.get(`[data-state="${availability}"]`).exists()).toBe(true)
      expect(w.text()).not.toContain('1m50s')
    }
    w.unmount()
  })
  it('shows collection/cooldown separately and leaves unconfigured accounts empty', async () => {
    const w = mount(AccountStateTickets, { props: { ...props, tickets: [{ ...ticket, state: 'cooldown', remainingSeconds: 0 }] } })
    expect(w.get('[data-state="cooldown"]').exists()).toBe(true)
    await w.setProps({ tickets: [] })
    expect(w.find('[data-testid="account-state-tickets"]').exists()).toBe(false)
    w.unmount()
  })
})
