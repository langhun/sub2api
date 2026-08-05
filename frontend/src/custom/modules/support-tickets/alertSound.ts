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

function playFrogCroak(context: AudioContext, startAt: number) {
  const endAt = startAt + 0.24
  const gain = context.createGain()
  const low = context.createOscillator()
  const high = context.createOscillator()
  const highGain = context.createGain()

  low.type = 'triangle'
  low.frequency.setValueAtTime(190, startAt)
  low.frequency.exponentialRampToValueAtTime(105, endAt)
  high.type = 'sine'
  high.frequency.setValueAtTime(380, startAt)
  high.frequency.exponentialRampToValueAtTime(185, endAt)
  highGain.gain.setValueAtTime(0.3, startAt)

  gain.gain.setValueAtTime(0.0001, startAt)
  gain.gain.exponentialRampToValueAtTime(0.12, startAt + 0.018)
  gain.gain.exponentialRampToValueAtTime(0.0001, endAt)

  low.connect(gain)
  high.connect(highGain)
  highGain.connect(gain)
  gain.connect(context.destination)
  low.start(startAt)
  high.start(startAt)
  low.stop(endAt)
  high.stop(endAt)
}

/** Play a short frog-like alert when a ticket receives a new external reply. */
export async function playSupportTicketAlertSound(): Promise<void> {
  const Constructor = getAudioContextConstructor()
  if (!Constructor) return

  if (!audioContext || audioContext.state === 'closed') {
    audioContext = new Constructor()
  }

  try {
    if (audioContext.state === 'suspended') await audioContext.resume()
    if (audioContext.state !== 'running') return

    playFrogCroak(audioContext, audioContext.currentTime + 0.01)
  } catch {
    // Autoplay policies can block audio until the user has interacted with the page.
  }
}
