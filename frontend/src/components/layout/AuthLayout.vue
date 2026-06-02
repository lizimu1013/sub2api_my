<template>
  <div
    class="auth-claude relative flex min-h-screen items-center justify-center overflow-hidden bg-cream p-4 text-ink transition-colors duration-300 dark:bg-night-900 dark:text-cream"
  >
    <!-- Decorative warm blobs -->
    <div class="pointer-events-none absolute inset-0 overflow-hidden">
      <div class="absolute -right-32 -top-32 h-80 w-80 rounded-full bg-clay-300/20 blur-3xl dark:bg-clay-700/15"></div>
      <div class="absolute -bottom-32 -left-32 h-80 w-80 rounded-full bg-clay-400/15 blur-3xl dark:bg-clay-800/15"></div>
    </div>

    <!-- Content Container -->
    <div class="relative z-10 w-full max-w-md">
      <!-- Logo / Brand -->
      <div class="mb-8 text-center">
        <template v-if="settingsLoaded">
          <div
            v-if="siteLogo"
            class="mb-4 inline-flex h-16 w-16 items-center justify-center overflow-hidden rounded-2xl bg-cream-paper shadow-sm ring-1 ring-cream-deep dark:bg-night-850 dark:ring-night-700"
          >
            <img :src="siteLogo" alt="Logo" class="h-full w-full object-contain" />
          </div>
          <div
            v-else
            class="mb-4 inline-flex h-16 w-16 items-center justify-center rounded-2xl bg-clay-100 text-clay-600 shadow-sm dark:bg-clay-900/30 dark:text-clay-300"
          >
            <svg class="h-9 w-9" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
              <path
                d="M12 1.5l1.6 6.3 4.8-4.2-2.6 5.9 6.4-1.1-5.4 3.6 5.4 3.6-6.4-1.1 2.6 5.9-4.8-4.2L12 22.5l-1.6-6.3-4.8 4.2 2.6-5.9-6.4 1.1 5.4-3.6L1.8 8.4l6.4 1.1L5.6 3.6l4.8 4.2L12 1.5z"
              />
            </svg>
          </div>
          <h1 class="mb-2 font-serif text-3xl font-semibold tracking-tight text-ink dark:text-cream">
            {{ siteName }}
          </h1>
          <p class="text-sm text-ink-mute dark:text-night-400">
            {{ siteSubtitle }}
          </p>
        </template>
      </div>

      <!-- Card -->
      <div
        class="rounded-2xl border border-cream-deep bg-cream-paper p-8 shadow-[0_1px_2px_rgba(20,20,19,0.04),0_24px_48px_-28px_rgba(20,20,19,0.18)] dark:border-night-700 dark:bg-night-850 dark:shadow-[0_24px_48px_-28px_rgba(0,0,0,0.6)]"
      >
        <slot />
      </div>

      <!-- Footer Links -->
      <div class="mt-6 text-center text-sm">
        <slot name="footer" />
      </div>

      <!-- Copyright -->
      <div class="mt-8 flex items-center justify-center gap-2 text-center text-xs text-ink-mute dark:text-night-400">
        <span class="text-clay-500">✳</span>
        &copy; {{ currentYear }} {{ siteName }}. All rights reserved.
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useAppStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'

const appStore = useAppStore()

const siteName = computed(() => appStore.siteName || 'MuchuAPI')
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || 'Subscription to API Conversion Platform')
const settingsLoaded = computed(() => appStore.publicSettingsLoaded)

const currentYear = computed(() => new Date().getFullYear())

onMounted(() => {
  appStore.fetchPublicSettings()
})
</script>

<style scoped>
/* Subtle paper grain — matches the landing page */
.auth-claude::before {
  content: '';
  position: absolute;
  inset: 0;
  z-index: 0;
  pointer-events: none;
  opacity: 0.4;
  background-image: radial-gradient(rgba(0, 0, 0, 0.025) 1px, transparent 1px);
  background-size: 4px 4px;
}
:global(.dark) .auth-claude::before {
  opacity: 0.5;
  background-image: radial-gradient(rgba(255, 255, 255, 0.02) 1px, transparent 1px);
}
</style>
