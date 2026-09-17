import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { getSpendingRanking, getConcurrencyRanking, type AdminUserSpendingRanking } from '@/api/admin/users'
import UserRankingDropdown from '../UserRankingDropdown.vue'

vi.mock('@/api/admin/users', () => ({ getSpendingRanking: vi.fn(), getConcurrencyRanking: vi.fn() }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string, values?: Record<string, unknown>) => values ? `${key} ${JSON.stringify(values)}` : key }) }))

const user = { id: 12, username: 'Example user', email: 'example@example.test' }
const spending: AdminUserSpendingRanking = {
  queried_at: '2026-09-17T03:20:00Z', window_start: '2026-09-16T16:00:00Z',
  users: [{ user_id: 12, user, cost: 1234.56, cost_cny: 49.38 }],
}
const mounted: ReturnType<typeof mount>[] = []
const render = (kind: 'spending' | 'concurrency') => {
  const wrapper = mount(UserRankingDropdown, { props: { kind }, attachTo: document.body })
  mounted.push(wrapper)
  return wrapper
}
const dialog = () => document.body.querySelector('[role="dialog"]')!

describe('UserRankingDropdown', () => {
  beforeEach(() => {
    vi.mocked(getSpendingRanking).mockReset().mockResolvedValue(spending)
    vi.mocked(getConcurrencyRanking).mockReset().mockResolvedValue({ queried_at: spending.queried_at, users: [{ user_id: 12, user, current_concurrency: 8 }] })
  })
  afterEach(() => {
    mounted.splice(0).forEach(wrapper => wrapper.unmount())
    document.body.innerHTML = ''
    vi.useRealTimers()
  })

  it('loads only the clicked ranking and shows both currencies and UTC+8 query time', async () => {
    vi.useFakeTimers()
    const wrapper = render('spending')
    await flushPromises()
    await vi.advanceTimersByTimeAsync(60000)
    expect(getSpendingRanking).not.toHaveBeenCalled()
    await wrapper.get('button').trigger('click')
    await flushPromises()
    expect(getSpendingRanking).toHaveBeenCalledTimes(1)
    expect(getConcurrencyRanking).not.toHaveBeenCalled()
    expect(dialog().textContent).toContain('$1,234.56')
    expect(dialog().textContent).toContain('49.38')
    expect(dialog().textContent).toContain('RMB')
    expect(dialog().textContent).toContain('Example user')
    expect(dialog().textContent).toContain('#12')
    expect(dialog().textContent).toContain('2026-09-17 11:20:00')
    await vi.advanceTimersByTimeAsync(60000)
    expect(getSpendingRanking).toHaveBeenCalledTimes(1)
  })

  it('shows only returned active users without ten-row padding and uses identity fallbacks', async () => {
    vi.mocked(getConcurrencyRanking).mockResolvedValueOnce({ queried_at: spending.queried_at, users: [
      { user_id: 12, user: { ...user, username: ' ' }, current_concurrency: 8 },
      { user_id: 13, user: null, current_concurrency: 1 },
    ] })
    const wrapper = render('concurrency')
    await wrapper.get('button').trigger('click')
    await flushPromises()
    expect(dialog().querySelectorAll('li')).toHaveLength(2)
    expect(dialog().textContent).toContain('example@example.test')
    expect(dialog().textContent).toContain('maxConcurrencyUserUnavailable')
    expect(dialog().textContent).toContain('#13')
    expect(dialog().textContent).not.toContain('USD')
    expect(getSpendingRanking).not.toHaveBeenCalled()
  })

  it('distinguishes no active requests from a failed query, and supports retry', async () => {
    vi.mocked(getConcurrencyRanking).mockRejectedValueOnce(new Error('Redis unavailable'))
    const wrapper = render('concurrency')
    await wrapper.get('button').trigger('click')
    await flushPromises()
    expect(dialog().querySelector('[role="alert"]')).not.toBeNull()
    expect(dialog().textContent).not.toContain('noConcurrency')
    vi.mocked(getConcurrencyRanking).mockResolvedValueOnce({ queried_at: spending.queried_at, users: [] })
    ;(dialog().querySelector('[data-testid="ranking-refresh"]') as HTMLButtonElement).click()
    await flushPromises()
    expect(dialog().querySelector('[role="alert"]')).toBeNull()
    expect(dialog().textContent).toContain('noConcurrency')
    expect(dialog().querySelectorAll('li')).toHaveLength(0)
  })

  it('refreshes on reopen, restores focus on Escape and ignores an aborted older response', async () => {
    let resolveOld!: (value: AdminUserSpendingRanking) => void
    vi.mocked(getSpendingRanking).mockReturnValueOnce(new Promise(resolve => { resolveOld = resolve }))
    const wrapper = render('spending')
    const trigger = wrapper.get('button')
    await trigger.trigger('click')
    await flushPromises()
    const signal = vi.mocked(getSpendingRanking).mock.calls[0][0]!
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    await flushPromises()
    expect(signal.aborted).toBe(true)
    expect(document.activeElement).toBe(trigger.element)
    expect(document.querySelector('[role="dialog"]')).toBeNull()
    await trigger.trigger('keydown', { key: 'ArrowDown' })
    await flushPromises()
    resolveOld({ ...spending, users: [{ user_id: 99, user: { ...user, username: 'Stale user' }, cost: 999, cost_cny: 39.96 }] })
    await flushPromises()
    expect(getSpendingRanking).toHaveBeenCalledTimes(2)
    expect(dialog().textContent).toContain('Example user')
    expect(dialog().textContent).not.toContain('Stale user')
    document.body.click()
    await flushPromises()
    expect(document.querySelector('[role="dialog"]')).toBeNull()
  })

  it('keeps the last successful snapshot visibly dated if manual refresh fails', async () => {
    const wrapper = render('spending')
    await wrapper.get('button').trigger('click')
    await flushPromises()
    vi.mocked(getSpendingRanking).mockRejectedValueOnce(new Error('network'))
    ;(dialog().querySelector('[data-testid="ranking-refresh"]') as HTMLButtonElement).click()
    await flushPromises()
    expect(dialog().querySelector('[role="alert"]')).not.toBeNull()
    expect(dialog().textContent).toContain('$1,234.56')
    expect(dialog().textContent).toContain('2026-09-17 11:20:00')
  })
})
