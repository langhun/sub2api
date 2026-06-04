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
    <!-- HTML mode -->
    <div v-else v-html="sanitizedHomeContent"></div>
  </div>

  <!-- Default Home Page -->
  <div
    v-else
    class="relative flex min-h-screen flex-col overflow-hidden bg-[var(--background)] text-[var(--foreground)]"
  >
    <!-- Header -->
    <header class="relative z-20 px-6 py-4">
      <nav class="mx-auto flex max-w-6xl items-center justify-between">
        <!-- Logo -->
        <div class="flex items-center">
          <div
            class="h-10 w-10 overflow-hidden rounded-lg border border-[var(--border)] bg-[var(--card)] shadow-sm"
          >
            <img :src="siteLogo || '/logo.png'" alt="Logo" class="h-full w-full object-contain" />
          </div>
        </div>

        <!-- Nav Actions -->
        <div class="flex items-center gap-3">
          <!-- Language Switcher -->
          <LocaleSwitcher />

          <!-- Doc Link -->
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="rounded-full p-2 text-[var(--muted-foreground)] transition-colors hover:bg-[var(--muted)] hover:text-[var(--foreground)]"
            :title="t('home.viewDocs')"
          >
            <Icon name="book" size="md" />
          </a>

          <!-- Theme Toggle -->
          <button
            @click="toggleTheme"
            class="rounded-full p-2 text-[var(--muted-foreground)] transition-colors hover:bg-[var(--muted)] hover:text-[var(--foreground)]"
            :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
          >
            <Icon v-if="isDark" name="sun" size="md" />
            <Icon v-else name="moon" size="md" />
          </button>

          <!-- Login / Dashboard Button -->
          <router-link
            v-if="isAuthenticated"
            :to="dashboardPath"
            class="inline-flex items-center gap-1.5 rounded-full bg-[var(--primary)] py-1 pl-1 pr-2.5 text-[var(--primary-foreground)] transition-colors hover:opacity-90"
          >
            <span
              class="flex h-5 w-5 items-center justify-center rounded-full bg-[var(--primary-foreground)] text-[10px] font-semibold text-[var(--primary)]"
            >
              {{ userInitial }}
            </span>
            <span class="text-xs font-medium">{{ t('home.dashboard') }}</span>
            <svg
              class="h-3 w-3 opacity-70"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              stroke-width="2"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M4.5 19.5l15-15m0 0H8.25m11.25 0v11.25"
              />
            </svg>
          </router-link>
          <router-link
            v-else
            to="/login"
            class="inline-flex items-center rounded-full bg-[var(--primary)] px-3 py-1 text-xs font-medium text-[var(--primary-foreground)] transition-colors hover:opacity-90"
          >
            {{ t('home.login') }}
          </router-link>
        </div>
      </nav>
    </header>

    <!-- Main Content -->
    <main class="relative z-10 flex-1 px-6 py-16">
      <div class="mx-auto max-w-6xl">
        <!-- Hero Section - Left/Right Layout -->
        <div class="mb-12 flex flex-col items-center justify-between gap-12 lg:flex-row lg:gap-16">
          <!-- Left: Text Content -->
          <div class="flex-1 text-center lg:text-left">
            <h1
              class="mb-4 text-4xl font-bold text-[var(--foreground)] md:text-5xl lg:text-6xl"
            >
              {{ siteName }}
            </h1>
            <p class="mb-8 text-lg text-[var(--muted-foreground)] md:text-xl">
              {{ siteSubtitle }}
            </p>

            <div class="mb-6 inline-flex items-center rounded-lg border border-emerald-500/40 bg-emerald-500/10 px-3 py-1.5 text-sm font-semibold text-emerald-700 dark:text-emerald-300">
              实时更新测试：{{ liveUpdateMarker }}
            </div>

            <!-- CTA Button -->
            <div>
              <router-link
                :to="isAuthenticated ? dashboardPath : '/login'"
                class="btn btn-primary h-12 rounded-xl px-8 text-base shadow-card"
              >
                {{ isAuthenticated ? t('home.goToDashboard') : t('home.getStarted') }}
                <Icon name="arrowRight" size="md" class="ml-2" :stroke-width="2" />
              </router-link>
            </div>
          </div>

          <!-- Right: Terminal Animation -->
          <div class="flex flex-1 justify-center lg:justify-end">
            <div class="terminal-container">
              <div class="terminal-window">
                <!-- Window header -->
                <div class="terminal-header">
                  <div class="terminal-buttons">
                    <span class="btn-close"></span>
                    <span class="btn-minimize"></span>
                    <span class="btn-maximize"></span>
                  </div>
                  <span class="terminal-title">terminal</span>
                </div>
                <!-- Terminal content -->
                <div class="terminal-body">
                  <div class="code-line line-1">
                    <span class="code-prompt">$</span>
                    <span class="code-cmd">curl</span>
                    <span class="code-flag">-X POST</span>
                    <span class="code-url">/v1/messages</span>
                  </div>
                  <div class="code-line line-2">
                    <span class="code-comment"># Routing to upstream...</span>
                  </div>
                  <div class="code-line line-3">
                    <span class="code-success">200 OK</span>
                    <span class="code-response">{ "content": "Hello!" }</span>
                  </div>
                  <div class="code-line line-4">
                    <span class="code-prompt">$</span>
                    <span class="cursor"></span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Feature Tags - Centered -->
        <div class="mb-12 flex flex-wrap items-center justify-center gap-4 md:gap-6">
          <div
            class="inline-flex items-center gap-2.5 rounded-lg border border-[var(--border)] bg-[var(--card)] px-3 py-2 shadow-sm"
          >
            <Icon name="swap" size="sm" class="text-[var(--muted-foreground)]" />
            <span class="text-sm font-medium text-[var(--foreground)]">{{
              t('home.tags.subscriptionToApi')
            }}</span>
          </div>
          <div
            class="inline-flex items-center gap-2.5 rounded-lg border border-[var(--border)] bg-[var(--card)] px-3 py-2 shadow-sm"
          >
            <Icon name="shield" size="sm" class="text-[var(--muted-foreground)]" />
            <span class="text-sm font-medium text-[var(--foreground)]">{{
              t('home.tags.stickySession')
            }}</span>
          </div>
          <div
            class="inline-flex items-center gap-2.5 rounded-lg border border-[var(--border)] bg-[var(--card)] px-3 py-2 shadow-sm"
          >
            <Icon name="chart" size="sm" class="text-[var(--muted-foreground)]" />
            <span class="text-sm font-medium text-[var(--foreground)]">{{
              t('home.tags.realtimeBilling')
            }}</span>
          </div>
        </div>

        <!-- Features Grid -->
        <div class="mb-12 grid gap-6 md:grid-cols-3">
          <!-- Feature 1: Unified Gateway -->
          <div
            class="card card-hover group p-6"
          >
            <div
              class="mb-4 flex h-12 w-12 items-center justify-center rounded-lg border border-[var(--border)] bg-[var(--muted)] text-[var(--foreground)] transition-colors group-hover:bg-[var(--accent)]"
            >
              <Icon name="server" size="lg" />
            </div>
            <h3 class="mb-2 text-lg font-semibold text-[var(--foreground)]">
              {{ t('home.features.unifiedGateway') }}
            </h3>
            <p class="text-sm leading-relaxed text-[var(--muted-foreground)]">
              {{ t('home.features.unifiedGatewayDesc') }}
            </p>
          </div>

          <!-- Feature 2: Account Pool -->
          <div
            class="card card-hover group p-6"
          >
            <div
              class="mb-4 flex h-12 w-12 items-center justify-center rounded-lg border border-[var(--border)] bg-[var(--muted)] text-[var(--foreground)] transition-colors group-hover:bg-[var(--accent)]"
            >
              <svg
                class="h-6 w-6"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                stroke-width="1.5"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  d="M18 18.72a9.094 9.094 0 003.741-.479 3 3 0 00-4.682-2.72m.94 3.198l.001.031c0 .225-.012.447-.037.666A11.944 11.944 0 0112 21c-2.17 0-4.207-.576-5.963-1.584A6.062 6.062 0 016 18.719m12 0a5.971 5.971 0 00-.941-3.197m0 0A5.995 5.995 0 0012 12.75a5.995 5.995 0 00-5.058 2.772m0 0a3 3 0 00-4.681 2.72 8.986 8.986 0 003.74.477m.94-3.197a5.971 5.971 0 00-.94 3.197M15 6.75a3 3 0 11-6 0 3 3 0 016 0zm6 3a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0zm-13.5 0a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0z"
                />
              </svg>
            </div>
            <h3 class="mb-2 text-lg font-semibold text-[var(--foreground)]">
              {{ t('home.features.multiAccount') }}
            </h3>
            <p class="text-sm leading-relaxed text-[var(--muted-foreground)]">
              {{ t('home.features.multiAccountDesc') }}
            </p>
          </div>

          <!-- Feature 3: Billing & Quota -->
          <div
            class="card card-hover group p-6"
          >
            <div
              class="mb-4 flex h-12 w-12 items-center justify-center rounded-lg border border-[var(--border)] bg-[var(--muted)] text-[var(--foreground)] transition-colors group-hover:bg-[var(--accent)]"
            >
              <svg
                class="h-6 w-6"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                stroke-width="1.5"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  d="M2.25 18.75a60.07 60.07 0 0115.797 2.101c.727.198 1.453-.342 1.453-1.096V18.75M3.75 4.5v.75A.75.75 0 013 6h-.75m0 0v-.375c0-.621.504-1.125 1.125-1.125H20.25M2.25 6v9m18-10.5v.75c0 .414.336.75.75.75h.75m-1.5-1.5h.375c.621 0 1.125.504 1.125 1.125v9.75c0 .621-.504 1.125-1.125 1.125h-.375m1.5-1.5H21a.75.75 0 00-.75.75v.75m0 0H3.75m0 0h-.375a1.125 1.125 0 01-1.125-1.125V15m1.5 1.5v-.75A.75.75 0 003 15h-.75M15 10.5a3 3 0 11-6 0 3 3 0 016 0zm3 0h.008v.008H18V10.5zm-12 0h.008v.008H6V10.5z"
                />
              </svg>
            </div>
            <h3 class="mb-2 text-lg font-semibold text-[var(--foreground)]">
              {{ t('home.features.balanceQuota') }}
            </h3>
            <p class="text-sm leading-relaxed text-[var(--muted-foreground)]">
              {{ t('home.features.balanceQuotaDesc') }}
            </p>
          </div>
        </div>

        <!-- Supported Providers -->
        <div class="mb-8 text-center">
          <h2 class="mb-3 text-2xl font-bold text-[var(--foreground)]">
            {{ t('home.providers.title') }}
          </h2>
          <p class="text-sm text-[var(--muted-foreground)]">
            {{ t('home.providers.description') }}
          </p>
        </div>

        <div class="mb-16 flex flex-wrap items-center justify-center gap-4">
          <!-- Claude - Supported -->
          <div
            class="flex items-center gap-2 rounded-lg border border-[var(--border)] bg-[var(--card)] px-4 py-2.5 shadow-sm"
          >
            <div
              class="flex h-8 w-8 items-center justify-center rounded-lg bg-[color-mix(in_oklch,var(--primary)_14%,transparent)] text-[var(--primary)]"
            >
              <span class="text-xs font-bold">C</span>
            </div>
            <span class="text-sm font-medium text-[var(--foreground)]">{{ t('home.providers.claude') }}</span>
            <span
              class="badge badge-gray rounded px-1.5 py-0.5 text-[10px]"
              >{{ t('home.providers.supported') }}</span
            >
          </div>
          <!-- GPT - Supported -->
          <div
            class="flex items-center gap-2 rounded-lg border border-[var(--border)] bg-[var(--card)] px-4 py-2.5 shadow-sm"
          >
            <div
              class="flex h-8 w-8 items-center justify-center rounded-lg bg-[color-mix(in_oklch,var(--primary)_14%,transparent)] text-[var(--primary)]"
            >
              <span class="text-xs font-bold">G</span>
            </div>
            <span class="text-sm font-medium text-[var(--foreground)]">GPT</span>
            <span
              class="badge badge-gray rounded px-1.5 py-0.5 text-[10px]"
              >{{ t('home.providers.supported') }}</span
            >
          </div>
          <!-- Gemini - Supported -->
          <div
            class="flex items-center gap-2 rounded-lg border border-[var(--border)] bg-[var(--card)] px-4 py-2.5 shadow-sm"
          >
            <div
              class="flex h-8 w-8 items-center justify-center rounded-lg bg-[color-mix(in_oklch,var(--primary)_14%,transparent)] text-[var(--primary)]"
            >
              <span class="text-xs font-bold">G</span>
            </div>
            <span class="text-sm font-medium text-[var(--foreground)]">{{ t('home.providers.gemini') }}</span>
            <span
              class="badge badge-gray rounded px-1.5 py-0.5 text-[10px]"
              >{{ t('home.providers.supported') }}</span
            >
          </div>
          <!-- Antigravity - Supported -->
          <div
            class="flex items-center gap-2 rounded-lg border border-[var(--border)] bg-[var(--card)] px-4 py-2.5 shadow-sm"
          >
            <div
              class="flex h-8 w-8 items-center justify-center rounded-lg bg-[color-mix(in_oklch,var(--primary)_14%,transparent)] text-[var(--primary)]"
            >
              <span class="text-xs font-bold">A</span>
            </div>
            <span class="text-sm font-medium text-[var(--foreground)]">{{ t('home.providers.antigravity') }}</span>
            <span
              class="badge badge-gray rounded px-1.5 py-0.5 text-[10px]"
              >{{ t('home.providers.supported') }}</span
            >
          </div>
          <!-- More - Coming Soon -->
          <div
            class="flex items-center gap-2 rounded-lg border border-[var(--border)] bg-[var(--card)] px-4 py-2.5 shadow-sm opacity-60"
          >
            <div
              class="flex h-8 w-8 items-center justify-center rounded-lg bg-[color-mix(in_oklch,var(--primary)_14%,transparent)] text-[var(--primary)]"
            >
              <span class="text-xs font-bold">+</span>
            </div>
            <span class="text-sm font-medium text-[var(--foreground)]">{{ t('home.providers.more') }}</span>
            <span
              class="badge badge-gray rounded px-1.5 py-0.5 text-[10px] opacity-80"
              >{{ t('home.providers.soon') }}</span
            >
          </div>
        </div>
      </div>
    </main>

    <!-- Footer -->
    <footer class="relative z-10 border-t border-[var(--border)] px-6 py-8">
      <div
        class="mx-auto flex max-w-6xl flex-col items-center justify-center gap-4 text-center sm:flex-row sm:text-left"
      >
        <p class="text-sm text-[var(--muted-foreground)]">
          &copy; {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}
        </p>
        <div class="flex items-center gap-4">
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="text-sm text-[var(--muted-foreground)] transition-colors hover:text-[var(--foreground)]"
          >
            {{ t('home.docs') }}
          </a>
          <a
            :href="githubUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="text-sm text-[var(--muted-foreground)] transition-colors hover:text-[var(--foreground)]"
          >
            GitHub
          </a>
        </div>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import DOMPurify from 'dompurify'
import { useAuthStore, useAppStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()

const authStore = useAuthStore()
const appStore = useAppStore()

// Site settings - directly from appStore (already initialized from injected config)
const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API')
const siteLogo = computed(() => appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '')
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || 'AI API Gateway Platform')
const docUrl = computed(() => appStore.cachedPublicSettings?.doc_url || appStore.docUrl || '')
const liveUpdateMarker = '3001-HMR-OK'
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')
const sanitizedHomeContent = computed(() => {
  const raw = homeContent.value
  if (!raw) return ''
  return DOMPurify.sanitize(raw)
})

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
const dashboardPath = computed(() => isAdmin.value ? '/admin/dashboard' : '/dashboard')
const userInitial = computed(() => {
  const user = authStore.user
  if (!user || !user.email) return ''
  return user.email.charAt(0).toUpperCase()
})

// Current year for footer
const currentYear = computed(() => new Date().getFullYear())

// Toggle theme
function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

// Initialize theme
function initTheme() {
  const savedTheme = localStorage.getItem('theme')
  if (
    savedTheme === 'dark' ||
    (!savedTheme && window.matchMedia('(prefers-color-scheme: dark)').matches)
  ) {
    isDark.value = true
    document.documentElement.classList.add('dark')
  }
}

onMounted(() => {
  initTheme()

  // Check auth state
  authStore.checkAuth()

  // Ensure public settings are loaded (will use cache if already loaded from injected config)
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
})
</script>

<style scoped>
/* Terminal Container */
.terminal-container {
  position: relative;
  display: inline-block;
}

/* Terminal Window */
.terminal-window {
  width: 420px;
  background: var(--card);
  border-radius: 12px;
  border: 1px solid var(--border);
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.04);
  overflow: hidden;
  transform: none;
  transition: transform 0.3s ease;
}

.terminal-window:hover {
  transform: translateY(-1px);
}

/* Terminal Header */
.terminal-header {
  display: flex;
  align-items: center;
  padding: 12px 16px;
  background: var(--muted);
  border-bottom: 1px solid var(--border);
}

.terminal-buttons {
  display: flex;
  gap: 8px;
}

.terminal-buttons span {
  width: 12px;
  height: 12px;
  border-radius: 50%;
}

.btn-close {
  background: #9ca3af;
}
.btn-minimize {
  background: #6b7280;
}
.btn-maximize {
  background: #4b5563;
}

.terminal-title {
  flex: 1;
  text-align: center;
  font-size: 12px;
  font-family: ui-monospace, monospace;
  color: #64748b;
  margin-right: 52px;
}

/* Terminal Body */
.terminal-body {
  padding: 20px 24px;
  font-family: ui-monospace, 'Fira Code', monospace;
  font-size: 14px;
  line-height: 2;
}

.code-line {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  opacity: 0;
  animation: line-appear 0.5s ease forwards;
}

.line-1 {
  animation-delay: 0.3s;
}
.line-2 {
  animation-delay: 1s;
}
.line-3 {
  animation-delay: 1.8s;
}
.line-4 {
  animation-delay: 2.5s;
}

@keyframes line-appear {
  from {
    opacity: 0;
    transform: translateY(5px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.code-prompt {
  color: #9ca3af;
  font-weight: bold;
}
.code-cmd {
  color: #e5e7eb;
}
.code-flag {
  color: #d1d5db;
}
.code-url {
  color: #171717;
}
.code-comment {
  color: #64748b;
  font-style: italic;
}
.code-success {
  color: #e5e7eb;
  background: rgba(229, 231, 235, 0.12);
  padding: 2px 8px;
  border-radius: 4px;
  font-weight: 600;
}
.code-response {
  color: #d1d5db;
}

/* Blinking Cursor */
.cursor {
  display: inline-block;
  width: 8px;
  height: 16px;
  background: #d1d5db;
  animation: blink 1s step-end infinite;
}

@keyframes blink {
  0%,
  50% {
    opacity: 1;
  }
  51%,
  100% {
    opacity: 0;
  }
}

/* Dark mode adjustments */
:deep(.dark) .terminal-window {
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.24);
}
</style>
