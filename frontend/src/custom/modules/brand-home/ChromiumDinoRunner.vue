<template>
  <section class="chromium-dino-runner" aria-label="Chromium Dino">
    <span class="icon-offline" aria-hidden="true"></span>
    <div id="offline-resources" class="hidden">
      <img id="offline-resources-1x" src="/custom/brand-home/chromium-offline-sprite.png" alt="" />
      <img id="offline-resources-2x" src="/custom/brand-home/chromium-offline-sprite-2x.png" alt="" />
    </div>
    <template id="audio-resources">
      <audio id="offline-sound-press" src="/custom/brand-home/button-press.mp3"></audio>
      <audio id="offline-sound-hit" src="/custom/brand-home/hit.mp3"></audio>
      <audio id="offline-sound-reached" src="/custom/brand-home/score-reached.mp3"></audio>
    </template>
  </section>
</template>

<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted } from 'vue'
import { Runner } from '@/custom/modules/brand-home/chromiumDino/offline'

onMounted(async () => {
  await nextTick()
  Runner.initializeInstance('.chromium-dino-runner')
})

onBeforeUnmount(() => {
  Runner.disposeInstance()
})
</script>

<style>
.chromium-dino-runner {
  box-sizing: border-box;
  width: 100%;
  max-width: 600px;
  min-height: 150px;
  padding: 0;
  position: relative;
}

.chromium-dino-runner .hidden,
.chromium-dino-runner .icon-offline,
.chromium-dino-runner .slow-speed-option {
  display: none;
}

.chromium-dino-runner .runner-container {
  direction: ltr;
  height: 150px;
  max-width: 600px;
  overflow: hidden;
  position: absolute;
  top: 35px;
  width: 44px;
}

.chromium-dino-runner .runner-canvas {
  display: block;
  max-width: 600px;
  opacity: 1;
  overflow: hidden;
  position: absolute;
  top: 0;
}

.arcade-mode .chromium-dino-runner,
.arcade-mode .chromium-dino-runner .runner-container,
.arcade-mode .chromium-dino-runner .runner-canvas {
  image-rendering: pixelated;
  max-width: 100%;
  overflow: hidden;
}

.arcade-mode .chromium-dino-runner {
  height: 100vh;
  max-width: 100%;
  overflow: hidden;
}

.arcade-mode .chromium-dino-runner .runner-container {
  left: 0;
  margin: auto;
  right: 0;
  transform-origin: top center;
  transition: transform 250ms cubic-bezier(0.4, 0, 1, 1) 400ms;
  z-index: 2;
}

html.inverted .dino-page {
  background: #202124;
}

html.inverted .dino-page .brand,
html.inverted .dino-page .account-link {
  color: #f1f3f4;
}

html.inverted .dino-page .runner-canvas {
  filter: invert(1);
}

@media (max-width: 700px) {
  .chromium-dino-runner .runner-canvas {
    max-width: none;
  }
}
</style>
