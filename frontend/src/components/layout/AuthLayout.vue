<template>
  <div
    class="relative flex min-h-[100svh] items-start justify-center overflow-x-hidden bg-[#f4f7fb] px-4 py-8 dark:bg-dark-950 sm:items-center sm:py-12"
  >
    <div class="relative z-10 w-full max-w-[420px]">
      <!-- Logo/Brand -->
      <div class="mb-6 text-center">
        <!-- Custom Logo or Default Logo -->
        <template v-if="settingsLoaded">
          <div
            class="mb-3 inline-flex h-16 w-16 items-center justify-center overflow-hidden rounded-lg border border-white bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900"
          >
            <img
              :src="siteLogo || '/logo.png'"
              :alt="`${siteName} Logo`"
              class="h-full w-full object-contain"
            />
          </div>
          <h1 class="mb-1 text-2xl font-semibold text-gray-950 dark:text-white">
            {{ siteName }}
          </h1>
          <p class="text-sm text-gray-600 dark:text-dark-300">
            {{ siteSubtitle }}
          </p>
        </template>
      </div>

      <!-- Card Container -->
      <div
        data-testid="auth-card"
        class="rounded-lg border border-gray-200/90 bg-white p-6 shadow-[0_16px_40px_rgba(14,39,71,0.09)] dark:border-dark-700 dark:bg-dark-900 sm:p-8"
      >
        <slot />
      </div>

      <!-- Footer Links -->
      <div class="mt-6 text-center text-sm">
        <slot name="footer" />
      </div>

      <nav
        v-if="legalDocuments.length > 0"
        aria-label="服务协议"
        class="mt-5 flex flex-wrap items-center justify-center gap-x-4 gap-y-2 text-xs"
      >
        <RouterLink
          v-for="doc in legalDocuments"
          :key="doc.id"
          :to="{ name: 'LegalDocument', params: { documentId: doc.id } }"
          target="_blank"
          rel="noopener noreferrer"
          class="text-gray-500 transition-colors hover:text-primary-700 dark:text-dark-400 dark:hover:text-primary-300"
        >
          {{ doc.title }}
        </RouterLink>
      </nav>

      <!-- Copyright -->
      <div class="mt-5 text-center text-xs text-gray-400 dark:text-dark-500">
        &copy; {{ currentYear }} {{ siteName }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useAppStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'

const appStore = useAppStore()

const siteName = computed(() => appStore.siteName || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const siteSubtitle = computed(
  () => appStore.cachedPublicSettings?.site_subtitle || '统一、安全、稳定的 AI API 服务'
)
const settingsLoaded = computed(() => appStore.publicSettingsLoaded)
const legalDocuments = computed(() => {
  const settings = appStore.cachedPublicSettings
  if (settings?.login_agreement_enabled !== true) {
    return []
  }
  return (settings.login_agreement_documents ?? []).filter(
    (doc) => doc.id?.trim() && doc.title?.trim()
  )
})

const currentYear = computed(() => new Date().getFullYear())

onMounted(() => {
  appStore.fetchPublicSettings()
})
</script>
