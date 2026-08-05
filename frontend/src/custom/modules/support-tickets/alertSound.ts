import frogCroakUrl from './assets/frog-croak.oga?url'

let frogCroak: HTMLAudioElement | undefined

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
    audio.pause()
    audio.currentTime = 0
    await audio.play()
  } catch {
    // Browsers can block sound until the user has clicked the preview control once.
  }
}
