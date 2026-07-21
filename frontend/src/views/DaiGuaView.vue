<template>
  <main class="dino-page" :class="{ 'is-night': nightMode }">
    <header class="dino-header">
      <router-link class="brand" to="/home" aria-label="返回默认首页">DaiGua</router-link>
      <nav class="header-actions" aria-label="页面导航">
        <router-link to="/home">默认首页</router-link>
        <router-link :to="isAuthenticated ? dashboardPath : '/login'">
          {{ isAuthenticated ? '控制台' : '登录' }}
        </router-link>
      </nav>
    </header>

    <section class="game-stage" aria-labelledby="dino-title">
      <div class="game-copy">
        <p class="eyebrow">DAIGUA ARCADE</p>
        <h1 id="dino-title">离线也向前跑</h1>
        <p>原版 Chromium 像素恐龙小游戏</p>
      </div>

      <button
        ref="gameButton"
        class="game-frame"
        type="button"
        :aria-label="gameAriaLabel"
        @click="handlePrimaryAction"
        @keydown.space.prevent="handlePrimaryAction"
        @keydown.enter.prevent="handlePrimaryAction"
      >
        <canvas ref="canvas" class="runner-canvas" aria-hidden="true"></canvas>
        <span class="sr-only">{{ gameAriaLabel }}</span>
      </button>

      <div class="game-meta" aria-live="polite">
        <span>{{ gameStatus }}</span>
        <span>最高分 {{ paddedHighScore }}</span>
      </div>
    </section>

    <footer class="dino-footer">
      <span>空格键、上箭头或点击跳跃</span>
      <span>Chromium Dino assets, BSD-style license</span>
    </footer>
  </main>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { useAuthStore } from '@/stores'

type GameState = 'ready' | 'running' | 'crashed'

interface Obstacle {
  height: number
  kind: 'small' | 'large' | 'pterodactyl'
  width: number
  x: number
  y: number
  frame: number
}

interface Cloud {
  x: number
  y: number
}

const WIDTH = 600
const HEIGHT = 150
const GROUND_Y = 140
const TREX_X = 50
const TREX_WIDTH = 44
const TREX_HEIGHT = 47
const TREX_GROUND_Y = GROUND_Y - TREX_HEIGHT
const HIGH_SCORE_KEY = 'daigua-dino-high-score'
const canvas = ref<HTMLCanvasElement | null>(null)
const gameButton = ref<HTMLButtonElement | null>(null)
const state = ref<GameState>('ready')
const score = ref(0)
const highScore = ref(0)
const nightMode = ref(false)
const authStore = useAuthStore()

let context: CanvasRenderingContext2D | null = null
let sprite: HTMLImageElement | null = null
let animationId = 0
let lastFrame = 0
let distance = 0
let speed = 6
let nextObstacleAt = 270
let nextCloudAt = 140
let tRexY = TREX_GROUND_Y
let jumpVelocity = 0
let runFrame = 0
let obstacleFrame = 0
let clouds: Cloud[] = []
let obstacles: Obstacle[] = []

const isAuthenticated = computed(() => authStore.isAuthenticated)
const dashboardPath = computed(() => authStore.isAdmin ? '/admin/dashboard' : '/dashboard')
const paddedHighScore = computed(() => String(highScore.value).padStart(5, '0'))
const gameStatus = computed(() => {
  if (state.value === 'crashed') return '游戏结束，点击重新开始'
  if (state.value === 'running') return `得分 ${String(score.value).padStart(5, '0')}`
  return '点击、空格或上箭头开始'
})
const gameAriaLabel = computed(() => state.value === 'crashed' ? '重新开始 DaiGua 恐龙游戏' : '开始或跳跃')

function configureCanvas(): void {
  const element = canvas.value
  if (!element) return
  const ratio = Math.min(window.devicePixelRatio || 1, 2)
  element.width = WIDTH * ratio
  element.height = HEIGHT * ratio
  context = element.getContext('2d')
  if (!context) return
  context.scale(ratio, ratio)
  context.imageSmoothingEnabled = false
}

function loadHighScore(): void {
  const stored = Number(window.localStorage.getItem(HIGH_SCORE_KEY))
  highScore.value = Number.isFinite(stored) ? Math.max(0, Math.floor(stored)) : 0
}

function resetGame(): void {
  state.value = 'ready'
  score.value = 0
  distance = 0
  speed = 6
  nextObstacleAt = 270
  nextCloudAt = 80
  tRexY = TREX_GROUND_Y
  jumpVelocity = 0
  runFrame = 0
  obstacleFrame = 0
  nightMode.value = false
  obstacles = []
  clouds = [{ x: 430, y: 35 }]
  draw()
}

function startGame(): void {
  if (state.value === 'crashed') resetGame()
  if (state.value === 'ready') state.value = 'running'
  jump()
}

function jump(): void {
  if (state.value !== 'running' || tRexY < TREX_GROUND_Y - 1) return
  jumpVelocity = -10
}

function handlePrimaryAction(): void {
  startGame()
  gameButton.value?.focus()
}

function handleKeydown(event: KeyboardEvent): void {
  if (event.defaultPrevented || event.ctrlKey || event.metaKey || event.altKey) return
  if (event.code !== 'Space' && event.code !== 'ArrowUp') return
  event.preventDefault()
  startGame()
}

function createObstacle(): Obstacle {
  const pterodactyl = score.value > 450 && Math.random() < 0.22
  if (pterodactyl) {
    return { kind: 'pterodactyl', x: WIDTH + 20, y: 80, width: 46, height: 40, frame: 0 }
  }
  const large = Math.random() > 0.48
  return large
    ? { kind: 'large', x: WIDTH + 20, y: 90, width: 25, height: 50, frame: 0 }
    : { kind: 'small', x: WIDTH + 20, y: 105, width: 17, height: 35, frame: 0 }
}

function collides(obstacle: Obstacle): boolean {
  const player = { x: TREX_X + 6, y: tRexY + 5, width: 31, height: 38 }
  const target = obstacle.kind === 'pterodactyl'
    ? { x: obstacle.x + 5, y: obstacle.y + 8, width: 36, height: 24 }
    : { x: obstacle.x + 2, y: obstacle.y + 2, width: obstacle.width - 4, height: obstacle.height - 3 }
  return player.x < target.x + target.width && player.x + player.width > target.x
    && player.y < target.y + target.height && player.y + player.height > target.y
}

function update(delta: number): void {
  const frameRatio = delta / (1000 / 60)
  distance += speed * frameRatio
  speed = Math.min(13, speed + delta * 0.00085)
  score.value = Math.floor(distance / 5)
  nightMode.value = Math.floor(score.value / 700) % 2 === 1

  jumpVelocity += 0.6 * frameRatio
  tRexY = Math.min(TREX_GROUND_Y, tRexY + jumpVelocity * frameRatio)
  if (tRexY === TREX_GROUND_Y) jumpVelocity = 0
  runFrame += delta
  obstacleFrame += delta

  if (distance >= nextCloudAt) {
    clouds.push({ x: WIDTH + 10, y: 26 + Math.floor(Math.random() * 42) })
    nextCloudAt = distance + 100 + Math.random() * 260
  }
  clouds.forEach(cloud => { cloud.x -= speed * 0.22 * frameRatio })
  clouds = clouds.filter(cloud => cloud.x > -50)

  if (distance >= nextObstacleAt) {
    const obstacle = createObstacle()
    obstacles.push(obstacle)
    nextObstacleAt = distance + 150 + Math.random() * 150 + speed * 12
  }
  obstacles.forEach(obstacle => {
    obstacle.x -= speed * frameRatio
    obstacle.frame = Math.floor(obstacleFrame / 100) % 2
  })
  obstacles = obstacles.filter(obstacle => obstacle.x + obstacle.width > -2)

  if (obstacles.some(collides)) {
    state.value = 'crashed'
    highScore.value = Math.max(highScore.value, score.value)
    window.localStorage.setItem(HIGH_SCORE_KEY, String(highScore.value))
  }
}

function drawSprite(sourceX: number, sourceY: number, sourceWidth: number, sourceHeight: number, x: number, y: number, width = sourceWidth, height = sourceHeight): void {
  if (!context || !sprite) return
  context.drawImage(sprite, sourceX, sourceY, sourceWidth, sourceHeight, x, y, width, height)
}

function drawScore(): void {
  if (!context) return
  context.fillStyle = nightMode.value ? '#f5f5f5' : '#535353'
  context.font = '10px monospace'
  context.textAlign = 'right'
  context.fillText(`${String(highScore.value).padStart(5, '0')}  ${String(score.value).padStart(5, '0')}`, WIDTH - 8, 16)
}

function draw(): void {
  if (!context) return
  context.clearRect(0, 0, WIDTH, HEIGHT)
  context.fillStyle = nightMode.value ? '#202124' : '#ffffff'
  context.fillRect(0, 0, WIDTH, HEIGHT)
  context.save()
  if (nightMode.value) context.filter = 'invert(1)'
  clouds.forEach(cloud => drawSprite(86, 2, 46, 14, Math.round(cloud.x), cloud.y))
  drawSprite(2, 54, 600, 12, 0, 127)
  obstacles.forEach(obstacle => {
    if (obstacle.kind === 'small') drawSprite(228, 2, 17, 35, Math.round(obstacle.x), obstacle.y)
    if (obstacle.kind === 'large') drawSprite(332, 2, 25, 50, Math.round(obstacle.x), obstacle.y)
    if (obstacle.kind === 'pterodactyl') drawSprite(134 + obstacle.frame * 46, 2, 46, 40, Math.round(obstacle.x), obstacle.y)
  })
  const frame = state.value === 'crashed' ? 220 : tRexY < TREX_GROUND_Y ? 0 : 88 + (Math.floor(runFrame / 84) % 2) * 44
  drawSprite(848 + frame, 2, TREX_WIDTH, TREX_HEIGHT, TREX_X, Math.round(tRexY))
  context.restore()
  drawScore()
  if (state.value === 'crashed') drawGameOver()
}

function drawGameOver(): void {
  if (!context) return
  context.fillStyle = nightMode.value ? '#f5f5f5' : '#535353'
  context.font = '14px Arial, sans-serif'
  context.textAlign = 'center'
  context.fillText('GAME OVER', WIDTH / 2, 70)
  drawSprite(2, 68, 36, 32, WIDTH / 2 - 18, 83)
}

function tick(timestamp: number): void {
  const delta = Math.min(34, timestamp - lastFrame || 16)
  lastFrame = timestamp
  if (state.value === 'running') update(delta)
  draw()
  animationId = window.requestAnimationFrame(tick)
}

onMounted(async () => {
  await nextTick()
  configureCanvas()
  loadHighScore()
  sprite = new Image()
  sprite.src = '/dai-gua/chromium-offline-sprite.png'
  sprite.addEventListener('load', resetGame, { once: true })
  window.addEventListener('keydown', handleKeydown)
  animationId = window.requestAnimationFrame(tick)
})

onBeforeUnmount(() => {
  window.cancelAnimationFrame(animationId)
  window.removeEventListener('keydown', handleKeydown)
})
</script>

<style scoped>
.dino-page { min-height: 100vh; display: flex; flex-direction: column; background: #fff; color: #202124; font-family: Arial, Helvetica, sans-serif; transition: background-color .25s, color .25s; }
.dino-page.is-night { background: #202124; color: #f5f5f5; }
.dino-header { display: flex; min-height: 72px; align-items: center; justify-content: space-between; padding: 0 40px; font-size: 14px; }
.brand { color: inherit; font-size: 18px; font-weight: 600; text-decoration: none; }
.header-actions { display: flex; gap: 20px; }
.header-actions a { color: inherit; opacity: .72; text-decoration: none; }
.header-actions a:hover { opacity: 1; text-decoration: underline; }
.game-stage { display: flex; flex: 1; flex-direction: column; justify-content: center; width: min(100% - 40px, 760px); margin: 0 auto; padding: 56px 0 40px; }
.game-copy { margin-bottom: 26px; }
.eyebrow { margin: 0 0 10px; font: 11px/1.2 monospace; letter-spacing: .12em; opacity: .6; }
h1 { margin: 0; font-size: clamp(30px, 5vw, 44px); font-weight: 500; letter-spacing: 0; }
.game-copy > p:last-child { margin: 12px 0 0; font-size: 14px; opacity: .65; }
.game-frame { display: block; width: 100%; padding: 0; overflow: hidden; border: 0; border-radius: 0; background: transparent; cursor: pointer; outline-offset: 6px; }
.runner-canvas { display: block; width: 100%; height: auto; aspect-ratio: 4 / 1; }
.game-meta { display: flex; justify-content: space-between; margin-top: 18px; font: 12px/1.3 monospace; opacity: .68; }
.dino-footer { display: flex; justify-content: space-between; gap: 20px; padding: 24px 40px 28px; font-size: 11px; opacity: .55; }
.sr-only { position: absolute; width: 1px; height: 1px; padding: 0; overflow: hidden; clip: rect(0, 0, 0, 0); white-space: nowrap; border: 0; }
@media (max-width: 640px) { .dino-header { min-height: 60px; padding: 0 20px; } .header-actions { gap: 14px; font-size: 12px; } .game-stage { width: min(100% - 28px, 760px); justify-content: flex-start; padding-top: 18vh; } .dino-footer { display: block; padding: 20px; line-height: 1.8; } .dino-footer span { display: block; } }
</style>
