<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { attachFluidShader, type FluidParams } from '@/lib/fluid-shader'

const canvas = ref<HTMLCanvasElement | null>(null)
let dispose: (() => void) | undefined

const lightParams: FluidParams = {
  mouseRadius: 0.22, mouseStrength: 1.1, decay: 0.96, distortBoost: 1.35,
  noiseBoost: 0, swirlBoost: 0.45, speed: 18, distortion: 22, swirl: 14,
  swirlIterations: 8, scale: 0.5, rotation: -5, proportion: 50, softness: 100,
  shapeScale: 10, offsetX: 0, offsetY: 65, color1: '#4A9E8E', color2: '#7EC8BB', color3: '#E0F2EF'
}

const darkParams: FluidParams = {
  ...lightParams, color1: '#1E3A5F', color2: '#2A4A7A', color3: '#0C121B'
}

onMounted(() => {
  if (!canvas.value) return
  const handle = attachFluidShader(canvas.value, document.documentElement.classList.contains('dark') ? darkParams : lightParams)
  const themeObserver = new MutationObserver(() => {
    handle.setParams(document.documentElement.classList.contains('dark') ? darkParams : lightParams)
  })
  themeObserver.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
  const media = window.matchMedia('(prefers-color-scheme: dark)')
  const onMediaChange = () => handle.setParams(document.documentElement.classList.contains('dark') ? darkParams : lightParams)
  if (typeof media.addEventListener === 'function') media.addEventListener('change', onMediaChange)
  else media.addListener?.(onMediaChange)
  dispose = () => {
    themeObserver.disconnect()
    if (typeof media.removeEventListener === 'function') media.removeEventListener('change', onMediaChange)
    else media.removeListener?.(onMediaChange)
    handle.dispose()
  }
})

onBeforeUnmount(() => dispose?.())
</script>

<template>
  <canvas ref="canvas" class="fluid-canvas" data-glass-fluid-canvas aria-hidden="true" />
</template>

<style scoped>
.fluid-canvas {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  display: block;
  pointer-events: none;
}
</style>
