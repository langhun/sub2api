<template>
  <main class="dino-page" :class="{ 'is-night': nightMode }">
    <header class="dino-header">
      <router-link class="brand" to="/home">DaiGua</router-link>
      <router-link class="account-link" :to="isAuthenticated ? dashboardPath : '/login'">
        {{ isAuthenticated ? '控制台' : '登录' }}
      </router-link>
    </header>
    <section class="interstitial-wrapper" aria-label="Chromium Dino Game">
      <button
        ref="gameButton"
        class="runner-container"
        type="button"
        :aria-label="gameAriaLabel"
        @click="handlePrimaryAction"
        @keydown.space.prevent="handlePrimaryAction"
        @keydown.enter.prevent="handlePrimaryAction"
      >
        <canvas ref="canvas" class="runner-canvas" aria-hidden="true"></canvas>
        <span class="sr-only">{{ gameAriaLabel }}</span>
      </button>
    </section>
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
let groundOffset = 0
let clouds: Cloud[] = []
let obstacles: Obstacle[] = []

const gameAriaLabel = computed(() => state.value === 'crashed' ? '重新开始 DaiGua 恐龙游戏' : '开始或跳跃')
const isAuthenticated = computed(() => authStore.isAuthenticated)
const dashboardPath = computed(() => authStore.isAdmin ? '/admin/dashboard' : '/dashboard')

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
  groundOffset = 0
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
  groundOffset = (groundOffset + speed * frameRatio) % WIDTH
  score.value = Math.round(distance * 0.025)
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

function drawMeter(value: string, startX: number): void {
  for (let index = 0; index < value.length; index++) {
    const character = value[index]
    if (!character || character === ' ') continue
    const spriteOffset = character === 'H' ? 10 : character === 'I' ? 11 : Number(character)
    if (!Number.isInteger(spriteOffset)) continue
    drawSprite(655 + spriteOffset * 10, 2, 10, 13, startX + index * 11, 5)
  }
}

function drawScore(): void {
  const distanceValue = String(score.value).padStart(5, '0')
  if (highScore.value > 0) {
    drawMeter(`HI ${String(highScore.value).padStart(5, '0')}`, 434)
  }
  drawMeter(distanceValue, 534)
}

function draw(): void {
  if (!context) return
  context.clearRect(0, 0, WIDTH, HEIGHT)
  context.fillStyle = nightMode.value ? '#202124' : '#ffffff'
  context.fillRect(0, 0, WIDTH, HEIGHT)
  context.save()
  if (nightMode.value) context.filter = 'invert(1)'
  clouds.forEach(cloud => drawSprite(86, 2, 46, 14, Math.round(cloud.x), cloud.y))
  const horizonX = -Math.round(groundOffset)
  drawSprite(2, 54, 600, 12, horizonX, 127)
  drawSprite(2, 54, 600, 12, horizonX + WIDTH, 127)
  obstacles.forEach(obstacle => {
    if (obstacle.kind === 'small') drawSprite(228, 2, 17, 35, Math.round(obstacle.x), obstacle.y)
    if (obstacle.kind === 'large') drawSprite(332, 2, 25, 50, Math.round(obstacle.x), obstacle.y)
    if (obstacle.kind === 'pterodactyl') drawSprite(134 + obstacle.frame * 46, 2, 46, 40, Math.round(obstacle.x), obstacle.y)
  })
  const waitFrame = runFrame % 3000 < 180 ? 44 : 0
  const frame = state.value === 'crashed'
    ? 220
    : tRexY < TREX_GROUND_Y
      ? 0
      : state.value === 'ready'
        ? waitFrame
        : 88 + (Math.floor(runFrame / 84) % 2) * 44
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
  if (state.value === 'ready') runFrame += delta
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
.dino-page { position: relative; min-height: 100vh; overflow: hidden; background: #fff; transition: background-color .25s; }
.dino-page.is-night { background: #202124; }
.dino-header { display: flex; align-items: center; justify-content: space-between; height: 64px; padding: 0 32px; font: 14px Arial, sans-serif; }
.brand { color: #202124; font-size: 16px; font-weight: 600; text-decoration: none; }
.account-link { color: #5f6368; text-decoration: none; }
.account-link:hover, .account-link:focus-visible { color: #202124; text-decoration: underline; }
.is-night .brand { color: #f1f3f4; }.is-night .account-link { color: #bdc1c6; }.is-night .account-link:hover, .is-night .account-link:focus-visible { color: #f1f3f4; }
.interstitial-wrapper { box-sizing: border-box; width: 100%; max-width: 600px; margin: calc(20vh - 64px) auto 0; }
.runner-container { display: block; width: 100%; padding: 0; overflow: hidden; border: 0; outline: 0; background: transparent; cursor: pointer; }
.runner-container:focus-visible { outline: 0; }
.runner-canvas { display: block; width: 100%; height: auto; aspect-ratio: 4 / 1; }
.sr-only { position: absolute; width: 1px; height: 1px; padding: 0; overflow: hidden; clip: rect(0, 0, 0, 0); white-space: nowrap; border: 0; }
@media (max-width: 700px) { .dino-header { height: 56px; padding: 0 20px; font-size: 13px; } .interstitial-wrapper { width: calc(100% - 40px); margin-top: calc(24vh - 56px); } }
</style>
