<template>
  <canvas
    ref="canvasRef"
    data-glass-fluid-canvas
    aria-hidden="true"
    class="absolute inset-0 h-full w-full"
  />
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, watch } from 'vue'
import { attachFluidShader, type FluidParams } from './fluidShader'
import { useTheme } from './useTheme'

const LIGHT_PARAMS: FluidParams = {
  mouseRadius: 0.22,
  mouseStrength: 1.1,
  decay: 0.96,
  distortBoost: 1.35,
  noiseBoost: 0,
  swirlBoost: 0.45,
  speed: 18,
  distortion: 22,
  swirl: 14,
  swirlIterations: 8,
  scale: 0.5,
  rotation: -5,
  proportion: 50,
  softness: 100,
  shapeScale: 10,
  offsetX: 0,
  offsetY: 65,
  color1: '#4A9E8E',  // 青绿色
  color2: '#7EC8BB',  // 浅青绿
  color3: '#D0EBE6',  // 极淡青绿
}

const DARK_PARAMS: FluidParams = {
  mouseRadius: 0.22,
  mouseStrength: 1.1,
  decay: 0.96,
  distortBoost: 1.35,
  noiseBoost: 0,
  swirlBoost: 0.45,
  speed: 16,
  distortion: 22,
  swirl: 14,
  swirlIterations: 8,
  scale: 0.5,
  rotation: -5,
  proportion: 50,
  softness: 100,
  shapeScale: 10,
  offsetX: 0,
  offsetY: 65,
  color1: '#2d4a6f',  // 深蓝
  color2: '#3d5a85',  // 中蓝
  color3: '#1a2838',  // 深蓝黑
}

const canvasRef = ref<HTMLCanvasElement | null>(null)
const { effectiveMode } = useTheme()
let cleanup: (() => void) | null = null

onMounted(() => {
  const canvas = canvasRef.value
  if (!canvas) {
    console.error('[FluidCanvas] Canvas element not found')
    return
  }

  console.log('[FluidCanvas] Initializing fluid shader...', effectiveMode.value)

  const initial = effectiveMode.value === 'dark' ? DARK_PARAMS : LIGHT_PARAMS
  const handle = attachFluidShader(canvas, initial)

  const unwatch = watch(effectiveMode, (mode) => {
    console.log('[FluidCanvas] Theme changed to:', mode)
    handle.setParams(mode === 'dark' ? DARK_PARAMS : LIGHT_PARAMS)
  })

  cleanup = () => {
    console.log('[FluidCanvas] Cleaning up')
    unwatch()
    handle.dispose()
  }
})

onBeforeUnmount(() => {
  if (cleanup) {
    cleanup()
  }
})
</script>
