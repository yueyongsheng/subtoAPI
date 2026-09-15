<template>
  <section class="card rounded-2xl p-5" :aria-label="t('admin.users.overview.title')" :aria-busy="loading">
    <div class="grid grid-cols-1 gap-5 sm:grid-cols-3">
      <div class="min-w-0">
        <p class="mb-2 flex items-center gap-2 text-sm text-gray-500 dark:text-dark-300">
          <Icon name="dollar" size="sm" class="text-primary-500" />
          {{ t('admin.users.overview.balance') }}
        </p>
        <p class="flex flex-wrap items-baseline gap-x-2 text-3xl font-semibold tabular-nums text-primary-500">
          <span data-testid="overview-balance">{{ stats ? money(stats.total_balance, 'USD') : '—' }}</span>
          <span class="text-xs font-normal text-gray-500 dark:text-dark-300">USD</span>
        </p>
        <p class="mt-2 flex flex-wrap items-baseline gap-x-1 text-xs text-gray-700 dark:text-dark-200">
          {{ t('admin.users.overview.cny') }}
          <span class="font-medium tabular-nums" data-testid="overview-cny">{{ stats ? money(stats.balance_cny, 'CNY') : '—' }}</span>
          <span class="text-gray-500 dark:text-dark-300">{{ t('admin.users.overview.conversion') }}</span>
        </p>
        <p class="mt-2 text-xs text-gray-500 dark:text-dark-300">
          {{ t('admin.users.overview.positiveUsers', { count: stats?.positive_balance_users ?? '—' }) }}
        </p>
      </div>
      <div class="min-w-0 border-t border-gray-100 pt-4 dark:border-dark-600 sm:border-l sm:border-t-0 sm:pl-5 sm:pt-0">
        <p class="mb-2 flex items-center gap-2 text-sm text-gray-500 dark:text-dark-300">
          <Icon name="bolt" size="sm" class="text-primary-500" />
          {{ t('admin.users.overview.concurrency') }}
        </p>
        <p class="text-3xl font-semibold tabular-nums text-gray-800 dark:text-white" data-testid="overview-concurrency">
          {{ stats?.current_concurrency ?? '—' }}
          <span v-if="stats?.current_concurrency != null" class="text-xs font-normal text-gray-500 dark:text-dark-300">{{ t('admin.users.overview.requests') }}</span>
        </p>
        <p class="mt-3 text-xs text-gray-500 dark:text-dark-300">{{ stats && stats.current_concurrency === null ? t('admin.users.overview.concurrencyUnavailable') : t('admin.users.overview.concurrencyHint') }}</p>
      </div>
      <div class="min-w-0 border-t border-gray-100 pt-4 dark:border-dark-600 sm:border-l sm:border-t-0 sm:pl-5 sm:pt-0">
        <p class="mb-2 flex items-center gap-2 text-sm text-gray-500 dark:text-dark-300">
          <Icon name="users" size="sm" class="text-primary-500" />
          {{ t('admin.users.overview.activeUsers') }}
        </p>
        <p class="text-3xl font-semibold tabular-nums text-gray-800 dark:text-white" data-testid="overview-active-users">
          {{ stats?.active_users_10m ?? '—' }}
          <span class="text-xs font-normal text-gray-500 dark:text-dark-300">{{ t('admin.users.overview.people') }}</span>
        </p>
        <p class="mt-3 text-xs text-gray-500 dark:text-dark-300">{{ t('admin.users.overview.activeUsersHint') }}</p>
        <p v-if="stats" class="mt-1 text-xs tabular-nums text-gray-500 dark:text-dark-300">{{ clock(stats.window_start) }} – {{ clock(stats.queried_at) }}</p>
      </div>
    </div>
    <div class="mt-5 flex flex-wrap items-center justify-between gap-3 border-t border-gray-100 pt-4 dark:border-dark-600">
      <div class="text-xs">
        <p v-if="stats" class="tabular-nums text-gray-700 dark:text-dark-200">{{ t('admin.users.overview.queriedAt') }}: {{ timestamp(stats.queried_at) }} (UTC+8)</p>
        <p v-if="error" role="alert" class="text-red-600 dark:text-red-400">{{ t('admin.users.overview.loadFailed') }}</p>
        <p class="mt-1 text-gray-500 dark:text-dark-300">{{ t('admin.users.overview.scope') }}</p>
      </div>
      <button type="button" class="btn btn-secondary" :disabled="loading" @click="loadOverview">
        <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
        {{ t('admin.users.overview.refresh') }}
      </button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { getOverview, type AdminUserOverview } from '@/api/admin/users'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const stats = ref<AdminUserOverview | null>(null)
const loading = ref(false)
const error = ref(false)
let controller: AbortController | null = null
const timeOptions: Intl.DateTimeFormatOptions = { timeZone: 'Asia/Shanghai', hour: '2-digit', minute: '2-digit', second: '2-digit', hourCycle: 'h23' }
const clock = (value: string) => new Intl.DateTimeFormat('zh-CN', timeOptions).format(new Date(value))
const timestamp = (value: string) => new Intl.DateTimeFormat('sv-SE', { ...timeOptions, year: 'numeric', month: '2-digit', day: '2-digit' }).format(new Date(value))
const money = (value: number, currency: string) => new Intl.NumberFormat(currency === 'CNY' ? 'zh-CN' : 'en-US', { style: 'currency', currency, minimumFractionDigits: 2, maximumFractionDigits: 2 }).format(value)

async function loadOverview() {
  if (loading.value) return
  controller = new AbortController()
  loading.value = true
  error.value = false
  try {
    stats.value = await getOverview(controller.signal)
  } catch {
    if (!controller.signal.aborted) error.value = true
  } finally {
    loading.value = false
  }
}

onMounted(loadOverview)
onUnmounted(() => controller?.abort())
</script>
