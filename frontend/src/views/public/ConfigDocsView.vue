<template>
  <div class="min-h-screen bg-gray-50 text-gray-950 dark:bg-dark-950 dark:text-white">
    <header class="glass border-b border-gray-200/70 dark:border-dark-700/70">
      <div class="mx-auto flex h-16 max-w-6xl items-center justify-between gap-4 px-4 sm:px-6">
        <RouterLink to="/home" class="flex min-w-0 items-center gap-3">
          <span class="flex h-10 w-10 flex-shrink-0 items-center justify-center overflow-hidden rounded-lg bg-white ring-1 ring-gray-200 dark:bg-dark-800 dark:ring-dark-700">
            <img :src="siteLogo || '/logo.png'" alt="Logo" class="h-full w-full object-contain" />
          </span>
          <span class="min-w-0">
            <span class="block truncate text-base font-bold text-gray-950 dark:text-white">{{ siteName }}</span>
            <span class="hidden truncate text-xs text-gray-500 dark:text-dark-400 sm:block">AI API 接入与配置中心</span>
          </span>
        </RouterLink>

        <nav class="flex items-center gap-1 sm:gap-2" aria-label="文档导航">
          <RouterLink
            to="/docs"
            class="hidden items-center gap-1.5 rounded-md px-3 py-2 text-sm font-medium text-gray-600 transition hover:bg-gray-100 hover:text-gray-950 dark:text-dark-300 dark:hover:bg-dark-800 dark:hover:text-white sm:flex"
          >
            <Icon name="book" size="sm" />
            文档
          </RouterLink>
          <button
            type="button"
            class="flex h-9 w-9 items-center justify-center rounded-md text-gray-500 transition hover:bg-gray-100 hover:text-gray-900 dark:text-dark-400 dark:hover:bg-dark-800 dark:hover:text-white"
            :title="isDark ? '切换为浅色主题' : '切换为深色主题'"
            :aria-label="isDark ? '切换为浅色主题' : '切换为深色主题'"
            @click="toggleTheme"
          >
            <Icon :name="isDark ? 'sun' : 'moon'" size="sm" />
          </button>
          <RouterLink
            :to="dashboardPath"
            class="inline-flex min-h-9 items-center justify-center rounded-lg bg-primary-600 px-3 py-2 text-sm font-semibold text-white shadow-sm shadow-primary-600/20 transition hover:bg-primary-700 sm:px-4"
          >
            {{ isAuthenticated ? '进入控制台' : '登录控制台' }}
          </RouterLink>
        </nav>
      </div>
    </header>

    <main v-if="!guideId" class="mx-auto max-w-6xl px-4 py-8 sm:px-6 lg:py-10">
      <section class="grid gap-8 rounded-xl border border-gray-100 bg-white p-6 shadow-sm dark:border-dark-700/70 dark:bg-dark-800/50 sm:p-8 lg:grid-cols-[1.15fr_0.85fr] lg:gap-12">
        <div class="flex flex-col justify-center">
          <p class="text-xs font-bold uppercase text-primary-600 dark:text-primary-400">Configuration Guides</p>
          <h1 class="mt-3 text-3xl font-bold text-gray-950 dark:text-white sm:text-4xl">配置文档</h1>
          <p class="mt-4 max-w-2xl text-sm leading-7 text-gray-600 dark:text-dark-200 sm:text-base">
            从桌面应用、命令行、IDE 插件到 SDK 调用，把常用客户端统一连接到悦享 API。
          </p>
        </div>

        <div class="grid gap-3 sm:grid-cols-2">
          <button
            v-for="parameter in connectionParameters"
            :key="parameter.label"
            type="button"
            class="group min-h-[98px] rounded-xl border border-gray-100 bg-gray-50 p-4 text-left transition hover:border-primary-200 hover:bg-primary-50/60 dark:border-dark-700/70 dark:bg-dark-800/70 dark:hover:border-primary-700 dark:hover:bg-primary-900/20"
            :title="`复制${parameter.label}`"
            @click="copyValue(parameter.value, `parameter-${parameter.label}`)"
          >
            <span class="flex items-center justify-between gap-2 text-xs font-medium text-gray-500 dark:text-dark-400">
              {{ parameter.label }}
              <Icon
                :name="copiedId === `parameter-${parameter.label}` ? 'check' : 'copy'"
                size="xs"
                class="text-gray-400 group-hover:text-primary-600 dark:group-hover:text-primary-400"
              />
            </span>
            <code class="mt-2 block break-all font-mono text-sm font-semibold leading-6 text-gray-950 dark:text-white">{{ parameter.value }}</code>
          </button>
        </div>
      </section>

      <section class="mt-5 grid gap-3 md:grid-cols-2" aria-label="接入准备">
        <div
          v-for="(step, index) in preparationSteps"
          :key="step.title"
          class="flex min-h-[108px] gap-4 rounded-xl border border-gray-100 bg-white p-4 shadow-sm dark:border-dark-700/70 dark:bg-dark-800/50"
        >
          <span class="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full bg-primary-600 text-sm font-bold text-white shadow-sm shadow-primary-600/20">
            {{ index + 1 }}
          </span>
          <div>
            <h2 class="text-base font-bold text-gray-950 dark:text-white">{{ step.title }}</h2>
            <p class="mt-1.5 text-sm leading-6 text-gray-600 dark:text-dark-300">{{ step.description }}</p>
          </div>
        </div>
      </section>

      <section class="mt-9" aria-labelledby="tutorials-title">
        <p class="text-xs font-bold text-primary-600 dark:text-primary-400">{{ configGuides.length }} 篇教程</p>
        <h2 id="tutorials-title" class="mt-2 text-2xl font-bold text-gray-950 dark:text-white">Codex 安装与配置</h2>
        <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-dark-300">桌面端、命令行、IDE 插件和直接 API 调用的接入方式。</p>

        <div class="mt-5 grid gap-3 md:grid-cols-2 lg:grid-cols-3">
          <RouterLink
            v-for="guide in configGuides"
            :key="guide.id"
            :to="`/docs/${guide.id}`"
            class="group flex min-h-[226px] flex-col rounded-xl border border-gray-100 bg-white p-5 shadow-sm transition hover:-translate-y-0.5 hover:border-primary-200 hover:shadow-md dark:border-dark-700/70 dark:bg-dark-800/50 dark:hover:border-primary-700"
          >
            <div class="flex items-center justify-between gap-3">
              <span class="inline-flex items-center gap-1.5 rounded-full border border-gray-200 bg-gray-50 px-2.5 py-1 text-xs font-semibold text-gray-600 dark:border-dark-600 dark:bg-dark-700 dark:text-dark-200">
                <Icon :name="guide.icon" size="xs" />
                {{ guide.badge }}
              </span>
              <span class="text-xs font-medium text-gray-500 dark:text-dark-400">{{ guide.minutes }} 分钟</span>
            </div>
            <h3 class="mt-5 text-lg font-bold text-gray-950 dark:text-white">{{ guide.title }}</h3>
            <p class="mt-2 flex-1 text-sm leading-6 text-gray-600 dark:text-dark-300">{{ guide.summary }}</p>
            <span class="mt-5 inline-flex items-center gap-2 text-sm font-semibold text-primary-600 dark:text-primary-400">
              查看教程
              <Icon name="arrowRight" size="sm" class="transition group-hover:translate-x-0.5" />
            </span>
          </RouterLink>
        </div>
      </section>

      <section class="mt-10 border-t border-gray-200 pt-8 dark:border-dark-800" aria-labelledby="faq-title">
        <div class="grid gap-6 lg:grid-cols-[0.65fr_1.35fr]">
          <div>
            <p class="text-xs font-bold text-primary-600 dark:text-primary-400">Troubleshooting</p>
            <h2 id="faq-title" class="mt-2 text-2xl font-bold text-gray-950 dark:text-white">常见问题</h2>
          </div>
          <div class="divide-y divide-gray-200 border-y border-gray-200 dark:divide-dark-700 dark:border-dark-700">
            <details v-for="faq in configDocFaqs" :key="faq.question" class="group py-4">
              <summary class="flex cursor-pointer list-none items-center justify-between gap-4 text-sm font-semibold text-gray-950 dark:text-white">
                {{ faq.question }}
                <Icon name="chevronDown" size="sm" class="flex-shrink-0 transition group-open:rotate-180" />
              </summary>
              <p class="mt-3 pr-8 text-sm leading-6 text-gray-600 dark:text-dark-300">{{ faq.answer }}</p>
            </details>
          </div>
        </div>
      </section>
    </main>

    <main v-else-if="currentGuide" class="mx-auto max-w-6xl px-4 py-7 sm:px-6 lg:py-9">
      <RouterLink to="/docs" class="inline-flex items-center gap-2 text-sm font-medium text-gray-600 transition hover:text-primary-600 dark:text-dark-300 dark:hover:text-primary-400">
        <Icon name="arrowLeft" size="sm" />
        返回配置文档
      </RouterLink>

      <section class="mt-5 border-b border-gray-200 pb-7 dark:border-dark-800">
        <div class="flex flex-wrap items-center gap-3 text-xs font-semibold text-gray-500 dark:text-dark-400">
          <span class="inline-flex items-center gap-1.5 rounded-full border border-gray-200 bg-white px-2.5 py-1 shadow-sm dark:border-dark-700 dark:bg-dark-800">
            <Icon :name="currentGuide.icon" size="xs" />
            {{ currentGuide.badge }}
          </span>
          <span>{{ currentGuide.minutes }} 分钟</span>
        </div>
        <h1 class="mt-4 text-3xl font-bold text-gray-950 dark:text-white sm:text-4xl">{{ currentGuide.title }}</h1>
        <p class="mt-3 max-w-3xl text-base leading-7 text-gray-600 dark:text-dark-200">{{ currentGuide.summary }}</p>
      </section>

      <div class="mt-7 grid gap-8 lg:grid-cols-[220px_minmax(0,1fr)] lg:gap-12">
        <aside class="lg:sticky lg:top-6 lg:self-start">
          <p class="mb-3 text-xs font-bold uppercase text-gray-500 dark:text-dark-400">本页目录</p>
          <nav class="flex gap-2 overflow-x-auto pb-2 lg:flex-col lg:overflow-visible lg:pb-0" aria-label="本页目录">
            <a
              v-for="section in currentGuide.sections"
              :key="section.id"
              :href="`#${section.id}`"
              class="flex-shrink-0 rounded-lg px-3 py-2 text-sm text-gray-600 transition hover:bg-white hover:text-primary-600 hover:shadow-sm dark:text-dark-300 dark:hover:bg-dark-800 dark:hover:text-primary-400"
            >
              {{ section.title }}
            </a>
          </nav>
          <div class="mt-5 hidden rounded-md border border-primary-200 bg-primary-50 p-4 text-xs leading-5 text-primary-800 dark:border-primary-800 dark:bg-primary-950/30 dark:text-primary-200 lg:block">
            示例中的 API Key 是占位符，使用前请替换为自己的密钥。
          </div>
        </aside>

        <article class="min-w-0 space-y-9">
          <section
            v-for="section in currentGuide.sections"
            :id="section.id"
            :key="section.id"
            class="scroll-mt-6"
          >
            <h2 class="text-xl font-bold text-gray-950 dark:text-white">{{ section.title }}</h2>
            <p v-if="section.description" class="mt-3 text-sm leading-7 text-gray-600 dark:text-dark-200">{{ section.description }}</p>

            <ul v-if="section.steps" class="mt-4 space-y-2">
              <li v-for="step in section.steps" :key="step" class="flex gap-3 text-sm leading-6 text-gray-700 dark:text-dark-200">
                <Icon name="checkCircle" size="sm" class="mt-1 flex-shrink-0 text-primary-600 dark:text-primary-400" />
                <span class="break-words">{{ step }}</span>
              </li>
            </ul>

            <div v-if="section.codeBlocks" class="mt-4 space-y-4">
              <div
                v-for="(block, blockIndex) in section.codeBlocks"
                :key="`${section.id}-${blockIndex}`"
                class="overflow-hidden rounded-md border border-gray-800 bg-[#090d12]"
              >
                <div class="flex h-10 items-center justify-between border-b border-gray-800 bg-[#111820] px-3">
                  <span class="font-mono text-xs text-gray-400">{{ block.label }}</span>
                  <button
                    type="button"
                    class="flex h-8 w-8 items-center justify-center rounded-md text-gray-400 transition hover:bg-gray-800 hover:text-white"
                    :title="copiedId === `${section.id}-${blockIndex}` ? '已复制' : '复制代码'"
                    :aria-label="copiedId === `${section.id}-${blockIndex}` ? '已复制' : '复制代码'"
                    @click="copyValue(block.code, `${section.id}-${blockIndex}`)"
                  >
                    <Icon :name="copiedId === `${section.id}-${blockIndex}` ? 'check' : 'copy'" size="sm" />
                  </button>
                </div>
                <pre class="overflow-x-auto p-4 text-sm leading-6 text-gray-100"><code :class="`language-${block.language}`">{{ block.code }}</code></pre>
              </div>
            </div>

            <div v-if="section.note" class="mt-4 flex gap-3 rounded-md border border-amber-200 bg-amber-50 p-4 text-sm leading-6 text-amber-900 dark:border-amber-800/70 dark:bg-amber-950/30 dark:text-amber-100">
              <Icon name="exclamationTriangle" size="sm" class="mt-1 flex-shrink-0" />
              <span>{{ section.note }}</span>
            </div>
          </section>

          <section class="border-t border-gray-200 pt-8 dark:border-dark-800">
            <h2 class="text-xl font-bold text-gray-950 dark:text-white">连接参数检查</h2>
            <div class="mt-4 overflow-x-auto rounded-xl border border-gray-100 bg-white shadow-sm dark:border-dark-700/70 dark:bg-dark-800/50">
              <table class="w-full min-w-[560px] text-left text-sm">
                <thead class="bg-gray-50 text-xs text-gray-500 dark:bg-dark-800 dark:text-dark-400">
                  <tr>
                    <th class="px-4 py-3 font-semibold">字段</th>
                    <th class="px-4 py-3 font-semibold">正确值</th>
                    <th class="px-4 py-3 font-semibold">常见错误</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-gray-200 dark:divide-dark-700">
                  <tr v-for="row in checklistRows" :key="row.field">
                    <td class="px-4 py-3 font-medium text-gray-950 dark:text-white">{{ row.field }}</td>
                    <td class="break-all px-4 py-3 font-mono text-xs text-primary-700 dark:text-primary-300">{{ row.value }}</td>
                    <td class="px-4 py-3 text-gray-600 dark:text-dark-300">{{ row.error }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </section>

          <div class="flex flex-col gap-3 border-t border-gray-200 pt-7 dark:border-dark-800 sm:flex-row sm:items-center sm:justify-between">
            <RouterLink to="/docs" class="inline-flex items-center gap-2 text-sm font-semibold text-gray-700 hover:text-primary-600 dark:text-dark-200 dark:hover:text-primary-400">
              <Icon name="arrowLeft" size="sm" />
              所有配置教程
            </RouterLink>
            <RouterLink to="/keys" class="inline-flex min-h-10 items-center justify-center gap-2 rounded-md bg-primary-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-primary-700">
              创建 API Key
              <Icon name="arrowRight" size="sm" />
            </RouterLink>
          </div>
        </article>
      </div>
    </main>

    <main v-else class="mx-auto max-w-3xl px-4 py-20 text-center sm:px-6">
      <Icon name="document" size="xl" class="mx-auto text-gray-400" />
      <h1 class="mt-4 text-2xl font-bold text-gray-950 dark:text-white">教程不存在</h1>
      <p class="mt-2 text-sm text-gray-600 dark:text-dark-300">该链接可能已更新，请返回配置文档首页重新选择。</p>
      <RouterLink to="/docs" class="mt-6 inline-flex items-center gap-2 rounded-md bg-primary-600 px-4 py-2 text-sm font-semibold text-white hover:bg-primary-700">
        <Icon name="arrowLeft" size="sm" />
        返回配置文档
      </RouterLink>
    </main>

    <footer class="mt-10 border-t border-gray-200 py-6 dark:border-dark-800">
      <div class="mx-auto flex max-w-6xl flex-col gap-2 px-4 text-xs text-gray-500 dark:text-dark-400 sm:flex-row sm:items-center sm:justify-between sm:px-6">
        <span>© {{ currentYear }} {{ siteName }}</span>
        <span>请妥善保管 API Key，不要在公开渠道发送完整密钥。</span>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import Icon from '@/components/icons/Icon.vue'
import { useClipboard } from '@/composables/useClipboard'
import {
  CONFIG_DOCS_API_KEY,
  CONFIG_DOCS_BASE_URL,
  CONFIG_DOCS_MODEL,
  CONFIG_DOCS_RESPONSES_URL,
  configDocFaqs,
  configGuides,
  getConfigGuide,
} from '@/config/configDocs'
import { useAppStore, useAuthStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'

const route = useRoute()
const appStore = useAppStore()
const authStore = useAuthStore()
const { copyToClipboard } = useClipboard()

const isDark = ref(document.documentElement.classList.contains('dark'))
const copiedId = ref('')
let copiedTimer: ReturnType<typeof setTimeout> | null = null

const guideId = computed(() => String(route.params.guideId || ''))
const currentGuide = computed(() => getConfigGuide(guideId.value))
const siteName = computed(() => {
  const configuredName = appStore.cachedPublicSettings?.site_name || appStore.siteName
  return configuredName && configuredName !== 'Sub2API' ? configuredName : '悦享 API'
})
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const isAuthenticated = computed(() => authStore.isAuthenticated)
const dashboardPath = computed(() => {
  if (!isAuthenticated.value) return '/login'
  return authStore.isAdmin ? '/admin/dashboard' : '/dashboard'
})
const currentYear = new Date().getFullYear()

const connectionParameters = [
  { label: 'Base URL', value: CONFIG_DOCS_BASE_URL },
  { label: 'API Key', value: CONFIG_DOCS_API_KEY },
  { label: '测试模型', value: CONFIG_DOCS_MODEL },
  { label: 'Responses API', value: CONFIG_DOCS_RESPONSES_URL },
]

const preparationSteps = [
  { title: '创建用户 API Key', description: '进入控制台 API Keys 页面创建或复制密钥，不要使用管理员登录凭据。' },
  { title: '填写 /v1 地址', description: `所有客户端统一填写 ${CONFIG_DOCS_BASE_URL}，避免重复追加 /v1。` },
  { title: '先发短消息测试', description: `首次接入建议调用 ${CONFIG_DOCS_MODEL}，确认成功后再用于正式任务。` },
  { title: '使用 Responses API', description: `完整请求地址为 ${CONFIG_DOCS_RESPONSES_URL}，认证方式为 Bearer API Key。` },
]

const checklistRows = [
  { field: 'Base URL', value: CONFIG_DOCS_BASE_URL, error: '重复写成 /v1/v1' },
  { field: 'API Key', value: CONFIG_DOCS_API_KEY, error: '缺少 Bearer 或密钥已停用' },
  { field: '模型', value: CONFIG_DOCS_MODEL, error: '模型名拼写不一致' },
  { field: '接口', value: '/v1/responses', error: '客户端仅支持旧式接口' },
]

async function copyValue(value: string, id: string) {
  const success = await copyToClipboard(value, '已复制')
  if (!success) return

  copiedId.value = id
  if (copiedTimer) clearTimeout(copiedTimer)
  copiedTimer = setTimeout(() => {
    copiedId.value = ''
  }, 1800)
}

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

function initTheme() {
  const savedTheme = localStorage.getItem('theme')
  if (savedTheme === 'dark' || (!savedTheme && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
    isDark.value = true
    document.documentElement.classList.add('dark')
  }
}

onMounted(() => {
  initTheme()
  authStore.checkAuth()
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
})
</script>
