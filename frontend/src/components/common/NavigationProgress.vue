<script setup lang="ts">
import { computed } from 'vue'
import { useNavigationLoadingState } from '@/composables/useNavigationLoading'

const { isLoading } = useNavigationLoadingState()
const isVisible = computed(() => isLoading.value)
</script>

<template>
  <Transition name="progress-fade">
    <div
      v-show="isVisible"
      class="navigation-progress"
      role="progressbar"
      aria-label="Loading"
      aria-valuenow="0"
      aria-valuemin="0"
      aria-valuemax="100"
    >
      <div class="navigation-progress-bar" />
    </div>
  </Transition>
</template>

<style scoped>
.navigation-progress {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  height: 4px;
  z-index: 9999;
  overflow: hidden;
  background: var(--muted);
  border-bottom: 1px solid var(--border);
}

.navigation-progress-bar {
  height: 100%;
  width: 100%;
  background: var(--primary);
  animation: progress-slide 1.5s ease-in-out infinite;
}

@keyframes progress-slide {
  0% {
    transform: translateX(-100%);
  }
  100% {
    transform: translateX(100%);
  }
}

.progress-fade-enter-active {
  transition: opacity 0.15s ease-out;
}

.progress-fade-leave-active {
  transition: opacity 0.3s ease-out;
}

.progress-fade-enter-from,
.progress-fade-leave-to {
  opacity: 0;
}

@media (prefers-reduced-motion: reduce) {
  .navigation-progress-bar {
    animation: progress-pulse 2s ease-in-out infinite;
  }

  @keyframes progress-pulse {
    0%,
    100% {
      opacity: 0.4;
    }
    50% {
      opacity: 1;
    }
  }
}
</style>