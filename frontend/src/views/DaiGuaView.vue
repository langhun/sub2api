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
  font: 500 14px Arial, sans-serif;
  height: 64px;
  justify-content: space-between;
  padding: 0 32px;
  position: relative;
  z-index: 3;
}

.dino-header .brand,
.dino-header .account-link {
  color: #202124;
  font: 500 14px Arial, sans-serif;
  text-decoration: none;
}

.dino-header .account-link { color: #5f6368; }

.dino-header .account-link:hover,
.dino-header .account-link:focus-visible {
  color: #202124;
  text-decoration: underline;
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
    font-size: 13px;
    height: 56px;
    padding: 0 20px;
  }

  .interstitial-wrapper {
    width: calc(100% - 40px);
  }
}
</style>
