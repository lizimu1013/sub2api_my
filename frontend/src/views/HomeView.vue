<template>
  <!-- Custom Home Content: Full Page Mode -->
  <div v-if="homeContent" class="min-h-screen">
    <!-- iframe mode -->
    <iframe
      v-if="isHomeContentUrl"
      :src="homeContent.trim()"
      class="h-screen w-full border-0"
      allowfullscreen
    ></iframe>
    <!-- HTML mode - SECURITY: homeContent is admin-only setting, XSS risk is acceptable -->
    <div v-else v-html="homeContent"></div>
  </div>

  <!-- Default Home Page - Claude / Anthropic style -->
  <div
    v-else
    ref="rootEl"
    class="home-claude relative min-h-screen bg-cream text-ink antialiased transition-colors duration-300 dark:bg-night-900 dark:text-cream"
  >
    <!-- Header -->
    <header
      class="sticky top-0 z-50 transition-all duration-300"
      :class="
        scrolled
          ? 'border-b border-cream-deep bg-cream/80 backdrop-blur-md dark:border-night-800 dark:bg-night-900/80'
          : ''
      "
    >
      <nav class="mx-auto flex max-w-5xl items-center justify-between px-6 py-5">
        <!-- Logo -->
        <div class="flex items-center gap-2.5">
          <div v-if="siteLogo" class="h-8 w-8 overflow-hidden rounded-lg">
            <img :src="siteLogo" alt="Logo" class="h-full w-full object-contain" />
          </div>
          <svg v-else class="h-6 w-6 text-clay-500" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
            <path
              d="M12 1.5l1.6 6.3 4.8-4.2-2.6 5.9 6.4-1.1-5.4 3.6 5.4 3.6-6.4-1.1 2.6 5.9-4.8-4.2L12 22.5l-1.6-6.3-4.8 4.2 2.6-5.9-6.4 1.1 5.4-3.6L1.8 8.4l6.4 1.1L5.6 3.6l4.8 4.2L12 1.5z"
            />
          </svg>
          <span class="font-serif text-lg font-semibold tracking-tight">{{ siteName }}</span>
        </div>

        <div class="hidden items-center gap-9 text-sm text-ink-soft md:flex dark:text-night-300">
          <a href="#pain" class="link-underline">{{ t('home.navLinks.pain') }}</a>
          <a href="#features" class="link-underline">{{ t('home.navLinks.features') }}</a>
          <a href="#compare" class="link-underline">{{ t('home.navLinks.compare') }}</a>
          <a href="#providers" class="link-underline">{{ t('home.navLinks.providers') }}</a>
        </div>

        <!-- Nav Actions -->
        <div class="flex items-center gap-2">
          <LocaleSwitcher />

          <!-- Doc Link -->
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="rounded-lg p-2 text-ink-mute transition-colors hover:bg-cream-deep hover:text-ink dark:text-night-400 dark:hover:bg-night-800 dark:hover:text-cream"
            :title="t('home.viewDocs')"
          >
            <Icon name="book" size="md" />
          </a>

          <!-- Theme Toggle -->
          <button
            @click="toggleTheme"
            class="rounded-lg p-2 text-ink-mute transition-colors hover:bg-cream-deep hover:text-ink dark:text-night-400 dark:hover:bg-night-800 dark:hover:text-cream"
            :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
          >
            <Icon v-if="isDark" name="sun" size="md" />
            <Icon v-else name="moon" size="md" />
          </button>

          <!-- Login / Dashboard -->
          <router-link
            v-if="isAuthenticated"
            :to="dashboardPath"
            class="inline-flex items-center gap-1.5 rounded-full bg-ink py-1 pl-1 pr-2.5 text-cream-paper transition-all hover:bg-ink-soft dark:bg-cream dark:text-ink dark:hover:bg-white"
          >
            <span
              class="flex h-5 w-5 items-center justify-center rounded-full bg-clay-500 text-[10px] font-semibold text-white"
            >
              {{ userInitial }}
            </span>
            <span class="text-xs font-medium">{{ t('home.dashboard') }}</span>
          </router-link>
          <router-link
            v-else
            to="/login"
            class="inline-flex items-center rounded-full bg-ink px-4 py-1.5 text-sm font-medium text-cream-paper transition-all hover:bg-ink-soft dark:bg-cream dark:text-ink dark:hover:bg-white"
          >
            {{ t('home.login') }}
          </router-link>
        </div>
      </nav>
    </header>

    <main class="relative z-10">
      <!-- Hero -->
      <section class="px-6 pb-24 pt-12 md:pt-20">
        <div class="mx-auto max-w-5xl">
          <div class="flex flex-col items-center gap-16 lg:flex-row lg:gap-14">
            <!-- Left -->
            <div class="flex-1 text-center lg:text-left">
              <div
                class="mb-7 inline-flex items-center gap-2 rounded-full border border-clay-200 bg-clay-50 px-3.5 py-1.5 text-xs font-medium text-clay-700 dark:border-clay-700/50 dark:bg-clay-900/20 dark:text-clay-300"
              >
                <span class="spark">✳</span> {{ t('home.heroBadge') }}
              </div>

              <h1
                class="mb-6 font-serif text-5xl font-semibold leading-[1.08] tracking-tight md:text-6xl lg:text-[4.25rem]"
              >
                {{ t('home.heroTitleLine1') }}<br />
                {{ t('home.heroTitleLine2') }}<span class="text-clay-500">{{ t('home.heroTitleAccent') }}</span>
              </h1>
              <p class="mx-auto mb-9 max-w-lg text-lg leading-relaxed text-ink-mute dark:text-night-300 lg:mx-0">
                {{ t('home.heroDescription') }}
              </p>

              <div class="flex flex-col items-center gap-3 sm:flex-row lg:justify-start">
                <router-link
                  :to="isAuthenticated ? dashboardPath : '/login'"
                  class="group inline-flex w-full items-center justify-center gap-2 rounded-xl bg-clay-500 px-7 py-3.5 text-base font-medium text-white shadow-sm transition-all hover:bg-clay-600 hover:shadow-md sm:w-auto"
                >
                  {{ isAuthenticated ? t('home.goToDashboard') : t('home.getStarted') }}
                  <Icon name="arrowRight" size="sm" :stroke-width="2" class="transition-transform group-hover:translate-x-1" />
                </router-link>
                <button
                  type="button"
                  @click="scrollToId('features')"
                  class="inline-flex w-full items-center justify-center rounded-xl border border-ink/15 bg-transparent px-7 py-3.5 text-base font-medium text-ink transition-all hover:border-ink/30 hover:bg-cream-deep sm:w-auto dark:border-cream/20 dark:text-cream dark:hover:bg-night-800"
                >
                  {{ t('home.learnMore') }}
                </button>
              </div>

              <!-- Stats -->
              <div class="mt-12 flex items-center justify-center gap-10 lg:justify-start">
                <div>
                  <div class="font-serif text-3xl font-semibold">4+</div>
                  <div class="mt-0.5 text-xs text-ink-mute dark:text-night-400">{{ t('home.stats.models') }}</div>
                </div>
                <div class="h-9 w-px bg-ink/10 dark:bg-cream/10"></div>
                <div>
                  <div class="font-serif text-3xl font-semibold">99.9<span class="text-xl">%</span></div>
                  <div class="mt-0.5 text-xs text-ink-mute dark:text-night-400">{{ t('home.stats.availability') }}</div>
                </div>
                <div class="h-9 w-px bg-ink/10 dark:bg-cream/10"></div>
                <div>
                  <div class="font-serif text-3xl font-semibold">{{ t('home.stats.billingValue') }}</div>
                  <div class="mt-0.5 text-xs text-ink-mute dark:text-night-400">{{ t('home.stats.billingLabel') }}</div>
                </div>
              </div>
            </div>

            <!-- Right: warm terminal -->
            <div class="flex flex-1 justify-center lg:justify-end">
              <div class="terminal-card">
                <div class="terminal-bar">
                  <span class="tdot bg-clay-500"></span>
                  <span class="tdot" style="background: #e0a387"></span>
                  <span class="tdot" style="background: #edc7b4"></span>
                  <span class="ml-1 flex-1 text-center font-mono text-xs text-ink-mute dark:text-night-400" style="margin-right: 42px">muchuapi — bash</span>
                </div>
                <div class="terminal-body text-ink-soft dark:text-night-300">
                  <div class="cl d1">
                    <span class="font-bold text-clay-500">❯</span>
                    <span class="text-clay-600 dark:text-clay-300">curl</span>
                    <span class="text-ink-mute dark:text-night-400">-X POST</span>
                    <span class="font-medium text-emerald-700 dark:text-emerald-400">/v1/messages</span>
                  </div>
                  <div class="cl d2">
                    <span class="italic text-ink-mute dark:text-night-400"># {{ t('home.terminalRouting') }}</span>
                  </div>
                  <div class="cl d3">
                    <span class="rounded-md bg-emerald-600/10 px-2 py-0.5 font-semibold text-emerald-700 dark:bg-emerald-400/10 dark:text-emerald-400">200 OK</span>
                    <span class="text-clay-600 dark:text-clay-300">{ "content": "Hello!" }</span>
                  </div>
                  <div class="cl d4">
                    <span class="font-bold text-clay-500">❯</span>
                    <span class="caret"></span>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- Trust pills -->
          <div class="mt-20 flex flex-wrap items-center justify-center gap-x-8 gap-y-3 text-sm text-ink-mute dark:text-night-400">
            <span class="flex items-center gap-2"><span class="spark">✳</span> {{ t('home.tags.subscriptionToApi') }}</span>
            <span class="hidden h-1 w-1 rounded-full bg-ink/20 sm:block dark:bg-cream/20"></span>
            <span class="flex items-center gap-2"><span class="spark">✳</span> {{ t('home.tags.stickySession') }}</span>
            <span class="hidden h-1 w-1 rounded-full bg-ink/20 sm:block dark:bg-cream/20"></span>
            <span class="flex items-center gap-2"><span class="spark">✳</span> {{ t('home.tags.realtimeBilling') }}</span>
            <span class="hidden h-1 w-1 rounded-full bg-ink/20 sm:block dark:bg-cream/20"></span>
            <span class="flex items-center gap-2"><span class="spark">✳</span> {{ t('home.tags.loadBalancing') }}</span>
          </div>
        </div>
      </section>

      <!-- Pain Points -->
      <section id="pain" class="px-6 py-24">
        <div class="mx-auto max-w-5xl">
          <div class="reveal mb-14 max-w-2xl">
            <p class="mb-3 text-sm font-medium uppercase tracking-widest text-clay-500">
              {{ t('home.painPoints.kicker') }}
            </p>
            <h2 class="font-serif text-3xl font-semibold leading-tight md:text-4xl">{{ t('home.painPoints.title') }}</h2>
            <p class="mt-4 text-lg text-ink-mute dark:text-night-300">{{ t('home.painPoints.description') }}</p>
          </div>
          <div class="grid gap-5 sm:grid-cols-2 lg:grid-cols-4">
            <div
              v-for="(key, i) in painKeys"
              :key="key"
              class="reveal card rounded-2xl p-6"
            >
              <div class="mb-4 font-serif text-2xl text-clay-500">{{ String(i + 1).padStart(2, '0') }}</div>
              <h3 class="mb-2 font-medium">{{ t(`home.painPoints.items.${key}.title`) }}</h3>
              <p class="text-sm leading-relaxed text-ink-mute dark:text-night-400">{{ t(`home.painPoints.items.${key}.desc`) }}</p>
            </div>
          </div>
        </div>
      </section>

      <!-- Features / Solutions -->
      <section id="features" class="px-6 py-24">
        <div class="mx-auto max-w-5xl">
          <div class="reveal mb-14 max-w-2xl">
            <p class="mb-3 text-sm font-medium uppercase tracking-widest text-clay-500">{{ t('home.solutions.title') }}</p>
            <h2 class="font-serif text-3xl font-semibold leading-tight md:text-4xl">{{ t('home.solutions.subtitle') }}</h2>
          </div>
          <div class="grid gap-6 md:grid-cols-3">
            <!-- One-Click Access -->
            <div class="reveal card rounded-2xl p-8">
              <div class="mb-6 flex h-12 w-12 items-center justify-center rounded-xl bg-clay-100 text-clay-600 dark:bg-clay-900/30 dark:text-clay-300">
                <svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.6">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 5.25a3 3 0 013 3m3 0a6 6 0 01-7.029 5.912c-.563-.097-1.159.026-1.563.43L10.5 17.25H8.25v2.25H6v2.25H2.25v-2.818c0-.597.237-1.17.659-1.591l6.499-6.499c.404-.404.527-1 .43-1.563A6 6 0 1121.75 8.25z"/>
                </svg>
              </div>
              <h3 class="mb-2 font-serif text-xl font-semibold">{{ t('home.features.unifiedGateway') }}</h3>
              <p class="text-sm leading-relaxed text-ink-mute dark:text-night-400">{{ t('home.features.unifiedGatewayDesc') }}</p>
            </div>
            <!-- Reliable -->
            <div class="reveal card rounded-2xl p-8">
              <div class="mb-6 flex h-12 w-12 items-center justify-center rounded-xl bg-clay-100 text-clay-600 dark:bg-clay-900/30 dark:text-clay-300">
                <svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.6">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12c0 1.268-.63 2.39-1.593 3.068a3.745 3.745 0 01-1.043 3.296 3.745 3.745 0 01-3.296 1.043A3.745 3.745 0 0112 21c-1.268 0-2.39-.63-3.068-1.593a3.746 3.746 0 01-3.296-1.043 3.745 3.745 0 01-1.043-3.296A3.745 3.745 0 013 12c0-1.268.63-2.39 1.593-3.068a3.745 3.745 0 011.043-3.296 3.746 3.746 0 013.296-1.043A3.746 3.746 0 0112 3c1.268 0 2.39.63 3.068 1.593a3.746 3.746 0 013.296 1.043 3.746 3.746 0 011.043 3.296A3.745 3.745 0 0121 12z"/>
                </svg>
              </div>
              <h3 class="mb-2 font-serif text-xl font-semibold">{{ t('home.features.multiAccount') }}</h3>
              <p class="text-sm leading-relaxed text-ink-mute dark:text-night-400">{{ t('home.features.multiAccountDesc') }}</p>
            </div>
            <!-- Pay as you go -->
            <div class="reveal card rounded-2xl p-8">
              <div class="mb-6 flex h-12 w-12 items-center justify-center rounded-xl bg-clay-100 text-clay-600 dark:bg-clay-900/30 dark:text-clay-300">
                <svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.6">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M2.25 18.75a60.07 60.07 0 0115.797 2.101c.727.198 1.453-.342 1.453-1.096V18.75M3.75 4.5v.75A.75.75 0 013 6h-.75m0 0v-.375c0-.621.504-1.125 1.125-1.125H20.25M2.25 6v9m18-10.5v.75c0 .414.336.75.75.75h.75m-1.5-1.5h.375c.621 0 1.125.504 1.125 1.125v9.75c0 .621-.504 1.125-1.125 1.125h-.375m1.5-1.5H21a.75.75 0 00-.75.75v.75m0 0H3.75m0 0h-.375a1.125 1.125 0 01-1.125-1.125V15m1.5 1.5v-.75A.75.75 0 003 15h-.75M15 10.5a3 3 0 11-6 0 3 3 0 016 0zm3 0h.008v.008H18V10.5zm-12 0h.008v.008H6V10.5z"/>
                </svg>
              </div>
              <h3 class="mb-2 font-serif text-xl font-semibold">{{ t('home.features.balanceQuota') }}</h3>
              <p class="text-sm leading-relaxed text-ink-mute dark:text-night-400">{{ t('home.features.balanceQuotaDesc') }}</p>
            </div>
          </div>
        </div>
      </section>

      <!-- Comparison -->
      <section id="compare" class="px-6 py-24">
        <div class="mx-auto max-w-3xl">
          <div class="reveal mb-12 text-center">
            <h2 class="font-serif text-3xl font-semibold md:text-4xl">{{ t('home.comparison.title') }}</h2>
          </div>
          <div class="reveal overflow-hidden rounded-2xl border border-cream-deep bg-cream-paper dark:border-night-700 dark:bg-night-850">
            <table class="w-full text-left text-sm">
              <thead>
                <tr class="border-b border-cream-deep dark:border-night-700">
                  <th class="px-6 py-5 font-medium text-ink-mute dark:text-night-400">{{ t('home.comparison.headers.feature') }}</th>
                  <th class="px-6 py-5 font-medium text-ink-mute dark:text-night-400">{{ t('home.comparison.headers.official') }}</th>
                  <th class="bg-clay-50 px-6 py-5 font-serif font-semibold text-clay-700 dark:bg-clay-900/15 dark:text-clay-300">{{ t('home.comparison.headers.us') }}</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-cream-deep dark:divide-night-700/60">
                <tr v-for="key in comparisonKeys" :key="key">
                  <td class="px-6 py-4 font-medium">{{ t(`home.comparison.items.${key}.feature`) }}</td>
                  <td class="px-6 py-4 text-ink-mute dark:text-night-400">{{ t(`home.comparison.items.${key}.official`) }}</td>
                  <td class="bg-clay-50/60 px-6 py-4 dark:bg-clay-900/10">{{ t(`home.comparison.items.${key}.us`) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </section>

      <!-- Providers -->
      <section id="providers" class="px-6 py-24">
        <div class="mx-auto max-w-5xl">
          <div class="reveal mb-12 text-center">
            <h2 class="font-serif text-3xl font-semibold md:text-4xl">{{ t('home.providers.title') }}</h2>
            <p class="mt-3 text-lg text-ink-mute dark:text-night-300">{{ t('home.providers.description') }}</p>
          </div>
          <div class="reveal flex flex-wrap items-center justify-center gap-4">
            <div
              v-for="p in providers"
              :key="p.label"
              class="flex items-center gap-2.5 rounded-xl border border-cream-deep bg-cream-paper px-5 py-3 dark:border-night-700 dark:bg-night-850"
            >
              <div class="flex h-8 w-8 items-center justify-center rounded-lg text-xs font-bold text-white" :class="p.badge">
                {{ p.glyph }}
              </div>
              <span class="text-sm font-medium">{{ p.label }}</span>
              <span class="rounded bg-clay-100 px-1.5 py-0.5 text-[10px] font-medium text-clay-600 dark:bg-clay-900/30 dark:text-clay-300">{{ t('home.providers.supported') }}</span>
            </div>
            <!-- More - Soon -->
            <div class="flex items-center gap-2.5 rounded-xl border border-dashed border-cream-deep bg-transparent px-5 py-3 opacity-70 dark:border-night-700">
              <div class="flex h-8 w-8 items-center justify-center rounded-lg bg-ink-mute/30 text-xs font-bold text-ink-mute dark:text-night-400">+</div>
              <span class="text-sm font-medium">{{ t('home.providers.more') }}</span>
              <span class="rounded bg-cream-deep px-1.5 py-0.5 text-[10px] font-medium text-ink-mute dark:bg-night-800 dark:text-night-400">{{ t('home.providers.soon') }}</span>
            </div>
          </div>
        </div>
      </section>

      <!-- CTA -->
      <section class="px-6 py-24">
        <div class="reveal mx-auto max-w-4xl rounded-3xl border border-clay-200 bg-clay-50 px-8 py-16 text-center dark:border-clay-800/40 dark:bg-clay-900/15">
          <div class="mb-4 text-2xl text-clay-500"><span class="spark">✳</span></div>
          <h2 class="mb-3 font-serif text-3xl font-semibold md:text-4xl">{{ t('home.cta.title') }}</h2>
          <p class="mx-auto mb-8 max-w-md text-lg text-ink-mute dark:text-night-300">{{ t('home.cta.description') }}</p>
          <router-link
            :to="isAuthenticated ? dashboardPath : '/login'"
            class="group inline-flex items-center gap-2 rounded-xl bg-clay-500 px-8 py-3.5 text-base font-medium text-white shadow-sm transition-all hover:bg-clay-600 hover:shadow-md"
          >
            {{ isAuthenticated ? t('home.goToDashboard') : t('home.cta.button') }}
            <Icon name="arrowRight" size="sm" :stroke-width="2" class="transition-transform group-hover:translate-x-1" />
          </router-link>
        </div>
      </section>
    </main>

    <!-- Footer -->
    <footer class="relative z-10 border-t border-cream-deep px-6 py-10 dark:border-night-800">
      <div class="mx-auto flex max-w-5xl flex-col items-center justify-between gap-4 text-center sm:flex-row sm:text-left">
        <div class="flex items-center gap-2 text-sm text-ink-mute dark:text-night-400">
          <span class="spark">✳</span> &copy; {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}
        </div>
        <div class="flex items-center gap-6 text-sm">
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="text-ink-mute transition-colors hover:text-ink dark:text-night-400 dark:hover:text-cream"
          >
            {{ t('home.docs') }}
          </a>
          <a
            :href="githubUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="text-ink-mute transition-colors hover:text-ink dark:text-night-400 dark:hover:text-cream"
          >
            GitHub
          </a>
        </div>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'
import { sanitizeUrl } from '@/utils/url'

const { t } = useI18n()

const authStore = useAuthStore()
const appStore = useAppStore()

// Site settings - directly from appStore (already initialized from injected config)
const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'MuchuAPI')
const siteLogo = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const docUrl = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.doc_url || appStore.docUrl || ''))
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')

// Check if homeContent is a URL (for iframe display)
const isHomeContentUrl = computed(() => {
  const content = homeContent.value.trim()
  return content.startsWith('http://') || content.startsWith('https://')
})

// Theme
const isDark = ref(document.documentElement.classList.contains('dark'))

// GitHub URL
const githubUrl = 'https://github.com/Wei-Shaw/sub2api'

// Auth state
const isAuthenticated = computed(() => authStore.isAuthenticated)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => (isAdmin.value ? '/admin/dashboard' : '/dashboard'))
const userInitial = computed(() => {
  const user = authStore.user
  if (!user || !user.email) return ''
  return user.email.charAt(0).toUpperCase()
})

// Current year for footer
const currentYear = computed(() => new Date().getFullYear())

// Static section keys (drive v-for over i18n)
const painKeys = ['expensive', 'complex', 'unstable', 'noControl'] as const
const comparisonKeys = ['pricing', 'models', 'management', 'stability', 'control'] as const
const providers = [
  { label: 'Claude', glyph: 'C', badge: 'bg-clay-500' },
  { label: 'GPT', glyph: 'G', badge: 'bg-emerald-600' },
  { label: 'Gemini', glyph: 'G', badge: 'bg-blue-600' },
  { label: 'Antigravity', glyph: 'A', badge: 'bg-rose-500' }
]

// Toggle theme
function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

// Initialize theme
function initTheme() {
  const savedTheme = localStorage.getItem('theme')
  if (savedTheme === 'dark' || (!savedTheme && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
    isDark.value = true
    document.documentElement.classList.add('dark')
  }
}

// Smooth-scroll to in-page section
function scrollToId(id: string) {
  document.getElementById(id)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

// Sticky header background on scroll
const scrolled = ref(false)
function onScroll() {
  scrolled.value = window.scrollY > 16
}

// Scroll-reveal animation
const rootEl = ref<HTMLElement | null>(null)
let observer: IntersectionObserver | null = null

onMounted(() => {
  initTheme()
  authStore.checkAuth()
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }

  window.addEventListener('scroll', onScroll, { passive: true })
  onScroll()

  nextTick(() => {
    if (!('IntersectionObserver' in window) || !rootEl.value) return
    observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry, i) => {
          if (entry.isIntersecting) {
            const el = entry.target as HTMLElement
            el.style.transitionDelay = `${(i % 4) * 80}ms`
            el.classList.add('visible')
            observer?.unobserve(el)
          }
        })
      },
      { threshold: 0.12 }
    )
    rootEl.value.querySelectorAll('.reveal').forEach((el) => observer?.observe(el))
  })
})

onUnmounted(() => {
  window.removeEventListener('scroll', onScroll)
  observer?.disconnect()
})
</script>

<style scoped>
/* Paper grain — very subtle */
.home-claude::before {
  content: '';
  position: fixed;
  inset: 0;
  z-index: 0;
  pointer-events: none;
  opacity: 0.4;
  background-image: radial-gradient(rgba(0, 0, 0, 0.025) 1px, transparent 1px);
  background-size: 4px 4px;
}
:global(.dark .home-claude::before) {
  opacity: 0.5;
  background-image: radial-gradient(rgba(255, 255, 255, 0.02) 1px, transparent 1px);
}

.spark {
  display: inline-block;
  color: #cc785c;
}

/* Terminal card */
.terminal-card {
  width: 100%;
  max-width: 26rem;
  background: #faf9f5;
  border: 1px solid #e8e4d8;
  border-radius: 18px;
  box-shadow:
    0 1px 2px rgba(20, 20, 19, 0.04),
    0 24px 48px -24px rgba(20, 20, 19, 0.18);
  overflow: hidden;
  transition:
    transform 0.4s cubic-bezier(0.16, 1, 0.3, 1),
    box-shadow 0.4s;
}
.terminal-card:hover {
  transform: translateY(-6px);
  box-shadow:
    0 1px 2px rgba(20, 20, 19, 0.04),
    0 36px 64px -28px rgba(204, 120, 92, 0.28);
}
:global(.dark .home-claude .terminal-card) {
  background: #262624;
  border-color: #3a3a36;
  box-shadow: 0 24px 48px -24px rgba(0, 0, 0, 0.6);
}

.terminal-bar {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  padding: 0.85rem 1.1rem;
  background: #f0eee6;
  border-bottom: 1px solid #e8e4d8;
}
:global(.dark .home-claude .terminal-bar) {
  background: #1f1e1d;
  border-bottom-color: #3a3a36;
}
.tdot {
  width: 11px;
  height: 11px;
  border-radius: 50%;
}

.terminal-body {
  padding: 1.25rem 1.35rem;
  font-family: ui-monospace, 'SF Mono', 'Fira Code', monospace;
  font-size: 13px;
  line-height: 2.05;
}
.cl {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
  opacity: 0;
  transform: translateY(6px);
  animation: cl-rise 0.55s cubic-bezier(0.16, 1, 0.3, 1) forwards;
}
.d1 {
  animation-delay: 0.35s;
}
.d2 {
  animation-delay: 1.05s;
}
.d3 {
  animation-delay: 1.8s;
}
.d4 {
  animation-delay: 2.55s;
}
@keyframes cl-rise {
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
.caret {
  display: inline-block;
  width: 7px;
  height: 15px;
  background: #cc785c;
  animation: caret-blink 1s step-end infinite;
}
@keyframes caret-blink {
  0%,
  50% {
    opacity: 1;
  }
  51%,
  100% {
    opacity: 0;
  }
}

/* Scroll reveal */
.reveal {
  opacity: 0;
  transform: translateY(22px);
  transition:
    opacity 0.7s cubic-bezier(0.16, 1, 0.3, 1),
    transform 0.7s cubic-bezier(0.16, 1, 0.3, 1);
}
.reveal.visible {
  opacity: 1;
  transform: translateY(0);
}

/* Cards */
.card {
  background: #faf9f5;
  border: 1px solid #e8e4d8;
  transition:
    transform 0.3s,
    box-shadow 0.3s,
    border-color 0.3s;
}
.card:hover {
  transform: translateY(-4px);
  box-shadow: 0 24px 48px -28px rgba(20, 20, 19, 0.2);
  border-color: #e0a387;
}
:global(.dark .home-claude .card) {
  background: #262624;
  border-color: #30302d;
}
:global(.dark .home-claude .card:hover) {
  border-color: #bd5d3a;
  box-shadow: 0 24px 48px -28px rgba(0, 0, 0, 0.6);
}

.link-underline {
  background-image: linear-gradient(#cc785c, #cc785c);
  background-position: 0 100%;
  background-repeat: no-repeat;
  background-size: 0% 1.5px;
  transition: background-size 0.3s;
}

.link-underline:hover {
  background-size: 100% 1.5px;
}

/* Respect reduced motion */
@media (prefers-reduced-motion: reduce) {
  .cl,
  .reveal,
  .caret {
    animation: none !important;
    transition: none !important;
    opacity: 1 !important;
    transform: none !important;
  }
}
</style>
