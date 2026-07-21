import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'

import UsageFilters from '../UsageFilters.vue'

// --- i18n messages (only what UsageFilters needs) ---
const messages: Record<string, string> = {
  'admin.usage.userDeletedBadge': 'deleted',
  'admin.usage.userFilter': 'User',
  'admin.usage.searchUserPlaceholder': 'Search user...',
  'usage.apiKeyFilter': 'API Key',
  'admin.usage.searchApiKeyPlaceholder': 'Search API key...',
  'usage.model': 'Model',
  'admin.usage.allModels': 'All Models',
  'admin.usage.account': 'Account',
  'admin.usage.searchAccountPlaceholder': 'Search account...',
  'usage.type': 'Type',
  'admin.usage.allTypes': 'All Types',
  'usage.ws': 'WS',
  'usage.stream': 'Stream',
  'usage.sync': 'Sync',
  'admin.usage.billingType': 'Billing Type',
  'admin.usage.allBillingTypes': 'All Billing Types',
  'admin.usage.billingTypeBalance': 'Balance',
  'admin.usage.billingTypeSubscription': 'Subscription',
  'admin.usage.billingMode': 'Billing Mode',
  'admin.usage.allBillingModes': 'All Billing Modes',
  'admin.usage.billingModeToken': 'Token',
  'admin.usage.billingModePerRequest': 'Per Request',
  'admin.usage.billingModeImage': 'Image',
  'admin.usage.group': 'Group',
  'admin.usage.allGroups': 'All Groups',
  'common.refresh': 'Refresh',
  'common.reset': 'Reset',
  'admin.usage.cleanup.button': 'Cleanup',
  'usage.exportExcel': 'Export',
}

// Mock vue-i18n
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
    }),
  }
})

// Mock the admin API module — we control searchUsers return value per test
const mockSearchUsers = vi.fn()
const mockSearchApiKeys = vi.fn().mockResolvedValue([])
const mockGroupsList = vi.fn().mockResolvedValue({ items: [] })
const mockGetModelStats = vi.fn().mockResolvedValue({ models: [] })
const mockAccountsList = vi.fn().mockResolvedValue({ items: [] })

vi.mock('@/api/admin', () => ({
  adminAPI: {
    usage: {
      searchUsers: (...args: any[]) => mockSearchUsers(...args),
      searchApiKeys: (...args: any[]) => mockSearchApiKeys(...args),
    },
    groups: { list: (...args: any[]) => mockGroupsList(...args) },
    dashboard: { getModelStats: (...args: any[]) => mockGetModelStats(...args) },
    accounts: { list: (...args: any[]) => mockAccountsList(...args) },
  },
}))

// Default props helper
const defaultFilters = () => ({
  user_id: undefined,
  api_key_id: undefined,
  account_id: undefined,
  model: null,
  request_type: null,
  billing_type: null,
  billing_mode: null,
  group_id: null,
  start_date: '',
  end_date: '',
})

function mountFilters(filters = defaultFilters()) {
  return mount(UsageFilters, {
    props: {
      modelValue: filters,
      exporting: false,
      startDate: '2026-05-01',
      endDate: '2026-05-28',
      showActions: false,
      modelOptions: [],
    },
    global: {
      stubs: {
        Select: true,
        Teleport: true,
      },
    },
  })
}

describe('UsageFilters — user search dropdown', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    mockSearchUsers.mockReset()
    mockSearchApiKeys.mockResolvedValue([])
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('(a) labels deleted users with the i18n badge and (b) sorts active users before deleted ones, (c) selection sets user_id', async () => {
    // Arrange: mock returns deleted FIRST (proves sorting re-orders to active-first)
    mockSearchUsers.mockResolvedValue([
      { id: 2, email: 'gone@test.com', deleted: true },
      { id: 1, email: 'active@test.com', deleted: false },
    ])

    const wrapper = mountFilters()

    // Trigger focus (sets showUserDropdown = true) then input (fires debounceUserSearch)
    const input = wrapper.find('input[type="text"]')
    await input.trigger('focus')
    await input.setValue('test')
    await input.trigger('input')

    // Advance debounce timer (300ms) then flush the resolved promise
    vi.advanceTimersByTime(300)
    await flushPromises()

    // --- (b) Sort: active user should appear BEFORE deleted user ---
    // Check the underlying component state via rendered DOM order
    const buttons = wrapper.findAll('.usage-filter-dropdown button[type="button"]')
    const emailTexts = buttons.map((b) => b.text())

    // active@test.com should be listed first
    const activeIdx = emailTexts.findIndex((t) => t.includes('active@test.com'))
    const deletedIdx = emailTexts.findIndex((t) => t.includes('gone@test.com'))
    expect(activeIdx).toBeGreaterThanOrEqual(0)
    expect(deletedIdx).toBeGreaterThanOrEqual(0)
    expect(activeIdx).toBeLessThan(deletedIdx)

    // --- (a) Label: deleted user's button shows the badge text ---
    const deletedButton = buttons[deletedIdx]
    expect(deletedButton.text()).toContain('deleted')

    // active user's button does NOT show the badge text
    const activeButton = buttons[activeIdx]
    expect(activeButton.text()).not.toContain('deleted')

    // --- (c) Selection: clicking active user button sets filters.user_id ---
    await activeButton.trigger('click')
    await flushPromises()

    // The component emits 'update:modelValue' or modifies filters.user_id via toRef
    // selectUser sets filters.value.user_id = u.id and emits 'change'
    const changeEmits = wrapper.emitted('change')
    expect(changeEmits).toBeTruthy()
    expect(changeEmits!.length).toBeGreaterThan(0)

    // Also confirm user_id was set by checking the emitted change came through
    // (the component uses toRef so modelValue is mutated in place and 'change' is emitted)
    expect(wrapper.props('modelValue').user_id).toBe(1)
  })

  it('automatically applies an exact email match without requiring a dropdown click', async () => {
    mockSearchUsers.mockResolvedValue([
      { id: 4, email: 'aprendendo@163.com', deleted: false },
    ])

    const wrapper = mountFilters()
    const input = wrapper.find('input[type="text"]')

    await input.setValue('aprendendo@163.com')
    vi.advanceTimersByTime(300)
    await flushPromises()

    expect(wrapper.props('modelValue').user_id).toBe(4)
    expect(wrapper.emitted('change')).toHaveLength(1)
  })

  it('clears a stale selected user and API key as soon as the user text is edited', async () => {
    mockSearchUsers.mockResolvedValue([
      { id: 4, email: 'aprendendo@163.com', deleted: false },
    ])

    const wrapper = mountFilters()
    const input = wrapper.find('input[type="text"]')

    await input.setValue('aprendendo@163.com')
    vi.advanceTimersByTime(300)
    await flushPromises()
    wrapper.props('modelValue').api_key_id = 99

    await input.setValue('another@example.com')
    await flushPromises()

    expect(wrapper.props('modelValue').user_id).toBeUndefined()
    expect(wrapper.props('modelValue').api_key_id).toBeUndefined()
    expect((input.element as HTMLInputElement).value).toBe('another@example.com')
    expect(wrapper.emitted('change')).toHaveLength(2)
  })

  it('resolves an exact user before emitting refresh', async () => {
    mockSearchUsers.mockResolvedValue([
      { id: 4, email: 'aprendendo@163.com', deleted: false },
    ])

    const wrapper = mount(UsageFilters, {
      props: {
        modelValue: defaultFilters(),
        exporting: false,
        startDate: '2026-05-01',
        endDate: '2026-05-28',
        showActions: true,
        modelOptions: [],
      },
      global: { stubs: { Select: true, Teleport: true } },
    })
    const input = wrapper.find('input[type="text"]')

    await input.setValue('aprendendo@163.com')
    const refreshButton = wrapper.findAll('button').find((button) => button.text() === 'Refresh')
    expect(refreshButton).toBeTruthy()
    await refreshButton!.trigger('click')
    await flushPromises()

    expect(mockSearchUsers).toHaveBeenCalledWith('aprendendo@163.com')
    expect(wrapper.props('modelValue').user_id).toBe(4)
    expect(wrapper.emitted('refresh')).toHaveLength(1)
  })

  it.each([
    { value: '#4', event: 'keydown', eventOptions: { key: 'Enter' } },
    { value: 'aprendendo@163.com', event: 'change', eventOptions: {} },
  ])('resolves an exact user on $event', async ({ value, event, eventOptions }) => {
    mockSearchUsers.mockResolvedValue([
      { id: 4, email: 'aprendendo@163.com', deleted: false },
    ])

    const wrapper = mountFilters()
    const input = wrapper.find('input[type="text"]')

    await input.setValue(value)
    await input.trigger(event, eventOptions)
    await flushPromises()

    expect(wrapper.props('modelValue').user_id).toBe(4)
    expect(wrapper.emitted('change')).toHaveLength(1)
  })

  it('does not emit refresh when typed user text has no exact match', async () => {
    mockSearchUsers.mockResolvedValue([])

    const wrapper = mount(UsageFilters, {
      props: {
        modelValue: defaultFilters(),
        exporting: false,
        startDate: '2026-05-01',
        endDate: '2026-05-28',
        showActions: true,
        modelOptions: [],
      },
      global: { stubs: { Select: true, Teleport: true } },
    })

    await wrapper.find('input[type="text"]').setValue('missing@example.com')
    const refreshButton = wrapper.findAll('button').find((button) => button.text() === 'Refresh')
    await refreshButton!.trigger('click')
    await flushPromises()

    expect(wrapper.props('modelValue').user_id).toBeUndefined()
    expect(wrapper.emitted('refresh')).toBeUndefined()
  })
})

describe('UsageFilters — model options come from prop (no dup request)', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    mockGetModelStats.mockClear()
    mockGroupsList.mockClear()
  })
  afterEach(() => { vi.useRealTimers() })

  it('does not call dashboard.getModelStats on mount and renders model options from prop', async () => {
    const wrapper = mount(UsageFilters, {
      props: {
        modelValue: defaultFilters(),
        exporting: false,
        startDate: '2026-05-01',
        endDate: '2026-05-28',
        showActions: false,
        modelOptions: ['claude-3', 'gpt-4o'],
      },
      global: { stubs: { Select: true, Teleport: true } },
    })
    await flushPromises()

    expect(mockGetModelStats).not.toHaveBeenCalled()

    const opts = (wrapper.vm as any).modelOptions as Array<{ value: string | null; label: string }>
    expect(opts.map((o) => o.value)).toEqual([null, 'claude-3', 'gpt-4o'])
  })
})
