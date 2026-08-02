<template>
  <AppLayout>
    <div class="mx-auto max-w-[1600px] space-y-4">
      <div class="lg:hidden">
        <h1 class="text-xl font-semibold text-gray-900 dark:text-white">
          {{ t('modelPlaza.title') }}
        </h1>
        <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
          {{ t('modelPlaza.description') }}
        </p>
      </div>

      <section
        class="space-y-3 border-b border-gray-200 pb-5 dark:border-dark-700"
        :aria-label="t('modelPlaza.filtersLabel')"
      >
        <div class="filter-row">
          <span class="filter-label">{{ t('modelPlaza.platformLabel') }}</span>
          <div class="flex min-w-0 flex-wrap gap-2">
            <button
              type="button"
              class="filter-chip filter-chip-active"
              aria-pressed="true"
            >
              <PlatformIcon platform="openai" size="sm" />
              OpenAI
            </button>
          </div>
        </div>

        <div class="filter-row">
          <span class="filter-label">{{ t('modelPlaza.groupLabel') }}</span>
          <div
            v-if="catalog.groups.length > 0"
            class="flex min-w-0 flex-wrap gap-2"
            role="tablist"
            :aria-label="t('modelPlaza.groupLabel')"
          >
            <button
              v-for="group in catalog.groups"
              :key="group.id"
              type="button"
              class="filter-chip"
              :class="selectedGroupId === group.id ? 'filter-chip-active' : 'filter-chip-idle'"
              :aria-pressed="selectedGroupId === group.id"
              @click="selectedGroupId = group.id"
            >
              <span>{{ displayGroupName(group.name) }}</span>
              <span class="rate-badge" :class="selectedGroupId === group.id ? 'rate-badge-active' : ''">
                {{ formatRate(group.rate_multiplier) }}
              </span>
            </button>
          </div>
        </div>

        <div class="filter-row">
          <span class="filter-label">{{ t('modelPlaza.serviceTierLabel') }}</span>
          <div
            class="flex min-w-0 flex-wrap gap-2"
            role="group"
            :aria-label="t('modelPlaza.serviceTierLabel')"
          >
            <button
              v-for="tier in serviceTiers"
              :key="tier.value"
              type="button"
              class="filter-chip"
              :class="selectedTier === tier.value ? 'filter-chip-dark' : 'filter-chip-idle'"
              :aria-pressed="selectedTier === tier.value"
              @click="selectedTier = tier.value"
            >
              <Icon v-if="tier.value === 'fast'" name="bolt" size="xs" />
              {{ tier.label }}
            </button>
          </div>
        </div>

        <div class="filter-row items-start sm:items-center">
          <span class="filter-label pt-2.5 sm:pt-0">{{ t('modelPlaza.modelLabel') }}</span>
          <div class="flex min-w-0 flex-1 flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div class="relative w-full sm:max-w-xs">
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
            <div class="flex items-center gap-3 text-sm text-gray-500 dark:text-dark-400">
              <span>{{ t('modelPlaza.modelCount', { count: visibleModels.length }) }}</span>
              <span class="hidden sm:inline">{{ t('modelPlaza.priceUnit', { currency: catalog.currency || 'USD' }) }}</span>
              <button
                type="button"
                class="btn btn-secondary btn-icon h-10 w-10"
                :title="t('common.refresh')"
                :disabled="loading"
                @click="loadCatalog"
              >
                <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
              </button>
            </div>
          </div>
        </div>
      </section>

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

      <div v-if="loading && catalog.models.length === 0" class="overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
        <div class="h-16 animate-pulse border-b border-gray-100 bg-gray-50 dark:border-dark-700 dark:bg-dark-900/40"></div>
        <div v-for="index in 6" :key="index" class="h-16 animate-pulse border-b border-gray-100 last:border-0 dark:border-dark-700">
          <div class="mx-5 mt-6 h-3 w-2/3 rounded bg-gray-100 dark:bg-dark-700"></div>
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

      <section
        v-else
        class="overflow-hidden rounded-lg border border-emerald-200 bg-white shadow-sm dark:border-emerald-900/60 dark:bg-dark-800"
        :aria-label="t('modelPlaza.pricingTableLabel')"
      >
        <header class="flex flex-col gap-3 border-b border-gray-100 px-4 py-4 sm:flex-row sm:items-center sm:justify-between sm:px-5 dark:border-dark-700">
          <div class="flex min-w-0 items-center gap-3">
            <span class="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg border border-emerald-100 bg-emerald-50 text-emerald-700 dark:border-emerald-900/50 dark:bg-emerald-900/20 dark:text-emerald-300">
              <PlatformIcon platform="openai" size="lg" />
            </span>
            <div class="min-w-0">
              <div class="flex flex-wrap items-center gap-2">
                <h2 class="text-sm font-semibold text-gray-900 dark:text-white">
                  {{ displayGroupName(selectedGroup?.name ?? '') }}
                </h2>
                <span class="rounded-md bg-emerald-100 px-2 py-0.5 text-xs font-semibold text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300">
                  {{ formatRate(selectedGroup?.rate_multiplier ?? 0) }}
                </span>
                <span class="rounded-md bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-600 dark:bg-dark-700 dark:text-dark-300">
                  {{ selectedTierLabel }}
                </span>
              </div>
              <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">
                {{ t('modelPlaza.groupPriceDescription', { rate: formatRate(selectedGroup?.rate_multiplier ?? 0) }) }}
              </p>
            </div>
          </div>
          <div class="flex flex-wrap items-center gap-2 text-xs">
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
        </header>

        <div class="hidden overflow-x-auto lg:block">
          <table class="w-full min-w-[1180px] border-collapse text-left">
            <thead>
              <tr class="border-b border-gray-200 text-xs font-medium text-gray-500 dark:border-dark-700 dark:text-dark-400">
                <th rowspan="2" class="w-[17%] px-5 py-3">{{ t('modelPlaza.columns.model') }}</th>
                <th colspan="3" class="border-x border-emerald-100 bg-emerald-50/80 px-4 py-2 text-center font-semibold text-emerald-700 dark:border-emerald-900/40 dark:bg-emerald-900/10 dark:text-emerald-300">
                  {{ t('modelPlaza.actualPrice') }} · {{ t('modelPlaza.priceUnit', { currency: catalog.currency || 'USD' }) }}
                </th>
                <th colspan="3" class="px-4 py-2 text-center font-semibold">
                  {{ t('modelPlaza.basePrice') }} · {{ t('modelPlaza.priceUnit', { currency: catalog.currency || 'USD' }) }}
                </th>
                <th rowspan="2" class="w-24 px-4 py-3 text-right">{{ t('modelPlaza.discountRate') }}</th>
              </tr>
              <tr class="border-b border-gray-200 bg-gray-50/70 text-xs font-medium text-gray-500 dark:border-dark-700 dark:bg-dark-900/30 dark:text-dark-400">
                <th class="border-l border-emerald-100 bg-emerald-50/80 px-4 py-2 dark:border-emerald-900/40 dark:bg-emerald-900/10">{{ t('modelPlaza.columns.input') }}</th>
                <th class="bg-emerald-50/80 px-4 py-2 dark:bg-emerald-900/10">{{ t('modelPlaza.columns.output') }}</th>
                <th class="border-r border-emerald-100 bg-emerald-50/80 px-4 py-2 dark:border-emerald-900/40 dark:bg-emerald-900/10">{{ t('modelPlaza.columns.cache') }}</th>
                <th class="px-4 py-2">{{ t('modelPlaza.columns.input') }}</th>
                <th class="px-4 py-2">{{ t('modelPlaza.columns.output') }}</th>
                <th class="px-4 py-2">{{ t('modelPlaza.columns.cache') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="entry in visibleModels"
                :key="entry.model.platform + ':' + entry.model.name"
                class="border-b border-gray-100 last:border-b-0 hover:bg-gray-50/50 dark:border-dark-700/70 dark:hover:bg-dark-700/20"
              >
                <td class="px-5 py-4">
                  <div class="font-mono text-sm font-semibold text-gray-900 dark:text-white">
                    {{ entry.model.name }}
                  </div>
                  <div class="mt-1 text-xs uppercase text-gray-400">{{ entry.model.platform }}</div>
                </td>
                <td class="actual-price-cell border-l border-emerald-100 dark:border-emerald-900/40">
                  <PriceWithContext :base="entry.price.input" :model="entry.model" kind="input" />
                </td>
                <td class="actual-price-cell">
                  <PriceWithContext :base="entry.price.output" :model="entry.model" kind="output" />
                </td>
                <td class="actual-price-cell border-r border-emerald-100 dark:border-emerald-900/40">
                  <CachePrice :price="entry.price" :model="entry.model" />
                </td>
                <td class="base-price-cell">
                  <PriceWithContext :base="entry.base.input" :model="entry.model" kind="input" compact />
                </td>
                <td class="base-price-cell">
                  <PriceWithContext :base="entry.base.output" :model="entry.model" kind="output" compact />
                </td>
                <td class="base-price-cell">
                  <CachePrice :price="entry.base" :model="entry.model" compact />
                </td>
                <td class="px-4 py-4 text-right font-mono text-sm font-semibold text-gray-700 dark:text-dark-200">
                  {{ formatRate(selectedGroup?.rate_multiplier ?? 0) }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div class="divide-y divide-gray-100 lg:hidden dark:divide-dark-700">
          <article
            v-for="entry in visibleModels"
            :key="'mobile:' + entry.model.platform + ':' + entry.model.name"
            class="p-4"
          >
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <h3 class="break-all font-mono text-sm font-semibold text-gray-900 dark:text-white">
                  {{ entry.model.name }}
                </h3>
                <span class="mt-1 block text-xs uppercase text-gray-400">{{ entry.model.platform }}</span>
              </div>
              <span class="flex-shrink-0 rounded-md bg-emerald-100 px-2 py-1 font-mono text-xs font-semibold text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300">
                {{ formatRate(selectedGroup?.rate_multiplier ?? 0) }}
              </span>
            </div>

            <div class="mt-4 bg-emerald-50/70 px-3 py-3 dark:bg-emerald-900/10">
              <div class="mb-3 text-xs font-semibold text-emerald-700 dark:text-emerald-300">
                {{ t('modelPlaza.actualPrice') }}
              </div>
              <div class="grid grid-cols-2 gap-x-4 gap-y-4">
                <div>
                  <div class="price-label">{{ t('modelPlaza.columns.input') }}</div>
                  <PriceWithContext :base="entry.price.input" :model="entry.model" kind="input" />
                </div>
                <div>
                  <div class="price-label">{{ t('modelPlaza.columns.output') }}</div>
                  <PriceWithContext :base="entry.price.output" :model="entry.model" kind="output" />
                </div>
                <div class="col-span-2">
                  <div class="price-label">{{ t('modelPlaza.columns.cache') }}</div>
                  <CachePrice :price="entry.price" :model="entry.model" />
                </div>
              </div>
            </div>

            <div class="mt-3 grid grid-cols-3 gap-3 text-xs">
              <div>
                <div class="price-label">{{ t('modelPlaza.baseInput') }}</div>
                <div class="mt-1 font-mono font-semibold text-gray-700 dark:text-dark-200">{{ formatPrice(entry.base.input) }}</div>
              </div>
              <div>
                <div class="price-label">{{ t('modelPlaza.baseOutput') }}</div>
                <div class="mt-1 font-mono font-semibold text-gray-700 dark:text-dark-200">{{ formatPrice(entry.base.output) }}</div>
              </div>
              <div>
                <div class="price-label">{{ t('modelPlaza.baseCacheRead') }}</div>
                <div class="mt-1 font-mono font-semibold text-gray-700 dark:text-dark-200">{{ formatPrice(entry.base.cache_read) }}</div>
              </div>
            </div>
          </article>
        </div>
      </section>

      <div
        v-if="longContextRule"
        class="flex items-start gap-3 border-t border-amber-200 py-3 text-sm text-amber-900 dark:border-amber-900/50 dark:text-amber-200"
      >
        <Icon name="infoCircle" size="sm" class="mt-0.5 flex-shrink-0" />
        <span>{{ longContextRule }}</span>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onBeforeUnmount, onMounted, ref, type PropType } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import modelPlazaAPI, {
  type ModelPlazaCatalog,
  type ModelPlazaModel,
  type ModelPlazaTierPrice,
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

const selectedTierLabel = computed(() =>
  serviceTiers.value.find(tier => tier.value === selectedTier.value)?.label ?? '',
)

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
    return [{
      model,
      price: groupPrice[selectedTier.value],
      base: selectedTier.value === 'fast' ? model.base_fast : model.base_standard,
    }]
  })
})

const longContextRule = computed(() => {
  const model = visibleModels.value.find(entry => entry.model.long_context_threshold > 0)?.model
  if (!model) return ''
  return longContextDescription(model)
})

const PriceWithContext = defineComponent({
  props: {
    base: { type: Number, required: true },
    model: { type: Object as PropType<ModelPlazaModel>, required: true },
    kind: { type: String as PropType<'input' | 'output'>, required: true },
    compact: { type: Boolean, default: false },
  },
  setup(props) {
    return () => {
      const hasLongContext = props.model.long_context_threshold > 0
      const multiplier = props.kind === 'output'
        ? props.model.long_context_output_multiplier
        : props.model.long_context_input_multiplier
      if (!hasLongContext || props.compact) {
        return h('div', { class: 'font-mono text-sm font-semibold text-gray-800 dark:text-dark-100' }, formatPrice(props.base))
      }
      return h('div', { class: 'space-y-1' }, [
        priceLine(`≤${formatContext(props.model.long_context_threshold)}`, props.base),
        priceLine(`${formatContext(props.model.long_context_threshold)}–1M`, props.base * multiplier),
      ])
    }
  },
})

const CachePrice = defineComponent({
  props: {
    price: { type: Object as PropType<ModelPlazaTierPrice>, required: true },
    model: { type: Object as PropType<ModelPlazaModel>, required: true },
    compact: { type: Boolean, default: false },
  },
  setup(props) {
    return () => h('div', { class: 'space-y-1' }, [
      priceLine(t('modelPlaza.cacheWriteShort'), props.price.cache_write),
      priceLine(t('modelPlaza.cacheReadShort'), props.price.cache_read),
      ...(props.compact || props.model.long_context_threshold <= 0 ? [] : [
        h('div', { class: 'text-[11px] text-gray-400 dark:text-dark-500' },
          t('modelPlaza.longCachePrice', {
            price: formatPrice(props.price.cache_read * props.model.long_context_input_multiplier),
          })),
      ]),
    ])
  },
})

function priceLine(label: string, value: number) {
  return h('div', { class: 'flex items-baseline gap-2 whitespace-nowrap' }, [
    h('span', { class: 'text-[11px] text-gray-400 dark:text-dark-500' }, label),
    h('span', { class: 'font-mono text-sm font-semibold text-gray-800 dark:text-dark-100' }, formatPrice(value)),
  ])
}

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

function displayGroupName(name: string): string {
  const normalized = name.toLowerCase()
  if (normalized.includes('plus')) return t('modelPlaza.groups.plus')
  if (/(^|[-_\s])pro($|[-_\s])/.test(normalized)) return t('modelPlaza.groups.pro')
  return name
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

<style scoped>
.filter-row {
  @apply flex min-w-0 flex-col gap-2 sm:flex-row sm:items-center;
}

.filter-label {
  @apply w-14 flex-shrink-0 text-xs font-medium text-gray-500 dark:text-dark-400;
}

.filter-chip {
  @apply inline-flex min-h-8 items-center gap-1.5 rounded-md border px-3 py-1.5 text-xs font-medium transition-colors;
}

.filter-chip-active {
  @apply border-emerald-200 bg-emerald-100 text-emerald-700 dark:border-emerald-800 dark:bg-emerald-900/30 dark:text-emerald-300;
}

.filter-chip-dark {
  @apply border-gray-900 bg-gray-900 text-white dark:border-white dark:bg-white dark:text-dark-900;
}

.filter-chip-idle {
  @apply border-gray-200 bg-white text-gray-600 hover:border-gray-300 hover:bg-gray-50 dark:border-dark-700 dark:bg-dark-800 dark:text-dark-300 dark:hover:bg-dark-700;
}

.rate-badge {
  @apply rounded bg-gray-100 px-1.5 py-0.5 font-mono text-[10px] text-gray-500 dark:bg-dark-700 dark:text-dark-400;
}

.rate-badge-active {
  @apply bg-emerald-200/70 text-emerald-800 dark:bg-emerald-800/60 dark:text-emerald-200;
}

.actual-price-cell {
  @apply bg-emerald-50/60 px-4 py-4 align-top dark:bg-emerald-900/10;
}

.base-price-cell {
  @apply px-4 py-4 align-top;
}

.price-label {
  @apply text-[11px] font-medium text-gray-500 dark:text-dark-400;
}
</style>
