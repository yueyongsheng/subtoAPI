<template>
  <AppLayout>
    <div class="mx-auto max-w-[1440px] space-y-5">
      <div class="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
        <div class="lg:hidden">
          <h1 class="text-xl font-semibold text-gray-900 dark:text-white">
            {{ t('modelPlaza.title') }}
          </h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
            {{ t('modelPlaza.description') }}
          </p>
        </div>

        <div class="relative w-full lg:max-w-sm">
          <Icon
            name="search"
            size="sm"
            class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400"
          />
          <input
            v-model="searchQuery"
            type="search"
            class="input h-10 pl-9"
            :placeholder="t('modelPlaza.searchPlaceholder')"
          >
        </div>

        <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-end">
          <div
            v-if="catalog.groups.length > 0"
            class="flex min-h-10 max-w-full overflow-x-auto rounded-lg border border-gray-200 bg-white p-1 dark:border-dark-700 dark:bg-dark-800"
            role="tablist"
            :aria-label="t('modelPlaza.groupLabel')"
          >
            <button
              v-for="group in catalog.groups"
              :key="group.id"
              type="button"
              class="flex min-w-max items-center gap-2 rounded-md px-3 py-1.5 text-sm font-medium transition-colors"
              :class="selectedGroupId === group.id
                ? 'bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300'
                : 'text-gray-600 hover:bg-gray-50 dark:text-dark-300 dark:hover:bg-dark-700'"
              :aria-pressed="selectedGroupId === group.id"
              @click="selectedGroupId = group.id"
            >
              <span>{{ group.name }}</span>
              <span class="text-xs opacity-70">{{ formatRate(group.rate_multiplier) }}</span>
            </button>
          </div>

          <div
            class="inline-flex min-h-10 rounded-lg border border-gray-200 bg-white p-1 dark:border-dark-700 dark:bg-dark-800"
            role="group"
            :aria-label="t('modelPlaza.serviceTierLabel')"
          >
            <button
              v-for="tier in serviceTiers"
              :key="tier.value"
              type="button"
              class="flex flex-1 items-center justify-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium transition-colors sm:flex-none"
              :class="selectedTier === tier.value
                ? 'bg-gray-900 text-white dark:bg-white dark:text-dark-900'
                : 'text-gray-600 hover:bg-gray-50 dark:text-dark-300 dark:hover:bg-dark-700'"
              :aria-pressed="selectedTier === tier.value"
              @click="selectedTier = tier.value"
            >
              <Icon v-if="tier.value === 'fast'" name="bolt" size="xs" />
              {{ tier.label }}
            </button>
          </div>

          <button
            type="button"
            class="btn btn-secondary btn-icon h-10 w-10 flex-shrink-0"
            :title="t('common.refresh')"
            :disabled="loading"
            @click="loadCatalog"
          >
            <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
          </button>
        </div>
      </div>

      <div class="flex flex-wrap items-center gap-x-5 gap-y-2 border-y border-gray-200 py-3 text-sm text-gray-600 dark:border-dark-700 dark:text-dark-300">
        <span class="inline-flex items-center gap-2">
          <Icon name="grid" size="sm" class="text-primary-500" />
          {{ t('modelPlaza.modelCount', { count: visibleModels.length }) }}
        </span>
        <span>{{ t('modelPlaza.priceUnit', { currency: catalog.currency || 'USD' }) }}</span>
        <span v-if="selectedGroup?.is_custom_rate" class="text-primary-600 dark:text-primary-400">
          {{ t('modelPlaza.customRate') }}
        </span>
        <span v-if="selectedGroup?.peak_rate_enabled" class="text-amber-700 dark:text-amber-300">
          {{ t('modelPlaza.peakRate', {
            start: selectedGroup.peak_start,
            end: selectedGroup.peak_end,
            rate: formatRate(selectedGroup.peak_rate_multiplier),
          }) }}
        </span>
      </div>

      <div
        v-if="loadError"
        class="flex flex-col gap-3 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 sm:flex-row sm:items-center sm:justify-between dark:border-red-900/50 dark:bg-red-900/20 dark:text-red-300"
        role="alert"
      >
        <span>{{ loadError }}</span>
        <button type="button" class="font-medium underline" @click="loadCatalog">
          {{ t('modelPlaza.retry') }}
        </button>
      </div>

      <div v-if="loading && catalog.models.length === 0" class="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
        <div
          v-for="index in 6"
          :key="index"
          class="h-44 animate-pulse rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-800"
        >
          <div class="h-9 w-9 rounded-lg bg-gray-100 dark:bg-dark-700"></div>
          <div class="mt-4 h-4 w-2/3 rounded bg-gray-100 dark:bg-dark-700"></div>
          <div class="mt-6 grid grid-cols-2 gap-3">
            <div class="h-10 rounded bg-gray-100 dark:bg-dark-700"></div>
            <div class="h-10 rounded bg-gray-100 dark:bg-dark-700"></div>
          </div>
        </div>
      </div>

      <div
        v-else-if="visibleModels.length === 0"
        class="rounded-lg border border-dashed border-gray-300 bg-white/60 px-6 py-16 text-center dark:border-dark-600 dark:bg-dark-800/50"
      >
        <Icon name="inbox" size="xl" class="mx-auto text-gray-400" />
        <h2 class="mt-3 text-sm font-semibold text-gray-900 dark:text-white">
          {{ t('modelPlaza.emptyTitle') }}
        </h2>
        <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
          {{ t('modelPlaza.emptyDescription') }}
        </p>
      </div>

      <div v-else class="overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
        <div class="hidden overflow-x-auto md:block">
          <table class="w-full min-w-[860px] table-fixed border-collapse">
            <thead>
              <tr class="border-b border-gray-200 bg-gray-50/80 text-left text-xs font-medium text-gray-500 dark:border-dark-700 dark:bg-dark-900/40 dark:text-dark-400">
                <th class="w-[30%] px-5 py-3">{{ t('modelPlaza.columns.model') }}</th>
                <th class="px-4 py-3 text-right">{{ t('modelPlaza.columns.input') }}</th>
                <th class="px-4 py-3 text-right">{{ t('modelPlaza.columns.output') }}</th>
                <th class="px-4 py-3 text-right">{{ t('modelPlaza.columns.cacheWrite') }}</th>
                <th class="px-4 py-3 text-right">{{ t('modelPlaza.columns.cacheRead') }}</th>
                <th class="w-28 px-5 py-3 text-right">{{ t('modelPlaza.columns.context') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="entry in visibleModels"
                :key="entry.model.platform + ':' + entry.model.name"
                class="border-b border-gray-100 last:border-b-0 hover:bg-gray-50/60 dark:border-dark-700/70 dark:hover:bg-dark-700/30"
              >
                <td class="px-5 py-4">
                  <div class="flex min-w-0 items-center gap-3">
                    <span class="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg border border-emerald-100 bg-emerald-50 text-emerald-700 dark:border-emerald-900/50 dark:bg-emerald-900/20 dark:text-emerald-300">
                      <PlatformIcon :platform="entry.model.platform" size="lg" />
                    </span>
                    <div class="min-w-0">
                      <div class="truncate text-sm font-semibold text-gray-900 dark:text-white">
                        {{ entry.model.name }}
                      </div>
                      <div class="mt-0.5 text-xs uppercase text-gray-400">
                        {{ entry.model.platform }}
                      </div>
                    </div>
                  </div>
                </td>
                <td class="px-4 py-4 text-right font-mono text-sm font-semibold text-gray-900 dark:text-white">
                  {{ formatPrice(entry.price.input) }}
                </td>
                <td class="px-4 py-4 text-right font-mono text-sm font-semibold text-gray-900 dark:text-white">
                  {{ formatPrice(entry.price.output) }}
                </td>
                <td class="px-4 py-4 text-right font-mono text-sm text-gray-700 dark:text-dark-200">
                  {{ formatPrice(entry.price.cache_write) }}
                </td>
                <td class="px-4 py-4 text-right font-mono text-sm text-gray-700 dark:text-dark-200">
                  {{ formatPrice(entry.price.cache_read) }}
                </td>
                <td class="px-5 py-4 text-right">
                  <span
                    v-if="entry.model.long_context_threshold > 0"
                    class="inline-flex rounded-md bg-amber-50 px-2 py-1 text-xs font-medium text-amber-700 dark:bg-amber-900/20 dark:text-amber-300"
                    :title="longContextDescription(entry.model)"
                  >
                    {{ formatContext(entry.model.long_context_threshold) }}+
                  </span>
                  <span v-else class="text-sm text-gray-400">-</span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div class="divide-y divide-gray-100 md:hidden dark:divide-dark-700">
          <article
            v-for="entry in visibleModels"
            :key="'mobile:' + entry.model.platform + ':' + entry.model.name"
            class="p-4"
          >
            <div class="flex items-start justify-between gap-3">
              <div class="flex min-w-0 items-center gap-3">
                <span class="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg border border-emerald-100 bg-emerald-50 text-emerald-700 dark:border-emerald-900/50 dark:bg-emerald-900/20 dark:text-emerald-300">
                  <PlatformIcon :platform="entry.model.platform" size="lg" />
                </span>
                <div class="min-w-0">
                  <h2 class="break-all text-sm font-semibold text-gray-900 dark:text-white">
                    {{ entry.model.name }}
                  </h2>
                  <span class="text-xs uppercase text-gray-400">{{ entry.model.platform }}</span>
                </div>
              </div>
              <span
                v-if="entry.model.long_context_threshold > 0"
                class="flex-shrink-0 rounded-md bg-amber-50 px-2 py-1 text-xs font-medium text-amber-700 dark:bg-amber-900/20 dark:text-amber-300"
              >
                {{ formatContext(entry.model.long_context_threshold) }}+
              </span>
            </div>

            <dl class="mt-4 grid grid-cols-2 gap-x-4 gap-y-3 text-sm">
              <div>
                <dt class="text-xs text-gray-500 dark:text-dark-400">{{ t('modelPlaza.columns.input') }}</dt>
                <dd class="mt-1 font-mono font-semibold text-gray-900 dark:text-white">{{ formatPrice(entry.price.input) }}</dd>
              </div>
              <div>
                <dt class="text-xs text-gray-500 dark:text-dark-400">{{ t('modelPlaza.columns.output') }}</dt>
                <dd class="mt-1 font-mono font-semibold text-gray-900 dark:text-white">{{ formatPrice(entry.price.output) }}</dd>
              </div>
              <div>
                <dt class="text-xs text-gray-500 dark:text-dark-400">{{ t('modelPlaza.columns.cacheWrite') }}</dt>
                <dd class="mt-1 font-mono text-gray-700 dark:text-dark-200">{{ formatPrice(entry.price.cache_write) }}</dd>
              </div>
              <div>
                <dt class="text-xs text-gray-500 dark:text-dark-400">{{ t('modelPlaza.columns.cacheRead') }}</dt>
                <dd class="mt-1 font-mono text-gray-700 dark:text-dark-200">{{ formatPrice(entry.price.cache_read) }}</dd>
              </div>
            </dl>
          </article>
        </div>
      </div>

      <div
        v-if="longContextRule"
        class="flex items-start gap-3 rounded-lg border border-amber-200 bg-amber-50/70 px-4 py-3 text-sm text-amber-900 dark:border-amber-900/50 dark:bg-amber-900/15 dark:text-amber-200"
      >
        <Icon name="infoCircle" size="sm" class="mt-0.5 flex-shrink-0" />
        <span>{{ longContextRule }}</span>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import modelPlazaAPI, {
  type ModelPlazaCatalog,
  type ModelPlazaModel,
} from '@/api/modelPlaza'
import { extractApiErrorMessage } from '@/utils/apiError'

type ServiceTier = 'standard' | 'fast'

const emptyCatalog = (): ModelPlazaCatalog => ({
  currency: 'USD',
  unit: 'per_million_tokens',
  groups: [],
  models: [],
})

const { t } = useI18n()
const catalog = ref<ModelPlazaCatalog>(emptyCatalog())
const loading = ref(false)
const loadError = ref('')
const searchQuery = ref('')
const selectedGroupId = ref<number | null>(null)
const selectedTier = ref<ServiceTier>('standard')
let abortController: AbortController | null = null

const serviceTiers = computed(() => [
  { value: 'standard' as const, label: t('modelPlaza.tiers.standard') },
  { value: 'fast' as const, label: t('modelPlaza.tiers.fast') },
])

const selectedGroup = computed(() =>
  catalog.value.groups.find(group => group.id === selectedGroupId.value) ?? null,
)

const visibleModels = computed(() => {
  const groupId = selectedGroupId.value
  if (groupId == null) return []
  const query = searchQuery.value.trim().toLowerCase()

  return catalog.value.models.flatMap(model => {
    if (query && !model.name.toLowerCase().includes(query) && !model.platform.toLowerCase().includes(query)) {
      return []
    }
    const groupPrice = model.prices.find(price => price.group_id === groupId)
    if (!groupPrice) return []
    return [{ model, price: groupPrice[selectedTier.value] }]
  })
})

const longContextRule = computed(() => {
  const model = visibleModels.value.find(entry => entry.model.long_context_threshold > 0)?.model
  if (!model) return ''
  return longContextDescription(model)
})

async function loadCatalog() {
  abortController?.abort()
  const controller = new AbortController()
  abortController = controller
  loading.value = true
  loadError.value = ''
  try {
    const result = await modelPlazaAPI.getCatalog({ signal: controller.signal })
    if (controller.signal.aborted) return
    catalog.value = result
    if (!result.groups.some(group => group.id === selectedGroupId.value)) {
      selectedGroupId.value = result.groups[0]?.id ?? null
    }
  } catch (error: unknown) {
    const requestError = error as { name?: string; code?: string }
    if (requestError.name === 'AbortError' || requestError.code === 'ERR_CANCELED') return
    loadError.value = extractApiErrorMessage(error, t('modelPlaza.loadError'))
  } finally {
    if (abortController === controller) {
      loading.value = false
      abortController = null
    }
  }
}

function formatRate(value: number): string {
  return `${Number(value.toFixed(4))}x`
}

function formatPrice(value: number): string {
  if (!Number.isFinite(value)) return '$0.00'
  const absolute = Math.abs(value)
  const digits = absolute >= 10 ? 2 : absolute >= 1 ? 3 : absolute >= 0.1 ? 4 : 5
  return `$${value.toLocaleString(undefined, {
    minimumFractionDigits: 2,
    maximumFractionDigits: digits,
  })}`
}

function formatContext(value: number): string {
  if (value >= 1_000_000) return `${Number((value / 1_000_000).toFixed(1))}M`
  if (value >= 1_000) return `${Number((value / 1_000).toFixed(1))}K`
  return String(value)
}

function longContextDescription(model: ModelPlazaModel): string {
  return t('modelPlaza.longContextRule', {
    threshold: formatContext(model.long_context_threshold),
    inputRate: formatRate(model.long_context_input_multiplier),
    outputRate: formatRate(model.long_context_output_multiplier),
  })
}

onMounted(loadCatalog)
onBeforeUnmount(() => abortController?.abort())
</script>
