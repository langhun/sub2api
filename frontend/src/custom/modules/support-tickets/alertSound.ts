import frogCroakUrl from './assets/frog-croak.oga?url'

let frogCroak: HTMLAudioElement | undefined
let stopTimer: ReturnType<typeof setTimeout> | undefined

function stopAfterSingleCroak(audio: HTMLAudioElement) {
  if (stopTimer) clearTimeout(stopTimer)
  stopTimer = setTimeout(() => {
    stopTimer = undefined
    audio.pause()
    audio.currentTime = 0
  }, 900)
}

function getFrogCroak(): HTMLAudioElement | undefined {
  if (typeof window === 'undefined' || typeof Audio === 'undefined') return undefined
  if (!frogCroak) {
    frogCroak = new Audio(frogCroakUrl)
    frogCroak.preload = 'auto'
    frogCroak.volume = 0.8
  }
  return frogCroak
}

/** Play one recorded frog croak when a ticket receives a new external reply. */
export async function playSupportTicketAlertSound(): Promise<void> {
  const audio = getFrogCroak()
  if (!audio) return

  try {
    if (stopTimer) clearTimeout(stopTimer)
    audio.pause()
    audio.currentTime = 0
    await audio.play()
    stopAfterSingleCroak(audio)
  } catch {
    // Browsers can block sound until the user has clicked the preview control once.
  }
}
