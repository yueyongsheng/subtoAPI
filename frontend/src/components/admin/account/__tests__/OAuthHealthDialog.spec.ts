import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { checkOAuthHealth, adjustOAuthHealth, type OAuthHealthAccount } from '@/api/admin/oauthHealth'
import OAuthHealthDialog from '../OAuthHealthDialog.vue'
import OAuthHealthBadge from '../OAuthHealthBadge.vue'

vi.mock('@/api/admin/oauthHealth', async original => ({ ...await original<typeof import('@/api/admin/oauthHealth')>(), checkOAuthHealth: vi.fn(), adjustOAuthHealth: vi.fn() }))
vi.mock('@/api/client', () => ({ apiClient: {} }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string, args?: object) => key + (args ? JSON.stringify(args) : '') }) }))

const account = (id = 1): OAuthHealthAccount => ({ id, name: `fixture-${id}`, platform: 'openai', group_ids: [15, 16], concurrency: 10, schedulable: true, outcome: 'checked', health: {
  policy_version: 2, action: 'reduce', account_id: id, checked_at: new Date().toISOString(), window_start: new Date(Date.now() - 30 * 60000).toISOString(), window_end: new Date().toISOString(), status: 'rate_limited', reason: 'reduce_concurrency', concurrency: 10, recommended_concurrency: 5,
  stats: { observed_requests: 20, output_requests: 15, rate_limited_requests: 5, quota_requests: 0, overloaded_requests: 0, auth_requests: 0, other_error_requests: 0, pressure_requests: 5, pressure_minutes: 3 }
} })
const render = (groupId: number | null = 15) => mount(OAuthHealthDialog, { props: { show: true, groupId, scopeLabel: 'fixture-group' }, global: { stubs: { BaseDialog: { template: '<div><slot/><slot name="footer"/></div>' } } } })
const button = (w: ReturnType<typeof render>, key: string) => w.findAll('button').find(b => b.text().includes(key))!

describe('OAuth health controls', () => {
  beforeEach(() => { vi.mocked(checkOAuthHealth).mockReset().mockResolvedValue({ checked_at: new Date().toISOString(), group_id: 15, accounts: [account()] }); vi.mocked(adjustOAuthHealth).mockReset() })
  afterEach(() => vi.useRealTimers())
  it.each([15, null, -1])('uses the selected scope %s without applying on inspection', async group => {
    const w = render(group); await flushPromises()
    expect(checkOAuthHealth).toHaveBeenCalledWith(group)
    expect(adjustOAuthHealth).not.toHaveBeenCalled()
    expect(w.text()).toContain('sharedGroups')
    w.unmount()
  })
  it('applies recommendations without an unguarded restore shortcut', async () => {
    const w = render(); await flushPromises()
    const a = account(); a.concurrency = 5; a.outcome = 'reduce'; a.health!.concurrency = 5; a.health!.recommended_concurrency = 5; a.health!.reason = 'change_cooldown'; a.health!.action = '';  a.health!.last_change = { at: new Date().toISOString(), before: 10, after: 5, action: 'reduce', actor_id: 1 }
    vi.mocked(adjustOAuthHealth).mockResolvedValue({ checked_at: new Date().toISOString(), group_id: 15, accounts: [a] })
    await button(w, '.apply').trigger('click'); await flushPromises()
    expect(adjustOAuthHealth).toHaveBeenCalledWith(15, [expect.objectContaining({ id: 1 })], false)
    expect(button(w, '.apply').attributes('disabled')).toBeDefined()
    expect(w.findAll('button').some(b => b.text().includes('.restore'))).toBe(false)
    w.unmount()
  })
  it.each(['increase', 'cooldown', 'rollback'] as const)('offers actionable %s recommendations', async action => {
    const a = account(); a.health!.action = action; a.health!.reason = action === 'increase' ? 'increase_concurrency' : action === 'cooldown' ? 'cooldown_recommended' : 'rollback_increase'
    a.health!.recommended_concurrency = action === 'increase' ? 15 : action === 'cooldown' ? 10 : 5
    vi.mocked(checkOAuthHealth).mockResolvedValueOnce({ checked_at: new Date().toISOString(), group_id: 15, accounts: [a] })
    const w = render(); await flushPromises()
    expect(button(w, '.apply').attributes('disabled')).toBeUndefined()
    expect(w.text()).toContain(`.action.${action}`)
    w.unmount()
  })
  it('does not apply conflicting or insufficient evidence', async () => {
    const a = account(); a.outcome = 'conflict'
    vi.mocked(checkOAuthHealth).mockResolvedValueOnce({ checked_at: new Date().toISOString(), group_id: 15, accounts: [a] })
    const w = render(); await flushPromises()
    expect(button(w, '.apply').attributes('disabled')).toBeDefined()
    expect(w.text()).toContain('.conflict'); w.unmount()
  })
  it('clears actionable results on a failed recheck', async () => {
    const w = render(); await flushPromises()
    vi.mocked(checkOAuthHealth).mockRejectedValueOnce(new Error('offline'))
    await button(w, '.recheck').trigger('click'); await flushPromises()
    expect(w.find('[role="alert"]').exists()).toBe(true)
    expect(button(w, '.apply').attributes('disabled')).toBeDefined(); w.unmount()
  })
  it('labels old results historical and hides a copied account result', () => {
    const a = account(); a.health!.checked_at = new Date(Date.now() - 31 * 60000).toISOString()
    const w = mount(OAuthHealthBadge, { props: { value: a.health, accountId: 1 } })
    expect(w.text()).toContain('.historical'); w.unmount()
    const copied = mount(OAuthHealthBadge, { props: { value: a.health, accountId: 2 } })
    expect(copied.text()).toBe(''); copied.unmount()
  })
})
