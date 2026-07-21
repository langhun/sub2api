<template>
  <main class="dino-page">
    <header class="dino-header">
      <span class="brand">DaiGua</span>
      <router-link class="account-link" :to="isAuthenticated ? dashboardPath : '/login'">
        {{ isAuthenticated ? '控制台' : '登录' }}
      </router-link>
    </header>
    <section class="interstitial-wrapper">
      <ChromiumDinoRunner />
    </section>
  </main>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import ChromiumDinoRunner from '@/components/game/ChromiumDinoRunner.vue'
import { useAuthStore } from '@/stores'

const authStore = useAuthStore()
const isAuthenticated = computed(() => authStore.isAuthenticated)
const dashboardPath = computed(() => authStore.isAdmin ? '/admin/dashboard' : '/dashboard')
</script>

<style>
.dino-page {
  background: #fff;
  min-height: 100vh;
  overflow: hidden;
  position: relative;
  transition: background-color 0.25s;
}

.dino-header {
  align-items: center;
  display: flex;
  font-family: "Microsoft YaHei UI", "Segoe UI", sans-serif;
  font-size: 14px;
  font-weight: 600;
  height: 64px;
  justify-content: space-between;
  padding: 0 32px;
  position: relative;
  z-index: 3;
}

.dino-header .brand,
.dino-header .account-link {
  align-items: center;
  color: #202124;
  display: inline-flex;
  font: inherit;
  height: 24px;
  line-height: 1;
  text-decoration: none;
}

.dino-header .brand {
  font-weight: 700;
}

.dino-header .account-link {
  border-bottom: 1px solid transparent;
  color: #5f6368;
  font-weight: 700;
}

.dino-header .account-link:hover,
.dino-header .account-link:focus-visible {
  border-bottom-color: currentColor;
  color: #202124;
  outline: 0;
}

.interstitial-wrapper {
  box-sizing: border-box;
  margin: 0 auto;
  max-width: 600px;
  min-height: 250px;
  padding-top: 100px;
  position: relative;
  width: 100%;
}

.arcade-mode .interstitial-wrapper {
  height: 100vh;
  max-width: 100%;
  overflow: hidden;
}

@media (max-width: 700px) {
  .dino-header {
    font-size: 14px;
    height: 56px;
    padding: 0 20px;
  }

  .interstitial-wrapper {
    width: calc(100% - 40px);
  }
}
</style>
