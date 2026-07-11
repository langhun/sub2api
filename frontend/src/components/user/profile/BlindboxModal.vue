<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="show && result" class="blindbox-overlay" style="z-index: 60" @click.self="handleClose">
        <div class="blindbox-container" @click.stop>
          <div class="blindbox-glow" :class="rarityGlowClass"></div>

          <div class="blindbox-card">
            <div class="blindbox-card-inner">
              <div class="blindbox-sparkles" v-if="result.rarity === 'legendary'">
                <span v-for="i in 12" :key="i" class="sparkle" :style="sparkleStyle(i)"></span>
              </div>
              <div class="blindbox-shine" :class="rarityShineClass" v-if="result.rarity === 'epic' || result.rarity === 'legendary'"></div>

              <div class="blindbox-icon-wrapper" :class="rarityIconClass">
                <svg class="blindbox-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                  <path stroke-linecap="round" stroke-linejoin="round"
                    d="M21 11.25v8.25a1.5 1.5 0 01-1.5 1.5H5.25a1.5 1.5 0 01-1.5-1.5v-8.25M12 4.875A2.625 2.625 0 109.375 7.5H12m0-2.625V7.5m0-2.625A2.625 2.625 0 1114.625 7.5H12m0 0V21m-8.625-9.75h18c.621 0 1.125-.504 1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125h-18c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125z" />
                </svg>
              </div>

              <div class="blindbox-prize-name" :class="rarityTextClass">{{ result.prize_name }}</div>

              <div class="blindbox-rarity-row">
                <span class="blindbox-rarity-badge" :class="rarityBadgeClass">
                  {{ rarityLabel }}
                </span>
              </div>

              <div class="blindbox-reward-section" :class="rarityRewardBgClass">
                <span class="blindbox-reward-icon">{{ rewardIcon }}</span>
                <span class="blindbox-reward-text" :class="rarityTextClass">{{ rewardText }}</span>
              </div>

              <div v-if="result.reward_type === 'invitation_code' && result.reward_detail" class="blindbox-invite-section">
                <div class="blindbox-invite-label">{{ t('checkin.blindboxInviteCode') }}</div>
                <div class="blindbox-invite-code-row">
                  <code class="blindbox-invite-code">{{ result.reward_detail }}</code>
                  <button type="button" class="blindbox-copy-btn" @click="copyCode(result.reward_detail!)">
                    {{ copied ? t('common.copied') : t('common.copy') }}
                  </button>
                </div>
              </div>

              <button
                type="button"
                class="blindbox-close-btn"
                :class="rarityBtnClass"
                @click="handleClose"
              >
                {{ t('common.confirm') }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { BlindboxResult } from '@/api/checkin'

interface Props {
  show: boolean
  result: BlindboxResult | null
}

const props = defineProps<Props>()
const emit = defineEmits<{ (e: 'close'): void }>()
const { t } = useI18n()

const copied = ref(false)

const rarityLabel = computed(() => {
  if (!props.result) return ''
  const map: Record<string, string> = {
    common: t('checkin.blindboxCommon'),
    rare: t('checkin.blindboxRare'),
    epic: t('checkin.blindboxEpic'),
    legendary: t('checkin.blindboxLegendary'),
  }
  return map[props.result.rarity] || props.result.rarity
})

const rewardIcon = computed(() => {
  if (!props.result) return '🎁'
  switch (props.result.reward_type) {
    case 'balance': return '💰'
    case 'concurrency': return '⚡'
    case 'subscription': return '🎫'
    case 'invitation_code': return '💌'
    default: return '🎁'
  }
})

const rewardText = computed(() => {
  if (!props.result) return ''
  const v = props.result.reward_value
  switch (props.result.reward_type) {
    case 'balance':
      return t('checkin.blindboxBalanceReward', { value: v.toFixed(2) })
    case 'concurrency':
      return t('checkin.blindboxConcurrencyReward', { value: Math.round(v) })
    case 'subscription':
      return t('checkin.blindboxSubscriptionReward', { days: props.result.subscription_days || 0 })
    case 'invitation_code':
      return t('checkin.blindboxInvitationReward')
    default:
      return `${props.result.reward_type}: ${v}`
  }
})

const rarityGlowClass = computed(() => {
  if (!props.result) return ''
  return `glow-${props.result.rarity}`
})

const rarityIconClass = computed(() => {
  if (!props.result) return ''
  return `icon-${props.result.rarity}`
})

const rarityBadgeClass = computed(() => {
  if (!props.result) return ''
  return `badge-${props.result.rarity}`
})

const rarityTextClass = computed(() => {
  if (!props.result) return ''
  return `text-${props.result.rarity}`
})

const rarityBtnClass = computed(() => {
  if (!props.result) return ''
  return `btn-${props.result.rarity}`
})

const rarityRewardBgClass = computed(() => {
  if (!props.result) return ''
  return `reward-bg-${props.result.rarity}`
})

const rarityShineClass = computed(() => {
  if (!props.result) return ''
  return `shine-${props.result.rarity}`
})

function sparkleStyle(i: number) {
  const angle = (i * 30) * Math.PI / 180
  const r = 80 + Math.random() * 40
  const x = Math.cos(angle) * r
  const y = Math.sin(angle) * r
  const delay = i * 0.15
  const size = 3 + Math.random() * 5
  return {
    left: `calc(50% + ${x}px - ${size / 2}px)`,
    top: `calc(50% + ${y}px - ${size / 2}px)`,
    width: `${size}px`,
    height: `${size}px`,
    animationDelay: `${delay}s`,
  }
}

function handleClose() {
  emit('close')
}

async function copyCode(code: string) {
  try {
    await navigator.clipboard.writeText(code)
    copied.value = true
    setTimeout(() => { copied.value = false }, 2000)
  } catch { /* noop */ }
}
</script>

<style scoped>
.blindbox-overlay {
  position: fixed;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.65);
  backdrop-filter: blur(6px);
}

.blindbox-container {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
}

.blindbox-glow {
  position: absolute;
  width: 400px;
  height: 400px;
  border-radius: 50%;
  filter: blur(80px);
  opacity: 0.35;
  animation: pulse-glow 2.5s ease-in-out infinite;
}

.blindbox-card {
  position: relative;
  z-index: 1;
  width: 360px;
  animation: card-enter 0.6s cubic-bezier(0.34, 1.56, 0.64, 1);
}

.blindbox-card-inner {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 14px;
  padding: 36px 28px 28px;
  border-radius: 20px;
  background: white;
  box-shadow: 0 25px 60px -12px rgba(0, 0, 0, 0.3);
  overflow: hidden;
}

html.dark .blindbox-card-inner {
  background: #1a1a2e;
  box-shadow: 0 25px 60px -12px rgba(0, 0, 0, 0.7);
}

.blindbox-shine {
  position: absolute;
  inset: 0;
  pointer-events: none;
  background: linear-gradient(135deg, transparent 30%, rgba(255,255,255,0.08) 50%, transparent 70%);
  animation: shine-sweep 3s ease-in-out infinite;
}

html.dark .blindbox-shine {
  background: linear-gradient(135deg, transparent 30%, rgba(255,255,255,0.04) 50%, transparent 70%);
}

.shine-epic { animation-duration: 4s; }

.blindbox-icon-wrapper {
  width: 64px;
  height: 64px;
  border-radius: 18px;
  display: flex;
  align-items: center;
  justify-content: center;
  animation: icon-float 3s ease-in-out infinite;
}

.blindbox-icon {
  width: 32px;
  height: 32px;
}

.blindbox-prize-name {
  font-size: 22px;
  font-weight: 800;
  text-align: center;
  letter-spacing: -0.01em;
  animation: prize-pop 0.5s cubic-bezier(0.34, 1.56, 0.64, 1) 0.25s both;
}

.blindbox-rarity-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.blindbox-rarity-badge {
  padding: 3px 14px;
  border-radius: 9999px;
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.5px;
}

.blindbox-reward-section {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 10px 20px;
  border-radius: 12px;
  width: 100%;
}

.blindbox-reward-icon {
  font-size: 18px;
}

.blindbox-reward-text {
  font-size: 17px;
  font-weight: 700;
}

.blindbox-invite-section {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: 10px 14px;
  border-radius: 10px;
  background: #f9fafb;
  width: 100%;
}

html.dark .blindbox-invite-section {
  background: rgba(55, 65, 81, 0.5);
}

.blindbox-invite-label {
  font-size: 11px;
  color: #9ca3af;
}

.blindbox-invite-code-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.blindbox-invite-code {
  font-size: 13px;
  font-family: 'SF Mono', 'Fira Code', monospace;
  word-break: break-all;
  text-align: center;
  color: #111827;
  font-weight: 600;
  background: white;
  padding: 4px 10px;
  border-radius: 6px;
  border: 1px solid #e5e7eb;
}

html.dark .blindbox-invite-code {
  color: #f3f4f6;
  background: #1f2937;
  border-color: #374151;
}

.blindbox-copy-btn {
  padding: 4px 12px;
  border-radius: 6px;
  font-size: 11px;
  font-weight: 500;
  color: white;
  background: #6366f1;
  border: none;
  cursor: pointer;
  transition: all 0.2s;
}

.blindbox-copy-btn:hover {
  background: #4f46e5;
}

.blindbox-close-btn {
  margin-top: 4px;
  padding: 10px 48px;
  border-radius: 12px;
  font-size: 14px;
  font-weight: 600;
  color: white;
  border: none;
  cursor: pointer;
  transition: all 0.2s;
}

.blindbox-close-btn:hover {
  filter: brightness(1.1);
  transform: translateY(-1px);
}

.blindbox-close-btn:active {
  transform: translateY(0);
}

/* Glow colors */
.glow-common { background-color: #9ca3af; }
.glow-rare { background-color: #3b82f6; }
.glow-epic { background-color: #8b5cf6; }
.glow-legendary { background-color: #f59e0b; }

/* Icon wrapper styles */
.icon-common { background-color: #f3f4f6; color: #6b7280; }
.icon-rare { background-color: #dbeafe; color: #2563eb; }
.icon-epic { background-color: #ede9fe; color: #7c3aed; }
.icon-legendary { background-color: #fef3c7; color: #d97706; }

html.dark .icon-common { background-color: #374151; color: #9ca3af; }
html.dark .icon-rare { background-color: #1e3a5f; color: #60a5fa; }
html.dark .icon-epic { background-color: #2d1b69; color: #a78bfa; }
html.dark .icon-legendary { background-color: #451a03; color: #fbbf24; }

/* Badge styles */
.badge-common { background-color: #f3f4f6; color: #6b7280; }
.badge-rare { background-color: #dbeafe; color: #2563eb; }
.badge-epic { background-color: #ede9fe; color: #7c3aed; }
.badge-legendary { background-color: #fef3c7; color: #d97706; }

html.dark .badge-common { background-color: #374151; color: #9ca3af; }
html.dark .badge-rare { background-color: #1e3a5f; color: #60a5fa; }
html.dark .badge-epic { background-color: #2d1b69; color: #a78bfa; }
html.dark .badge-legendary { background-color: #451a03; color: #fbbf24; }

/* Text colors */
.text-common { color: #6b7280; }
.text-rare { color: #2563eb; }
.text-epic { color: #7c3aed; }
.text-legendary { color: #d97706; }

html.dark .text-common { color: #9ca3af; }
html.dark .text-rare { color: #60a5fa; }
html.dark .text-epic { color: #a78bfa; }
html.dark .text-legendary { color: #fbbf24; }

/* Button colors */
.btn-common { background-color: #6b7280; }
.btn-rare { background-color: #2563eb; }
.btn-epic { background-color: #7c3aed; }
.btn-legendary { background-color: #d97706; }

/* Reward section backgrounds */
.reward-bg-common { background-color: #f9fafb; }
.reward-bg-rare { background-color: #eff6ff; }
.reward-bg-epic { background-color: #f5f3ff; }
.reward-bg-legendary { background-color: #fffbeb; }

html.dark .reward-bg-common { background-color: rgba(55,65,81,0.4); }
html.dark .reward-bg-rare { background-color: rgba(30,58,95,0.4); }
html.dark .reward-bg-epic { background-color: rgba(45,27,105,0.3); }
html.dark .reward-bg-legendary { background-color: rgba(69,26,3,0.3); }

/* Sparkles */
.blindbox-sparkles {
  position: absolute;
  inset: 0;
  pointer-events: none;
}

.sparkle {
  position: absolute;
  background: #fbbf24;
  border-radius: 50%;
  animation: sparkle-float 2.5s ease-in-out infinite;
  box-shadow: 0 0 8px rgba(251, 191, 36, 0.6);
}

/* Animations */
@keyframes card-enter {
  from {
    opacity: 0;
    transform: scale(0.6) rotateY(180deg);
  }
  to {
    opacity: 1;
    transform: scale(1) rotateY(0deg);
  }
}

@keyframes icon-float {
  0%, 100% {
    transform: translateY(0);
  }
  50% {
    transform: translateY(-4px);
  }
}

@keyframes prize-pop {
  from {
    opacity: 0;
    transform: scale(0.5);
  }
  to {
    opacity: 1;
    transform: scale(1);
  }
}

@keyframes pulse-glow {
  0%, 100% {
    opacity: 0.25;
    transform: scale(1);
  }
  50% {
    opacity: 0.5;
    transform: scale(1.12);
  }
}

@keyframes sparkle-float {
  0%, 100% {
    opacity: 0;
    transform: translateY(0) scale(0);
  }
  20% {
    opacity: 1;
    transform: translateY(-8px) scale(1);
  }
  80% {
    opacity: 0.4;
    transform: translateY(-24px) scale(0.4);
  }
}

@keyframes shine-sweep {
  0%, 100% {
    transform: translateX(-100%);
  }
  50% {
    transform: translateX(100%);
  }
}

/* Modal transition */
.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.3s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}
</style>
