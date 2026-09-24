<template>
  <AppLayout>
    <TablePageLayout>
      <template #actions>
        <section class="card rounded-2xl p-5" aria-label="本地账号管理预览统计">
          <div class="grid grid-cols-1 gap-5 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-6">
            <div v-for="stat in overviewStats" :key="stat.label" class="min-w-0">
              <p class="mb-2 flex items-center gap-2 text-sm text-gray-500 dark:text-dark-300">
                <Icon :name="stat.icon" size="sm" class="text-primary-500" />
                {{ stat.label }}
              </p>
              <p class="text-3xl font-semibold tabular-nums text-gray-800 dark:text-white" :class="stat.accent ? 'text-primary-500 dark:text-primary-400' : ''">
                {{ stat.value }} <span class="text-xs font-normal text-gray-500 dark:text-dark-300">{{ stat.unit }}</span>
              </p>
              <p class="mt-2 text-xs text-gray-500 dark:text-dark-300">{{ stat.hint }}</p>
            </div>
            <div class="min-w-0 border-t border-gray-100 pt-4 dark:border-dark-600 sm:border-t-0 sm:pt-0 xl:border-l xl:pl-5">
              <p class="mb-2 flex items-center gap-2 text-sm text-gray-500 dark:text-dark-300">
                <Icon name="shield" size="sm" class="text-primary-500" />
                分组可用 OAuth
              </p>
              <div class="space-y-1.5 text-sm">
                <div v-for="group in groupAvailability" :key="group.id" class="flex items-center justify-between gap-3">
                  <span class="truncate text-gray-700 dark:text-dark-200">{{ group.name }}</span>
                  <span class="font-semibold tabular-nums" :class="group.count ? 'text-emerald-600 dark:text-emerald-400' : 'text-amber-600 dark:text-amber-400'">{{ group.count }} 个</span>
                </div>
              </div>
              <p class="mt-2 text-[11px] text-gray-500 dark:text-dark-300">本地演示快照 · 不触发探测</p>
            </div>
          </div>
          <div class="mt-5 flex flex-wrap items-center justify-between gap-3 border-t border-gray-100 pt-4 text-xs dark:border-dark-600">
            <div>
              <p class="tabular-nums text-gray-700 dark:text-dark-200">查询时间：2026-09-24 17:09:27 (UTC+8)</p>
              <p class="mt-1 text-gray-500 dark:text-dark-300">本地预览使用固定演示数据，点击「能力质量检测」查看批量探针流程。</p>
            </div>
            <button type="button" class="btn btn-secondary" @click="refreshPreview">
              <Icon name="refresh" size="sm" />刷新统计
            </button>
          </div>
        </section>
      </template>

      <template #filters>
        <div class="flex flex-wrap-reverse items-start justify-between gap-3">
          <AccountTableFilters
            v-model:searchQuery="params.search"
            :filters="params"
            :groups="groups"
            @update:filters="updateFilters"
          />
          <AccountTableActions :loading="false" @refresh="refreshPreview" @create="showHint('添加账号仅在正式页面可用')">
            <template #before>
              <button class="btn btn-secondary" data-testid="oauth-health-open" @click="showHint('OAuth 检测入口已保留，当前预览聚焦能力质量检测')">
                <Icon name="shield" size="sm" />OAuth 检测
              </button>
              <button class="btn btn-secondary" data-testid="oauth-quality-open" @click="openOAuthQuality">
                <Icon name="check" size="sm" />能力质量检测
              </button>
            </template>
            <template #after>
              <button class="btn btn-secondary px-2 md:px-3" title="自动刷新" @click="autoRefresh = !autoRefresh">
                <Icon name="refresh" size="sm" :class="autoRefresh ? 'animate-spin' : ''" />
                <span class="hidden md:inline">{{ autoRefresh ? '自动刷新：30s' : '自动刷新' }}</span>
              </button>
              <button class="btn btn-secondary px-2 md:px-3" @click="showTools = !showTools">
                <Icon name="more" size="sm" /><span class="hidden md:inline">更多操作</span>
              </button>
            </template>
          </AccountTableActions>
        </div>
        <div v-if="showTools" class="mt-2 rounded-lg border border-gray-200 bg-white p-3 text-sm text-gray-600 shadow-sm dark:border-dark-700 dark:bg-dark-800 dark:text-gray-300">
          预览工具菜单：从 CRS 同步 · 导入 / 导出 · 列显示
        </div>
      </template>

      <template #table>
        <div class="flex items-center justify-between gap-3 rounded-t-xl bg-blue-50 px-3 py-2 text-sm text-blue-800 dark:bg-blue-900/20 dark:text-blue-200">
          <span>批量编辑账号 <span class="ml-2 text-xs">全选当前结果（{{ filteredAccounts.length }}）</span></span>
          <button class="btn btn-primary px-3 py-1.5 text-xs" :disabled="!selectedIDs.length" @click="showHint(`已选择 ${selectedIDs.length} 个演示账号`)">批量更新</button>
        </div>
        <div class="flex min-h-0 flex-1 flex-col overflow-hidden">
          <DataTable
            :columns="columns"
            :data="filteredAccounts"
            :loading="false"
            row-key="id"
            :selectable="true"
            :selected-keys="selectedIDs"
            selection-label="选择演示账号"
            :virtualize-threshold="100"
            @update:selected-keys="updateSelectedIDs"
          >
            <template #cell-name="{ row }">
              <div class="flex min-w-[12rem] flex-col">
                <span class="font-medium text-gray-900 dark:text-white">{{ row.name }}</span>
                <span class="text-xs text-gray-500 dark:text-dark-400">{{ row.email }}</span>
              </div>
            </template>
            <template #cell-platform_type="{ row }">
              <div class="flex min-w-[8rem] flex-col gap-1">
                <span class="inline-flex w-fit items-center gap-1 rounded-md bg-emerald-100 px-2 py-0.5 text-xs font-medium text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300">● {{ row.platformLabel }} <span class="text-emerald-600/70">{{ typeLabel(row.type) }}</span></span>
                <span class="text-[11px] text-gray-500 dark:text-dark-400">Compact Auto</span>
              </div>
            </template>
            <template #cell-capacity="{ row }">
              <span class="rounded bg-gray-100 px-2 py-1 text-xs text-gray-600 dark:bg-dark-700 dark:text-gray-300">{{ row.current_concurrency }} / {{ row.concurrency }}</span>
            </template>
            <template #cell-status="{ row }">
              <span class="rounded-full px-2 py-0.5 text-xs font-medium" :class="row.schedulable ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300' : 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'">{{ row.schedulable ? '正常' : '暂停' }}</span>
            </template>
            <template #cell-schedulable="{ row }">
              <button type="button" class="relative inline-flex h-5 w-9 rounded-full border-2 border-transparent transition-colors" :class="row.schedulable ? 'bg-primary-500' : 'bg-gray-200 dark:bg-dark-600'" @click.stop="toggleSchedulable(row)">
                <span class="pointer-events-none inline-block h-4 w-4 rounded-full bg-white shadow transition" :class="row.schedulable ? 'translate-x-4' : 'translate-x-0'" />
              </button>
            </template>
            <template #cell-today_stats="{ row }">
              <div class="text-xs leading-5 text-gray-600 dark:text-gray-300">请求：{{ row.requests }}<br />Token：{{ row.tokens.toLocaleString() }}<br />成本：<span class="text-emerald-600">US${{ row.cost.toFixed(2) }}</span></div>
            </template>
            <template #cell-groups="{ row }">
              <span v-for="group in row.groups" :key="group" class="mr-1 inline-flex rounded bg-emerald-50 px-2 py-0.5 text-xs text-emerald-700 dark:bg-emerald-900/20 dark:text-emerald-300">{{ group }}</span>
            </template>
            <template #cell-usage="{ row }">
              <div class="flex gap-1 text-[11px] text-gray-500"><span class="rounded bg-gray-100 px-1.5 py-0.5 dark:bg-dark-700">{{ row.usage.requests }} req</span><span class="rounded bg-gray-100 px-1.5 py-0.5 dark:bg-dark-700">{{ row.usage.rate }}</span></div>
            </template>
            <template #cell-rate_multiplier="{ row }"><span class="font-mono text-sm">{{ row.rate_multiplier.toFixed(2) }}x</span></template>
            <template #cell-upstream_billing_rate="{ row }"><span class="text-sm text-gray-500">{{ row.upstream_rate }} <span class="text-primary-500">↻</span></span></template>
            <template #cell-last_used_at="{ row }"><span class="text-sm text-gray-500">{{ row.last_used_at }}</span></template>
            <template #cell-actions>
              <div class="flex items-center gap-1">
                <button class="rounded-lg p-1.5 text-xs text-gray-500 hover:bg-gray-100 dark:hover:bg-dark-700" @click.stop="showHint('编辑仅在正式页面保存')">编辑</button>
                <button class="rounded-lg p-1.5 text-xs text-gray-500 hover:bg-gray-100 dark:hover:bg-dark-700" @click.stop="showHint('更多操作仅用于正式账号')">更多</button>
              </div>
            </template>
          </DataTable>
        </div>
      </template>
    </TablePageLayout>

    <OAuthQualityDialog
      :show="showOAuthQuality"
      :group-id="oauthQualityGroupID"
      :scope-label="oauthQualityScopeLabel"
      :preview="true"
      :groups="groups"
      :initial-types="params.type ? [params.type] : ['oauth', 'apikey', 'setup-token', 'bedrock']"
      :initial-account-ids="selectedIDs"
      @close="showOAuthQuality = false"
    />

    <div v-if="toastMessage" class="fixed bottom-6 right-6 z-[60] rounded-lg bg-gray-900 px-4 py-3 text-sm text-white shadow-xl">{{ toastMessage }}</div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onUnmounted, reactive, ref } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import AccountTableFilters from '@/components/admin/account/AccountTableFilters.vue'
import AccountTableActions from '@/components/admin/account/AccountTableActions.vue'
import OAuthQualityDialog from '@/components/admin/account/OAuthQualityDialog.vue'
import DataTable from '@/components/common/DataTable.vue'
import Icon from '@/components/icons/Icon.vue'
import { previewQualityAccounts } from '@/components/admin/account/qualityPreview'

const authStore = useAuthStore()
const appStore = useAppStore()

// Set only an in-memory admin-shaped user. Keep the token empty so shared
// layout components do not start authenticated API calls in the preview.
authStore.token = null
authStore.user = {
  id: 1,
  username: 'yueyongsheng0125',
  email: 'preview@local.test',
  role: 'admin',
  balance: 8478.93,
  concurrency: 99,
  status: 'active',
  allowed_groups: null,
  balance_notify_enabled: false,
  balance_notify_threshold: null,
  balance_notify_extra_emails: [],
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-09-24T09:00:00Z',
}
appStore.siteName = '悦享 API'
appStore.siteLogo = '/logo.svg'
appStore.siteVersion = 'v0.2.8'
appStore.publicSettingsLoaded = true

// The real admin shell can auto-start its onboarding tour. Mark only the
// preview's synthetic user as seen, then restore the previous local value on
// leaving the page so the preview stays focused on account management.
const previewTourKey = 'admin_guide_1_admin_v4_interactive'
const previousTourValue = localStorage.getItem(previewTourKey)
localStorage.setItem(previewTourKey, 'true')
onUnmounted(() => {
  if (previousTourValue === null) localStorage.removeItem(previewTourKey)
  else localStorage.setItem(previewTourKey, previousTourValue)
})

const groups = [
  { id: 2, name: 'plus-quarantine' },
  { id: 3, name: 'pro-quarantine' },
  { id: 7, name: 'Grok' },
  { id: 9, name: 'Claude' },
] as any[]

const params = reactive({ search: '', platform: '', type: '', status: '', privacy_mode: '', group: '' })
const selectedIDs = ref<number[]>([])
const showOAuthQuality = ref(false)
const oauthQualityGroupID = ref<number | null>(null)
const oauthQualityScopeLabel = ref('全部分组')
const autoRefresh = ref(false)
const showTools = ref(false)
const toastMessage = ref('')

const overviewStats = [
  { icon: 'dollar', label: '用户剩余余额合计', value: '$86,821.22', unit: 'USD', hint: '折合人民币 ¥3,472.85（余额 ÷ 25）', accent: true },
  { icon: 'chart', label: '今日用户总消费', value: '$7,691.62', unit: 'USD', hint: '折合人民币 ¥307.66（消费 ÷ 25）', accent: false },
  { icon: 'bolt', label: '用户当前总并发数', value: '11', unit: '个', hint: '查询时所有用户正在处理的请求合计', accent: false },
  { icon: 'bolt', label: '当前单用户最高并发', value: '2', unit: '个', hint: '用户：1173514237@qq.com #53', accent: false },
  { icon: 'users', label: '近 10 分钟活跃用户', value: '11', unit: '人', hint: '按用户去重 · 仅统计 API 调用用户', accent: false },
] as const

const groupAvailability = [
  { id: 1, name: 'plus-quarantine', count: 3 },
  { id: 2, name: 'pro-quarantine', count: 2 },
]

const accounts = ref(previewQualityAccounts.map((account, index) => ({
  ...account,
  email: 'demo-' + account.id + '@local.test',
  platformLabel: account.platform === 'openai' ? 'OpenAI' : account.platform === 'grok' ? 'Grok' : 'Claude',
  current_concurrency: 0, concurrency: 10, requests: index * 3, tokens: index * 1200,
  cost: index * 0.3, groups: groups.filter(group => account.group_ids.includes(group.id)).map(group => group.name),
  usage: { requests: index * 3, rate: 'A $0.00' }, rate_multiplier: 1,
  upstream_rate: '未探测', last_used_at: '5分钟前',
})))

const filteredAccounts = computed(() => accounts.value.filter((account) => {
  const search = params.search.trim().toLowerCase()
  if (search && !`${account.name} ${account.email}`.toLowerCase().includes(search)) return false
  if (params.platform && account.platform !== params.platform) return false
  if (params.type && account.type !== params.type) return false
  if (params.status === 'active' && (!account.schedulable)) return false
  if (params.group && params.group !== 'ungrouped' && !account.group_ids.includes(Number(params.group))) return false
  if (params.group === 'ungrouped' && account.group_ids.length) return false
  return true
}))

const columns = [
  { key: 'name', label: '名称', sortable: true, class: 'sticky left-0 z-[1] min-w-[15rem] bg-white dark:bg-dark-900' },
  { key: 'id', label: '账号 ID', sortable: true },
  { key: 'platform_type', label: '平台 / 类型' },
  { key: 'capacity', label: '容量' },
  { key: 'status', label: '状态', sortable: true },
  { key: 'schedulable', label: '调度' },
  { key: 'today_stats', label: '今日统计' },
  { key: 'groups', label: '分组' },
  { key: 'usage', label: '用量窗口' },
  { key: 'priority', label: '优先级', formatter: () => '100' },
  { key: 'rate_multiplier', label: '账号倍率' },
  { key: 'upstream_billing_rate', label: '上游声明倍率' },
  { key: 'last_used_at', label: '最近使用' },
  { key: 'actions', label: '操作' },
]

function typeLabel(type: string) { return ({ oauth: 'OAuth', apikey: 'API Key', 'setup-token': 'Setup Token', bedrock: 'AWS Bedrock' } as Record<string, string>)[type] ?? type }
function updateFilters(next: Record<string, unknown>) { Object.assign(params, next) }
function scopeForPreview() {
  if (!params.group) return { id: null, label: '全部分组' }
  if (params.group === 'ungrouped') return { id: -1, label: '未分组' }
  const group = groups.find(item => String(item.id) === String(params.group))
  return { id: Number(params.group), label: group?.name ?? String(params.group) }
}
function openOAuthQuality() {
  const scope = scopeForPreview()
  oauthQualityGroupID.value = scope.id
  oauthQualityScopeLabel.value = scope.label
  showOAuthQuality.value = true
}
function showHint(message: string) {
  toastMessage.value = message
  window.setTimeout(() => { toastMessage.value = '' }, 1800)
}
function refreshPreview() { showHint('统计已刷新（本地演示数据）') }
function toggleSchedulable(account: { schedulable: boolean }) { account.schedulable = !account.schedulable; showHint('已更新演示调度状态') }
function updateSelectedIDs(ids: Array<string | number>) { selectedIDs.value = ids.map(Number) }
</script>

