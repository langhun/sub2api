<template>
  <div data-testid="checkin-captcha-gate" class="space-y-2">
    <CaptchaChallenge
      v-if="active && settingsReady && captchaEnabled && captchaConfigured"
      ref="challengeRef"
      action="checkin"
      :turnstile-enabled="turnstileEnabled"
      :turnstile-site-key="turnstileSiteKey"
      :tencent-enabled="tencentEnabled"
      :tencent-app-id="tencentAppId"
      :tencent-region="tencentRegion"
      :aliyun-enabled="aliyunEnabled"
      :aliyun-scene-id="aliyunSceneId"
      :aliyun-prefix="aliyunPrefix"
      :aliyun-region="aliyunRegion"
      @verify="handleVerified"
      @expire="handleExpired"
      @error="handleError"
    />
    <p v-if="active && settingsReady && captchaEnabled && !captchaConfigured" class="text-sm text-red-600 dark:text-red-400" role="alert">
      {{ t('checkin.verificationUnavailable') }}
    </p>
    <div v-else-if="active && !settingsReady" class="space-y-2" role="status">
      <p class="text-sm text-gray-600 dark:text-dark-300">
        {{ settingsLoading ? t('checkin.verificationLoading') : t('checkin.verificationSettingsFailed') }}
      </p>
      <button
        v-if="!settingsLoading"
        type="button"
        class="btn btn-secondary btn-sm"
        @click="loadSettings(true)"
      >
        {{ t('common.retry') }}
      </button>
    </div>
    <p v-else-if="errorMessage" class="text-sm text-red-600 dark:text-red-400" role="alert">
      {{ errorMessage }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import CaptchaChallenge from '@/components/CaptchaChallenge.vue'
import { useAppStore } from '@/stores/app'
import type { CheckinCaptchaProof } from '../api/checkin'

const props = defineProps<{ active: boolean }>()
const emit = defineEmits<{ readyChange: [ready: boolean] }>()

const { t } = useI18n()
const appStore = useAppStore()
const challengeRef = ref<InstanceType<typeof CaptchaChallenge> | null>(null)
const token = ref('')
const randstr = ref('')
const errorMessage = ref('')
const settingsResolved = ref(false)
let settingsRequestSequence = 0

const settings = computed(() => appStore.cachedPublicSettings)
const turnstileEnabled = computed(() => settings.value?.turnstile_enabled === true)
const turnstileSiteKey = computed(() => String(settings.value?.turnstile_site_key || ''))
const tencentEnabled = computed(() => settings.value?.tencent_captcha_enabled === true)
const tencentAppId = computed(() => String(settings.value?.tencent_captcha_app_id || ''))
const tencentRegion = computed(() => String(settings.value?.tencent_captcha_region || 'cn'))
const aliyunEnabled = computed(() => settings.value?.aliyun_captcha_enabled === true)
const aliyunSceneId = computed(() => String(settings.value?.aliyun_captcha_scene_id || ''))
const aliyunPrefix = computed(() => String(settings.value?.aliyun_captcha_prefix || ''))
const aliyunRegion = computed(() => String(settings.value?.aliyun_captcha_region || 'cn'))

const captchaEnabled = computed(() => turnstileEnabled.value || tencentEnabled.value || aliyunEnabled.value)
const settingsReady = computed(() => settingsResolved.value && settings.value !== null)
const settingsLoading = ref(false)
const captchaConfigured = computed(() => {
  if (turnstileEnabled.value) return Boolean(turnstileSiteKey.value)
  if (tencentEnabled.value) return Boolean(tencentAppId.value)
  if (aliyunEnabled.value) return Boolean(aliyunSceneId.value && aliyunPrefix.value)
  return true
})
const ready = computed(() => {
  if (!props.active) return false
  if (!settingsReady.value) return false
  if (!captchaEnabled.value) return true
  if (!captchaConfigured.value) return false
  if (turnstileEnabled.value) return Boolean(token.value)
  return true
})

watch(ready, (value) => emit('readyChange', value), { immediate: true })
watch(() => props.active, (active) => {
  if (active) {
    reset()
    void loadSettings(true)
    return
  }
  settingsRequestSequence++
  settingsResolved.value = false
  settingsLoading.value = false
  token.value = ''
  randstr.value = ''
  errorMessage.value = ''
}, { immediate: true })

async function loadSettings(force: boolean): Promise<void> {
  if (!props.active || (settingsReady.value && !force)) return
  const requestSequence = ++settingsRequestSequence
  settingsResolved.value = false
  settingsLoading.value = true
  try {
    const loaded = await appStore.fetchPublicSettings(force)
    if (requestSequence !== settingsRequestSequence || !props.active) return
    settingsResolved.value = loaded !== null && appStore.cachedPublicSettings !== null
  } finally {
    if (requestSequence === settingsRequestSequence) {
      settingsLoading.value = false
    }
  }
}

function handleVerified(value: string, verifiedRandstr: string): void {
  token.value = value
  randstr.value = verifiedRandstr
  errorMessage.value = ''
}

function handleExpired(): void {
  token.value = ''
  randstr.value = ''
  errorMessage.value = t('checkin.verificationExpired')
}

function handleError(): void {
  token.value = ''
  randstr.value = ''
  errorMessage.value = t('checkin.verificationFailed')
}

async function resolveProof(): Promise<CheckinCaptchaProof | null> {
  errorMessage.value = ''
  if (!captchaEnabled.value) return {}
  if (!captchaConfigured.value) {
    errorMessage.value = t('checkin.verificationUnavailable')
    return null
  }
  if (turnstileEnabled.value) {
    if (!token.value) {
      errorMessage.value = t('checkin.completeVerification')
      return null
    }
    return { turnstile_token: token.value }
  }

  const proof = await challengeRef.value?.verifyAction()
  if (!proof) {
    errorMessage.value = t('checkin.completeVerification')
    return null
  }
  if (tencentEnabled.value) {
    return {
      tencent_captcha_ticket: proof.token,
      tencent_captcha_randstr: proof.randstr,
    }
  }
  return { turnstile_token: proof.token }
}

function reset(): void {
  token.value = ''
  randstr.value = ''
  errorMessage.value = ''
  challengeRef.value?.reset()
}

defineExpose({ resolveProof, reset })
</script>
