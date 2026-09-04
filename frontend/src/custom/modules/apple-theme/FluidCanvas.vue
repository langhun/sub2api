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
  color1: '#4A9E8E',
  color2: '#7EC8BB',
  color3: '#E0F2EF',
}

const DARK_PARAMS: FluidParams = {
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
  color1: '#1E3A5F',
  color2: '#2A4A7A',
  color3: '#0C121B',
}

const canvasRef = ref<HTMLCanvasElement | null>(null)
const { effectiveMode } = useTheme()

onMounted(() => {
  const canvas = canvasRef.value
  if (!canvas) return

  // 初始化流体着色器
  const initial = effectiveMode.value === 'dark' ? DARK_PARAMS : LIGHT_PARAMS
  const handle = attachFluidShader(canvas, initial)

  // 监听主题变化
  const unwatch = watch(effectiveMode, (mode) => {
    handle.setParams(mode === 'dark' ? DARK_PARAMS : LIGHT_PARAMS)
  })

  onBeforeUnmount(() => {
    unwatch()
    handle.dispose()
  })
})
</script>
