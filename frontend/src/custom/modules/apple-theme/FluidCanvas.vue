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

// 基础参数
const baseParams = {
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
}

// Arctic Skyline - 北极天际
const ARCTIC_LIGHT: FluidParams = {
  ...baseParams,
  color1: '#0ea5e9',  // 天蓝
  color2: '#38bdf8',  // 亮蓝
  color3: '#bae6fd',  // 浅蓝
}

const ARCTIC_DARK: FluidParams = {
  ...baseParams,
  color1: '#1e3a8a',  // 深蓝
  color2: '#2563eb',  // 中蓝
  color3: '#1e293b',  // 石板灰
}

// Frosted Lilac - 霜冻薰衣草
const LILAC_LIGHT: FluidParams = {
  ...baseParams,
  color1: '#a78bfa',  // 紫色
  color2: '#c4b5fd',  // 淡紫
  color3: '#ede9fe',  // 极淡紫
}

const LILAC_DARK: FluidParams = {
  ...baseParams,
  color1: '#5b2a86',  // 深紫
  color2: '#7c3aed',  // 中紫
  color3: '#4c1d95',  // 深紫黑
}

// Ocean Glass - 海洋玻璃
const OCEAN_LIGHT: FluidParams = {
  ...baseParams,
  color1: '#14b8a6',  // 青色
  color2: '#5eead4',  // 亮青
  color3: '#a7f3d0',  // 淡青
}

const OCEAN_DARK: FluidParams = {
  ...baseParams,
  color1: '#0b1320',  // 深海蓝黑
  color2: '#0ea5a5',  // 海蓝
  color3: '#1e293b',  // 深灰
}

// Warm Gradient - 温暖渐变
const WARM_LIGHT: FluidParams = {
  ...baseParams,
  color1: '#f97316',  // 橙色
  color2: '#fb923c',  // 亮橙
  color3: '#fed7aa',  // 淡橙
}

const WARM_DARK: FluidParams = {
  ...baseParams,
  color1: '#7c2d12',  // 深橙棕
  color2: '#c2410c',  // 深橙
  color3: '#431407',  // 深棕
}

// Space - 太空科幻（深邃、低对比度、不刺眼）
const SPACE_LIGHT: FluidParams = {
  ...baseParams,
  speed: 12,  // 慢一点，更飘渺
  color1: '#2A2F38',  // 深灰
  color2: '#6D7786',  // 中灰
  color3: '#C3CAD4',  // 浅灰
}

const SPACE_DARK: FluidParams = {
  ...baseParams,
  speed: 10,  // 更慢，像在太空漂浮
  color1: '#18204A',  // 深邃蓝
  color2: '#2B3A73',  // 宇宙蓝
  color3: '#1B2A55',  // 暗蓝
}

// Apple (原始) - 蓝灰色
const APPLE_LIGHT: FluidParams = {
  ...baseParams,
  color1: '#93C5FD',  // 蓝色 300
  color2: '#BFDBFE',  // 蓝色 200
  color3: '#DBEAFE',  // 蓝色 100
}

const APPLE_DARK: FluidParams = {
  ...baseParams,
  color1: '#1e3a8a',  // 深蓝色 900
  color2: '#1e40af',  // 蓝色 800
  color3: '#1e293b',  // 石板灰 800
}

const canvasRef = ref<HTMLCanvasElement | null>(null)
const { effectiveMode, themeStyle } = useTheme()
let cleanup: (() => void) | null = null

// 根据主题和模式获取对应的流体参数
function getFluidParams(): FluidParams {
  const isDark = effectiveMode.value === 'dark'

  switch (themeStyle.value) {
    case 'arctic':
      return isDark ? ARCTIC_DARK : ARCTIC_LIGHT
    case 'lilac':
      return isDark ? LILAC_DARK : LILAC_LIGHT
    case 'ocean':
      return isDark ? OCEAN_DARK : OCEAN_LIGHT
    case 'warm':
      return isDark ? WARM_DARK : WARM_LIGHT
    case 'space':
      return isDark ? SPACE_DARK : SPACE_LIGHT
    case 'apple':
    default:
      return isDark ? APPLE_DARK : APPLE_LIGHT
  }
}

onMounted(() => {
  const canvas = canvasRef.value
  if (!canvas) {
    console.error('[FluidCanvas] Canvas element not found')
    return
  }

  console.log('[FluidCanvas] Initializing fluid shader...', themeStyle.value, effectiveMode.value)

  // 初始化流体着色器
  const initial = getFluidParams()
  const handle = attachFluidShader(canvas, initial)

  // 监听主题和模式变化
  const unwatchMode = watch(effectiveMode, () => {
    console.log('[FluidCanvas] Theme mode changed to:', effectiveMode.value)
    handle.setParams(getFluidParams())
  })

  const unwatchStyle = watch(themeStyle, () => {
    console.log('[FluidCanvas] Theme style changed to:', themeStyle.value)
    handle.setParams(getFluidParams())
  })

  cleanup = () => {
    console.log('[FluidCanvas] Cleaning up')
    unwatchMode()
    unwatchStyle()
    handle.dispose()
  }
})

onBeforeUnmount(() => {
  if (cleanup) {
    cleanup()
  }
})
</script>
