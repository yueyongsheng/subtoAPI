<template>
  <button
    ref="triggerRef"
    type="button"
    class="ml-auto inline-flex shrink-0 items-center gap-1 rounded px-1 py-0.5 text-xs font-medium text-primary-600 hover:bg-primary-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-400 dark:text-primary-400 dark:hover:bg-primary-500/10"
    :data-testid="`${kind}-ranking-trigger`"
    :aria-label="title"
    :aria-expanded="opened"
    :aria-controls="popoverId"
    aria-haspopup="dialog"
    @click="toggle"
    @keydown.down.prevent="open"
  >
    {{ t('admin.users.overview.ranking.all') }}
    <Icon name="chevronDown" size="sm" class="transition-transform" :class="{ 'rotate-180': opened }" />
  </button>
  <Teleport to="body">
    <div
      v-if="opened"
      :id="popoverId"
      ref="popoverRef"
      role="dialog"
      :aria-labelledby="`${popoverId}-title`"
      :aria-busy="loading"
      tabindex="-1"
      class="fixed z-50 flex flex-col overflow-hidden rounded-xl border border-gray-200 bg-white shadow-xl outline-none dark:border-dark-600 dark:bg-dark-800"
      :style="popoverStyle"
    >
      <div class="flex shrink-0 items-center justify-between gap-3 border-b border-gray-100 px-4 py-3 dark:border-dark-600">
        <div class="flex items-center gap-2">
          <h3 :id="`${popoverId}-title`" class="text-sm font-semibold text-gray-900 dark:text-white">{{ title }}</h3>
          <span v-if="report" class="rounded-full bg-primary-50 px-2 py-0.5 text-xs tabular-nums text-primary-600 dark:bg-primary-500/10 dark:text-primary-400">{{ t('admin.users.overview.ranking.count', { count: rows.length }) }}</span>
        </div>
        <button type="button" class="rounded p-1 text-gray-400 hover:bg-gray-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-400 dark:hover:bg-dark-700" :aria-label="t('common.close')" @click="close(true)">
          <Icon name="x" size="sm" />
        </button>
      </div>
      <div class="min-h-0 overflow-y-auto overscroll-contain px-4 [scrollbar-width:thin] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary-400" tabindex="0" :aria-label="title">
        <p v-if="loading && !report" role="status" class="py-8 text-center text-sm text-gray-500">{{ t('common.loading') }}</p>
        <p v-if="error" role="alert" class="py-3 text-sm text-red-600 dark:text-red-400">{{ t('admin.users.overview.ranking.failed') }}</p>
        <template v-if="report">
          <p v-if="!rows.length" class="py-8 text-center text-sm text-gray-500">{{ t(`admin.users.overview.ranking.${kind === 'spending' ? 'noSpending' : 'noConcurrency'}`) }}</p>
          <template v-else>
            <div class="sticky top-0 flex justify-between gap-3 bg-white py-2 text-xs text-gray-500 dark:bg-dark-800 dark:text-dark-300" aria-hidden="true">
              <span>{{ t('admin.users.overview.ranking.user') }}</span>
              <span>{{ t(`admin.users.overview.ranking.${kind === 'spending' ? 'spendingColumn' : 'concurrencyColumn'}`) }} ↓</span>
            </div>
            <ol class="divide-y divide-gray-100 pb-1 dark:divide-dark-700">
              <li v-for="(row, index) in rows" :key="row.user_id" :data-user-id="row.user_id" class="flex items-center gap-3 py-2.5">
                <span class="flex h-6 w-6 shrink-0 items-center justify-center rounded-full text-xs font-medium tabular-nums" :class="index < 3 ? 'bg-primary-50 text-primary-600 dark:bg-primary-500/10 dark:text-primary-400' : 'bg-gray-100 text-gray-500 dark:bg-dark-700 dark:text-dark-300'">{{ index + 1 }}</span>
                <div class="min-w-0 flex-1">
                  <p class="break-words text-sm leading-5 text-gray-800 dark:text-dark-100" style="overflow-wrap: anywhere">{{ userName(row.user) }}</p>
                  <p class="text-[11px] tabular-nums text-gray-500 dark:text-dark-300">#{{ row.user_id }}</p>
                </div>
                <div class="shrink-0 text-right tabular-nums">
                  <template v-if="'cost' in row">
                    <p class="text-sm font-semibold text-gray-800 dark:text-white">{{ money(row.cost, 'USD') }} <span class="text-[10px] font-normal text-gray-500">USD</span></p>
                    <p class="mt-0.5 text-xs text-gray-500 dark:text-dark-300">{{ money(row.cost_cny, 'CNY') }} <span class="text-[10px]">RMB</span></p>
                  </template>
                  <p v-else class="text-base font-semibold text-primary-600 dark:text-primary-400">{{ row.current_concurrency }} <span class="text-xs font-normal text-gray-500">{{ t('admin.users.overview.requests') }}</span></p>
                </div>
              </li>
            </ol>
          </template>
        </template>
      </div>
      <div class="shrink-0 border-t border-gray-100 bg-gray-50 px-4 py-2.5 text-[11px] text-gray-500 dark:border-dark-600 dark:bg-dark-900 dark:text-dark-300">
        <p>{{ t(`admin.users.overview.ranking.${kind === 'spending' ? 'spendingHint' : 'concurrencyHint'}`) }}</p>
        <div class="mt-1.5 flex items-center justify-between gap-2">
          <p v-if="report" class="tabular-nums">{{ t('admin.users.overview.queriedAt') }}: {{ timestamp(report.queried_at) }} (UTC+8)</p>
          <span v-else />
          <button type="button" class="inline-flex shrink-0 items-center gap-1 rounded px-1 py-0.5 text-primary-600 hover:bg-primary-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-400 disabled:opacity-50 dark:text-primary-400 dark:hover:bg-primary-500/10" :disabled="loading" data-testid="ranking-refresh" @click="load">
            <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
            {{ t('common.refresh') }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, useId, type CSSProperties } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { getSpendingRanking, getConcurrencyRanking, type AdminUserSpendingRanking, type AdminUserConcurrencyRanking, type AdminUserOverview } from '@/api/admin/users'

const props = defineProps<{ kind: 'spending' | 'concurrency' }>()
const { t } = useI18n()
const title = computed(() => t(`admin.users.overview.ranking.${props.kind === 'spending' ? 'spendingTitle' : 'concurrencyTitle'}`))
const report = ref<AdminUserSpendingRanking | AdminUserConcurrencyRanking | null>(null)
const rows = computed(() => report.value?.users ?? [])
const opened = ref(false)
const loading = ref(false)
const error = ref(false)
const triggerRef = ref<HTMLButtonElement | null>(null)
const popoverRef = ref<HTMLElement | null>(null)
const popoverId = `user-ranking-${useId()}`
const popoverStyle = ref<CSSProperties>({ visibility: 'hidden' })
let controller: AbortController | null = null
const timestamp = (value: string) => new Intl.DateTimeFormat('sv-SE', { timeZone: 'Asia/Shanghai', year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit', hourCycle: 'h23' }).format(new Date(value))
const money = (value: number, currency: string) => new Intl.NumberFormat(currency === 'CNY' ? 'zh-CN' : 'en-US', { style: 'currency', currency, minimumFractionDigits: 2, maximumFractionDigits: 2 }).format(value)
const userName = (user: AdminUserOverview['max_concurrency_user']) => user
  ? user.username?.trim() || user.email?.trim() || t('admin.users.overview.unnamedUser')
  : t('admin.users.overview.maxConcurrencyUserUnavailable')

async function positionPopover() {
  if (!opened.value || !triggerRef.value) return
  const trigger = triggerRef.value.getBoundingClientRect()
  const margin = 12
  const width = Math.min(440, window.innerWidth - margin * 2)
  popoverStyle.value = { ...popoverStyle.value, width: `${width}px`, maxHeight: `${window.innerHeight - margin * 2}px` }
  await nextTick()
  if (!opened.value || !popoverRef.value) return
  const height = popoverRef.value.getBoundingClientRect().height
  const below = window.innerHeight - trigger.bottom - margin - 8
  const above = trigger.top - margin - 8
  const openBelow = below >= height || below >= above
  const maxHeight = Math.max(0, openBelow ? below : above)
  popoverStyle.value = {
    width: `${width}px`, maxHeight: `${maxHeight}px`,
    left: `${Math.max(margin, Math.min(trigger.right - width, window.innerWidth - width - margin))}px`,
    top: `${Math.max(margin, openBelow ? trigger.bottom + 8 : trigger.top - 8 - Math.min(height, maxHeight))}px`,
  }
}

async function load() {
  controller?.abort()
  const request = new AbortController()
  controller = request
  loading.value = true
  error.value = false
  try {
    const result = await (props.kind === 'spending' ? getSpendingRanking(request.signal) : getConcurrencyRanking(request.signal))
    if (!request.signal.aborted) report.value = result
  } catch {
    if (!request.signal.aborted) error.value = true
  } finally {
    if (controller === request) {
      loading.value = false
      void positionPopover()
    }
  }
}

async function open() {
  if (opened.value) return
  opened.value = true
  report.value = null
  popoverStyle.value = { visibility: 'hidden' }
  void load()
  await positionPopover()
  if (opened.value) popoverRef.value?.focus({ preventScroll: true })
}

function close(restoreFocus = false) {
  opened.value = false
  controller?.abort()
  if (restoreFocus) triggerRef.value?.focus({ preventScroll: true })
}
function toggle() {
  if (opened.value) close(true)
  else void open()
}
function onOutsideInteraction(event: Event) {
  if (!opened.value || !(event.target instanceof Node)) return
  if (triggerRef.value?.contains(event.target) || popoverRef.value?.contains(event.target)) return
  close()
}
function onKeydown(event: KeyboardEvent) {
  if (opened.value && event.key === 'Escape') {
    event.preventDefault()
    close(true)
  }
}
function onViewportChange(event: Event) {
  if (!opened.value || (event.target instanceof Node && popoverRef.value?.contains(event.target))) return
  const rect = triggerRef.value?.getBoundingClientRect()
  if (rect && (rect.bottom < 0 || rect.top > window.innerHeight)) close()
  else void positionPopover()
}
onMounted(() => {
  document.addEventListener('click', onOutsideInteraction, true)
  document.addEventListener('focusin', onOutsideInteraction)
  document.addEventListener('keydown', onKeydown)
  window.addEventListener('resize', onViewportChange)
  window.addEventListener('scroll', onViewportChange, true)
})
onUnmounted(() => {
  controller?.abort()
  document.removeEventListener('click', onOutsideInteraction, true)
  document.removeEventListener('focusin', onOutsideInteraction)
  document.removeEventListener('keydown', onKeydown)
  window.removeEventListener('resize', onViewportChange)
  window.removeEventListener('scroll', onViewportChange, true)
})
</script>
