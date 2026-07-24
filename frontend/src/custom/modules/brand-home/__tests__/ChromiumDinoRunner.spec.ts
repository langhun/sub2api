import { existsSync, readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'

const { disposeInstance, initializeInstance } = vi.hoisted(() => ({
  disposeInstance: vi.fn(),
  initializeInstance: vi.fn(),
}))

vi.mock('@/custom/modules/brand-home/chromiumDino/offline', () => ({
  Runner: { disposeInstance, initializeInstance },
}))

import ChromiumDinoRunner from '../ChromiumDinoRunner.vue'

const assetDirectory = resolve(process.cwd(), 'public/custom/brand-home')

describe('ChromiumDinoRunner', () => {
  afterEach(() => {
    disposeInstance.mockReset()
    initializeInstance.mockReset()
  })

  it('uses module-owned static assets and releases the singleton when unmounted', async () => {
    const wrapper = mount(ChromiumDinoRunner)

    await flushPromises()

    expect(initializeInstance).toHaveBeenCalledWith('.chromium-dino-runner')
    expect(wrapper.get('#offline-resources-1x').attributes('src')).toBe('/custom/brand-home/chromium-offline-sprite.png')
    expect(wrapper.get('#offline-resources-2x').attributes('src')).toBe('/custom/brand-home/chromium-offline-sprite-2x.png')

    const audioResources = wrapper.get('#audio-resources').element as HTMLTemplateElement
    expect(audioResources.content.querySelector('#offline-sound-press')?.getAttribute('src')).toBe('/custom/brand-home/button-press.mp3')
    expect(audioResources.content.querySelector('#offline-sound-hit')?.getAttribute('src')).toBe('/custom/brand-home/hit.mp3')
    expect(audioResources.content.querySelector('#offline-sound-reached')?.getAttribute('src')).toBe('/custom/brand-home/score-reached.mp3')

    wrapper.unmount()

    expect(disposeInstance).toHaveBeenCalledOnce()
  })

  it('keeps all Chromium assets and the license with the module public files', () => {
    for (const filename of [
      'button-press.mp3',
      'chromium-offline-sprite-2x.png',
      'chromium-offline-sprite.png',
      'hit.mp3',
      'score-reached.mp3',
      'chromium-dino-LICENSE',
    ]) {
      expect(existsSync(resolve(assetDirectory, filename))).toBe(true)
    }

    expect(readFileSync(resolve(assetDirectory, 'chromium-dino-LICENSE'), 'utf8')).toContain('Chromium Authors')
  })
})
