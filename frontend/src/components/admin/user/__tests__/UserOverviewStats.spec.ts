import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { getOverview, type AdminUserOverview } from '@/api/admin/users'
import UserOverviewStats from '../UserOverviewStats.vue'

vi.mock('@/api/admin/users', () => ({ getOverview: vi.fn() }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string, values?: Record<string, unknown>) => values ? `${key} ${JSON.stringify(values)}` : key }) }))

const snapshot: AdminUserOverview = {
  total_users: 208, positive_balance_users: 192, total_balance: 240073.05,
  balance_cny: 9602.92, current_concurrency: 88, max_user_concurrency: 14, active_users_10m: 36,
  today_user_cost: 123.45, today_user_cost_cny: 4.94,
  max_concurrency_user: { id: 750, username: '示例用户', email: 'example@example.test' },
  max_concurrency_user_count: 1,
  queried_at: '2026-09-15T01:00:00Z', window_start: '2026-09-15T00:50:00Z'
}
const render = () => mount(UserOverviewStats)

describe('UserOverviewStats', () => {
  beforeEach(() => vi.mocked(getOverview).mockReset().mockResolvedValue({ ...snapshot }))
  afterEach(() => vi.useRealTimers())

  it('shows server totals, positive-balance scope and the UTC+8 query window', async () => {
    const wrapper = render()
    await flushPromises()
    expect(wrapper.get('[data-testid="overview-balance"]').text()).toBe('$240,073.05')
    expect(wrapper.get('[data-testid="overview-cny"]').text()).toContain('9,602.92')
    expect(wrapper.get('[data-testid="overview-today-spend"]').text()).toBe('$123.45')
    expect(wrapper.get('[data-testid="overview-today-spend-cny"]').text()).toContain('4.94')
    expect(wrapper.get('[data-testid="overview-max-concurrency"]').text()).toContain('14')
    expect(wrapper.get('[data-testid="overview-max-user"]').text()).toContain('示例用户')
    expect(wrapper.get('[data-testid="overview-max-user"]').text()).toContain('#750')
    expect(wrapper.text()).toContain('192')
    expect(wrapper.text()).toContain('2026-09-15 09:00:00')
    expect(wrapper.text()).toContain('08:50:00 – 09:00:00')
    wrapper.unmount()
  })

  it('refreshes only on request and retains the last snapshot when a refresh fails', async () => {
    vi.useFakeTimers()
    const wrapper = render()
    await flushPromises()
    await vi.advanceTimersByTimeAsync(10 * 60 * 1000)
    expect(getOverview).toHaveBeenCalledTimes(1)
    vi.mocked(getOverview).mockResolvedValueOnce({ ...snapshot, current_concurrency: 12, max_user_concurrency: 8, today_user_cost: 250.50, today_user_cost_cny: 10.02 })
    await wrapper.get('button').trigger('click')
    await flushPromises()
    expect(wrapper.get('[data-testid="overview-concurrency"]').text()).toContain('12')
    expect(wrapper.get('[data-testid="overview-max-concurrency"]').text()).toContain('8')
    expect(wrapper.get('[data-testid="overview-today-spend"]').text()).toBe('$250.50')
    expect(wrapper.get('[data-testid="overview-today-spend-cny"]').text()).toContain('10.02')
    vi.mocked(getOverview).mockRejectedValueOnce(new Error('network'))
    await wrapper.get('button').trigger('click')
    await flushPromises()
    expect(wrapper.get('[role="alert"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="overview-balance"]').text()).toBe('$240,073.05')
    wrapper.unmount()
  })

  it('does not display zero when concurrency is unavailable', async () => {
    vi.mocked(getOverview).mockResolvedValueOnce({ ...snapshot, current_concurrency: null, max_user_concurrency: null })
    const wrapper = render()
    await flushPromises()
    expect(wrapper.get('[data-testid="overview-concurrency"]').text()).toBe('—')
    expect(wrapper.get('[data-testid="overview-max-concurrency"]').text()).toBe('—')
    expect(wrapper.get('[data-testid="overview-max-user"]').text()).toBe('—')
    wrapper.unmount()
  })

  it.each([
    [{ ...snapshot, max_concurrency_user_count: 3 }, 'maxConcurrencyTied'],
    [{ ...snapshot, max_concurrency_user: { id: 750, username: ' ', email: 'fallback@example.test' } }, 'fallback@example.test'],
    [{ ...snapshot, max_concurrency_user: { id: 750, username: '', email: '' } }, 'unnamedUser'],
    [{ ...snapshot, max_concurrency_user: null }, 'maxConcurrencyUserUnavailable'],
    [{ ...snapshot, max_user_concurrency: 0, max_concurrency_user: null, max_concurrency_user_count: 0 }, 'noConcurrentUsers']
  ])('shows identity fallback or occupancy state', async (response, expected) => {
    vi.mocked(getOverview).mockResolvedValueOnce(response)
    const wrapper = render()
    await flushPromises()
    expect(wrapper.get('[data-testid="overview-max-user"]').text()).toContain(expected)
    wrapper.unmount()
  })
})
