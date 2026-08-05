type AudioContextConstructor = new () => AudioContext

let audioContext: AudioContext | undefined

function getAudioContextConstructor(): AudioContextConstructor | undefined {
  if (typeof window === 'undefined') return undefined
  const current = window as unknown as {
    AudioContext?: AudioContextConstructor
    webkitAudioContext?: AudioContextConstructor
  }
  return current.AudioContext ?? current.webkitAudioContext
}

function playTone(context: AudioContext, startAt: number, frequency: number) {
  const oscillator = context.createOscillator()
  const gain = context.createGain()
  const endAt = startAt + 0.09

  oscillator.type = 'sine'
  oscillator.frequency.setValueAtTime(frequency, startAt)
  gain.gain.setValueAtTime(0.0001, startAt)
  gain.gain.exponentialRampToValueAtTime(0.08, startAt + 0.01)
  gain.gain.exponentialRampToValueAtTime(0.0001, endAt)

  oscillator.connect(gain)
  gain.connect(context.destination)
  oscillator.start(startAt)
  oscillator.stop(endAt)
}

/** Play a short in-page alert when a ticket receives a new external reply. */
export async function playSupportTicketAlertSound(): Promise<void> {
  const Constructor = getAudioContextConstructor()
  if (!Constructor) return

  if (!audioContext || audioContext.state === 'closed') {
    audioContext = new Constructor()
  }

  try {
    if (audioContext.state === 'suspended') await audioContext.resume()
    if (audioContext.state !== 'running') return

    const startAt = audioContext.currentTime + 0.01
    playTone(audioContext, startAt, 740)
    playTone(audioContext, startAt + 0.12, 920)
  } catch {
    // Autoplay policies can block audio until the user has interacted with the page.
  }
}
