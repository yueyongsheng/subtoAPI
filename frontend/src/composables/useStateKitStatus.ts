import { computed, onMounted, onUnmounted, ref } from 'vue'
import * as plugins from '@/api/admin/plugins'

const PLUGIN_KEY = 'io.github.wangyunjeff.sub2api-state-kit'
const STATES = ['disabled', 'waiting_host', 'waiting_account', 'renewing', 'harvesting', 'ready', 'cooldown', 'expired', 'queued'] as const
const ERRORS = ['upstream_unauthorized', 'upstream_forbidden', 'upstream_rate_limited', 'model_mismatch', 'state_312', 'harvest_failed', 'fixed_proxy_validation_failed', 'unexpected_state_length', 'identity_unavailable', 'identity_changed', 'invalid_dynamic_proxy', 'managed_proxy_unavailable', 'ticket_persistence_failed', 'attempts_exhausted']

export type StateKitTicketState = typeof STATES[number]
export type StateKitAvailability = 'loading' | 'absent' | 'healthy' | 'disabled' | 'unavailable' | 'stale'
export interface StateKitTicket {
  accountId: number
  model: string
  state: StateKitTicketState
  remainingSeconds: number
  expiresAt: string
  lastError: string
  attempts: number
}

function record(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

// Keep only display fields. Never retain raw plugin events, proxies, configuration or STATE.
export function parseStateKitTickets(raw: string): StateKitTicket[] {
  if (raw.length > 2 * 1024 * 1024) throw new Error('Invalid STATE status')
  const snapshot: unknown = JSON.parse(raw)
  if (!record(snapshot) || snapshot.host_ready !== true || !Array.isArray(snapshot.tickets) || snapshot.tickets.length > 1024) {
    throw new Error('Invalid STATE status')
  }
  const seen = new Set<string>()
  return snapshot.tickets.map((row: unknown) => {
    if (!record(row) || typeof row.account_id !== 'number' || !Number.isSafeInteger(row.account_id) || row.account_id <= 0 ||
      typeof row.model !== 'string' || !/^gpt-[A-Za-z0-9][A-Za-z0-9._-]{0,94}$/.test(row.model) ||
      typeof row.state !== 'string' || !STATES.some(state => state === row.state) ||
      typeof row.remaining_seconds !== 'number' || !Number.isInteger(row.remaining_seconds) || row.remaining_seconds < 0 || row.remaining_seconds > 3600) {
      throw new Error('Invalid STATE ticket')
    }
    const key = `${row.account_id}:${row.model}`
    if (seen.has(key)) throw new Error('Duplicate STATE ticket')
    seen.add(key)
    return {
      accountId: row.account_id, model: row.model, state: row.state as StateKitTicketState,
      remainingSeconds: row.remaining_seconds,
      expiresAt: typeof row.expires_at === 'string' && Number.isFinite(Date.parse(row.expires_at)) ? row.expires_at : '',
      lastError: typeof row.last_error === 'string' && row.last_error ? (ERRORS.includes(row.last_error) ? row.last_error : 'unknown') : '',
      attempts: typeof row.attempts === 'number' && Number.isInteger(row.attempts) && row.attempts >= 0 && row.attempts <= 32 ? row.attempts : 0
    }
  })
}

export function useStateKitStatus() {
  const tickets = ref<StateKitTicket[]>([])
  const current = ref<StateKitAvailability>('loading')
  const loading = ref(false)
  const queriedAt = ref(0)
  const elapsedSeconds = ref(0)
  const stale = ref(false)
  let receivedAt = 0
  let snapshotStartedAt = 0
  let lastAttemptAt = -Infinity
  let controller: AbortController | null = null
  let timer: ReturnType<typeof setInterval> | undefined
  let closed = false

  const availability = computed(() => current.value === 'healthy' && stale.value ? 'stale' : current.value)
  const ticketsByAccount = computed(() => {
    const grouped: Record<number, StateKitTicket[]> = {}
    for (const ticket of tickets.value) (grouped[ticket.accountId] ??= []).push(ticket)
    return grouped
  })

  async function refresh() {
    if (closed || loading.value || document.visibilityState === 'hidden') return
    loading.value = true
    const active = new AbortController()
    controller = active
    const requestStartedAt = performance.now()
    lastAttemptAt = requestStartedAt
    try {
      const installed = await plugins.list(active.signal)
      if (closed || active.signal.aborted) return
      const plugin = installed.find(item => item.plugin_key === PLUGIN_KEY)
      if (!plugin) {
        tickets.value = []
        current.value = 'absent'
      } else if (plugin.state === 'disabled') {
        current.value = 'disabled'
      } else {
        const snapshot = await plugins.status(plugin.id, active.signal)
        if (closed || active.signal.aborted) return
        if (!snapshot.healthy || !snapshot.status_json) throw new Error('STATE runtime unavailable')
        tickets.value = parseStateKitTickets(snapshot.status_json)
        current.value = 'healthy'
      }
      queriedAt.value = Date.now()
      receivedAt = performance.now()
      snapshotStartedAt = requestStartedAt
      // Subtract network time conservatively; the plugin's remaining TTL is the authority.
      elapsedSeconds.value = Math.ceil((receivedAt - requestStartedAt) / 1000)
      stale.value = false
    } catch {
      if (!closed && !active.signal.aborted) current.value = 'unavailable'
    } finally {
      if (controller === active) {
        controller = null
        loading.value = false
      }
    }
  }

  function tick() {
    if (queriedAt.value) {
      const age = Math.max(performance.now() - receivedAt, Date.now() - queriedAt.value)
      elapsedSeconds.value = Math.max(elapsedSeconds.value, Math.ceil((performance.now() - snapshotStartedAt) / 1000))
      stale.value = age > 45_000
    }
    if (document.visibilityState !== 'hidden' && performance.now() - lastAttemptAt >= 30_000) void refresh()
  }

  function onVisibility() {
    if (document.visibilityState === 'hidden') {
      controller?.abort()
      stale.value = true
    } else {
      tick()
      void refresh()
    }
  }

  onMounted(() => {
    void refresh()
    timer = setInterval(tick, 1000)
    document.addEventListener('visibilitychange', onVisibility)
  })
  onUnmounted(() => {
    closed = true
    controller?.abort()
    clearInterval(timer)
    document.removeEventListener('visibilitychange', onVisibility)
  })
  return { ticketsByAccount, availability, loading, queriedAt, elapsedSeconds, refresh }
}
