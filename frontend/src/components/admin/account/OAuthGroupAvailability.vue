<template>
  <div ref="containerRef" class="min-w-0 border-t border-gray-100 pt-4 dark:border-dark-600 sm:border-t-0 sm:pt-0 xl:border-l xl:pl-5" :aria-busy="loading" data-testid="oauth-group-availability">
    <div class="mb-1 flex items-center gap-2 text-sm text-gray-500 dark:text-dark-300">
      <Icon name="shield" size="sm" class="shrink-0 text-primary-500" />
      <span class="min-w-0 truncate" :title="t('admin.accounts.oauthAvailability.title')">{{ t('admin.accounts.oauthAvailability.title') }}</span>
      <HelpTooltip :content="t('admin.accounts.oauthAvailability.hint')" trigger="click" placement="bottom">
        <template #trigger>
          <button type="button" class="flex rounded text-gray-400 hover:text-primary-500 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-400" :aria-label="t('admin.accounts.oauthAvailability.details')" :title="t('admin.accounts.oauthAvailability.hint')">
            <Icon name="infoCircle" size="sm" />
          </button>
        </template>
      </HelpTooltip>
      <button
        v-if="report?.groups.length"
        ref="allGroupsTrigger"
        type="button"
        class="ml-auto inline-flex shrink-0 items-center gap-1 rounded px-1 py-0.5 text-xs font-medium text-primary-600 hover:bg-primary-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-400 dark:text-primary-400 dark:hover:bg-primary-500/10"
        :aria-label="t('admin.accounts.oauthAvailability.viewAll', { count: report.groups.length })"
        :aria-expanded="showAllGroups"
        :aria-controls="popoverId"
        aria-haspopup="dialog"
        @click="toggleAllGroups"
        @keydown.down.prevent="openAllGroups"
      >
        {{ t('admin.accounts.oauthAvailability.allCount', { count: report.groups.length }) }}
        <Icon name="chevronDown" size="sm" class="transition-transform" :class="{ 'rotate-180': showAllGroups }" />
      </button>
    </div>
    <p v-if="!report && !error" class="text-sm text-gray-500">{{ loading ? t('common.loading') : '—' }}</p>
    <p v-if="error" role="alert" class="mb-2 text-xs text-red-600 dark:text-red-400">{{ t('admin.accounts.oauthAvailability.failed') }}</p>
    <template v-if="report">
      <dl v-if="report.groups.length" class="max-h-16 overflow-y-auto overscroll-contain rounded pr-2 [scrollbar-width:thin] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-400" :tabindex="report.groups.length > 2 ? 0 : undefined" :aria-label="t('admin.accounts.oauthAvailability.title')">
        <div v-for="group in sortedGroups" :key="group.group_id" class="flex h-8 items-center justify-between gap-3" :data-group-id="group.group_id">
          <dt class="min-w-0 truncate text-sm text-gray-700 dark:text-dark-200" :title="group.group_name">{{ group.group_name }}</dt>
          <dd class="shrink-0 text-base font-semibold tabular-nums" :class="group.available_accounts ? 'text-emerald-600 dark:text-emerald-400' : 'text-amber-600 dark:text-amber-400'">{{ group.available_accounts }}<span class="ml-1 text-xs font-normal text-gray-500">{{ t('admin.accounts.oauthAvailability.unit') }}</span></dd>
        </div>
      </dl>
      <p v-else class="text-xs text-gray-500">{{ t('admin.accounts.oauthAvailability.empty') }}</p>
      <p class="mt-1 truncate text-[11px] leading-4 tabular-nums text-gray-500 dark:text-dark-300" :title="`${t('admin.accounts.oauthAvailability.queriedAt')} ${timestamp(report.queried_at)} (UTC+8)`">{{ t('admin.accounts.oauthAvailability.queriedAt') }} {{ timestamp(report.queried_at) }} (UTC+8)</p>
    </template>
    <Teleport to="body">
      <div
        v-if="showAllGroups && report"
        :id="popoverId"
        ref="popoverRef"
        role="dialog"
        :aria-labelledby="`${popoverId}-title`"
        tabindex="-1"
        class="fixed z-50 flex flex-col overflow-hidden rounded-xl border border-gray-200 bg-white shadow-xl outline-none dark:border-dark-600 dark:bg-dark-800"
        :style="popoverStyle"
      >
        <div class="flex shrink-0 items-center justify-between gap-3 border-b border-gray-100 px-4 py-3 dark:border-dark-600">
          <div class="flex items-center gap-2">
            <h3 :id="`${popoverId}-title`" class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.accounts.oauthAvailability.allGroups') }}</h3>
            <span class="rounded-full bg-primary-50 px-2 py-0.5 text-xs tabular-nums text-primary-600 dark:bg-primary-500/10 dark:text-primary-400">{{ report.groups.length }}</span>
          </div>
          <button type="button" class="rounded p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-400 dark:hover:bg-dark-700 dark:hover:text-dark-200" :aria-label="t('common.close')" @click="closeAllGroups(true)">
            <Icon name="x" size="sm" />
          </button>
        </div>
        <div class="min-h-0 overflow-y-auto overscroll-contain px-4 [scrollbar-width:thin] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary-400" tabindex="0" :aria-label="t('admin.accounts.oauthAvailability.allGroups')">
          <div class="sticky top-0 flex justify-between gap-4 bg-white py-2 text-xs text-gray-500 dark:bg-dark-800 dark:text-dark-300" aria-hidden="true">
            <span>{{ t('admin.accounts.oauthAvailability.groupName') }}</span>
            <span :title="t('admin.accounts.oauthAvailability.sortAscending')">{{ t('admin.accounts.oauthAvailability.availableAccounts') }} ↑</span>
          </div>
          <dl class="divide-y divide-gray-100 pb-1 dark:divide-dark-700">
            <div v-for="group in sortedGroups" :key="group.group_id" class="flex items-center justify-between gap-5 py-2.5">
              <dt class="min-w-0 break-words text-sm leading-5 text-gray-700 dark:text-dark-100" style="overflow-wrap: anywhere">{{ group.group_name }}</dt>
              <dd class="shrink-0 text-base font-semibold tabular-nums" :class="group.available_accounts ? 'text-emerald-600 dark:text-emerald-400' : 'text-amber-600 dark:text-amber-400'">
                {{ group.available_accounts }}<span class="ml-1 text-xs font-normal text-gray-500 dark:text-dark-300">{{ t('admin.accounts.oauthAvailability.unit') }}</span>
              </dd>
            </div>
          </dl>
        </div>
        <div class="shrink-0 border-t border-gray-100 bg-gray-50 px-4 py-2.5 text-[11px] text-gray-500 dark:border-dark-600 dark:bg-dark-900 dark:text-dark-300">
          <p class="tabular-nums">{{ t('admin.accounts.oauthAvailability.queriedAt') }} {{ timestamp(report.queried_at) }} (UTC+8)</p>
          <p v-if="error" class="mt-1 text-red-600 dark:text-red-400">{{ t('admin.accounts.oauthAvailability.failed') }}</p>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, useId, watch, type CSSProperties } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import HelpTooltip from '@/components/common/HelpTooltip.vue'
import { getOAuthGroupAvailability, type OAuthGroupAvailabilityReport } from '@/api/admin/oauthAvailability'

const props = defineProps<{ refreshKey: number }>()
const { t } = useI18n()
const report = ref<OAuthGroupAvailabilityReport | null>(null)
const sortedGroups = computed(() => [...(report.value?.groups ?? [])].sort(
  (a, b) => a.available_accounts - b.available_accounts || a.group_id - b.group_id,
))
const loading = ref(false)
const error = ref(false)
const containerRef = ref<HTMLElement | null>(null)
const allGroupsTrigger = ref<HTMLButtonElement | null>(null)
const popoverRef = ref<HTMLElement | null>(null)
const popoverId = `oauth-groups-${useId()}`
const showAllGroups = ref(false)
const popoverStyle = ref<CSSProperties>({ visibility: 'hidden' })
let controller: AbortController | null = null
const timestamp = (value: string) => new Intl.DateTimeFormat('sv-SE', { timeZone: 'Asia/Shanghai', year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit', hourCycle: 'h23' }).format(new Date(value))

async function positionPopover() {
  if (!showAllGroups.value || !allGroupsTrigger.value || !containerRef.value) return
  const trigger = allGroupsTrigger.value.getBoundingClientRect()
  const container = containerRef.value.getBoundingClientRect()
  const margin = 12
  const width = Math.min(Math.max(container.width, 320), 400, window.innerWidth - margin * 2)
  popoverStyle.value = { ...popoverStyle.value, width: `${width}px`, maxHeight: `${Math.min(420, window.innerHeight - margin * 2)}px` }
  await nextTick()
  if (!showAllGroups.value || !popoverRef.value) return
  const height = popoverRef.value.getBoundingClientRect().height
  const below = window.innerHeight - trigger.bottom - margin - 8
  const above = trigger.top - margin - 8
  const openBelow = below >= height || below >= above
  const maxHeight = Math.max(0, Math.min(420, openBelow ? below : above))
  popoverStyle.value = {
    width: `${width}px`,
    maxHeight: `${maxHeight}px`,
    left: `${Math.max(margin, Math.min(container.right - width, window.innerWidth - width - margin))}px`,
    top: `${Math.max(margin, openBelow ? trigger.bottom + 8 : trigger.top - 8 - Math.min(height, maxHeight))}px`,
  }
}

async function openAllGroups() {
  if (!report.value?.groups.length || showAllGroups.value) return
  showAllGroups.value = true
  popoverStyle.value = { visibility: 'hidden' }
  await positionPopover()
  if (showAllGroups.value) popoverRef.value?.focus({ preventScroll: true })
}

function closeAllGroups(restoreFocus = false) {
  showAllGroups.value = false
  if (restoreFocus) allGroupsTrigger.value?.focus({ preventScroll: true })
}

function toggleAllGroups() {
  if (showAllGroups.value) closeAllGroups(true)
  else void openAllGroups()
}

function onOutsideInteraction(event: Event) {
  if (!showAllGroups.value || !(event.target instanceof Node)) return
  if (allGroupsTrigger.value?.contains(event.target) || popoverRef.value?.contains(event.target)) return
  closeAllGroups()
}

function onKeydown(event: KeyboardEvent) {
  if (showAllGroups.value && event.key === 'Escape') {
    event.preventDefault()
    closeAllGroups(true)
  }
}

function onViewportChange(event: Event) {
  if (!showAllGroups.value || (event.target instanceof Node && popoverRef.value?.contains(event.target))) return
  const rect = allGroupsTrigger.value?.getBoundingClientRect()
  if (rect && (rect.bottom < 0 || rect.top > window.innerHeight)) closeAllGroups()
  else void positionPopover()
}
async function load() {
  controller?.abort()
  const request = new AbortController()
  controller = request
  loading.value = true
  error.value = false
  try {
    const result = await getOAuthGroupAvailability(request.signal)
    if (!request.signal.aborted) report.value = result
  } catch {
    if (!request.signal.aborted) error.value = true
  } finally {
    if (controller === request) loading.value = false
  }
}
onMounted(() => {
  void load()
  document.addEventListener('click', onOutsideInteraction, true)
  document.addEventListener('focusin', onOutsideInteraction)
  document.addEventListener('keydown', onKeydown)
  window.addEventListener('resize', onViewportChange)
  window.addEventListener('scroll', onViewportChange, true)
})
watch(() => props.refreshKey, load)
watch(report, () => {
  if (!report.value?.groups.length) closeAllGroups()
  else void positionPopover()
}, { flush: 'post' })
onUnmounted(() => {
  controller?.abort()
  document.removeEventListener('click', onOutsideInteraction, true)
  document.removeEventListener('focusin', onOutsideInteraction)
  document.removeEventListener('keydown', onKeydown)
  window.removeEventListener('resize', onViewportChange)
  window.removeEventListener('scroll', onViewportChange, true)
})
</script>
