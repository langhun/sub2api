import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { attachFluidShader, SITE_FLUID_PARAMS, type FluidShaderHandle } from '../fluidShader'

function mockWebGL() {
  return {
    VERTEX_SHADER: 1, FRAGMENT_SHADER: 2, COMPILE_STATUS: 3, LINK_STATUS: 4,
    ARRAY_BUFFER: 5, STATIC_DRAW: 6, FLOAT: 7, TEXTURE_2D: 8, RGBA: 9,
    UNSIGNED_BYTE: 10, TEXTURE_MIN_FILTER: 11, TEXTURE_MAG_FILTER: 12,
    TEXTURE_WRAP_S: 13, TEXTURE_WRAP_T: 14, LINEAR: 15, CLAMP_TO_EDGE: 16,
    FRAMEBUFFER: 17, COLOR_ATTACHMENT0: 18, FRAMEBUFFER_COMPLETE: 19,
    TEXTURE0: 20, TRIANGLE_STRIP: 21,
    createShader: vi.fn((): object | null => ({})),
    shaderSource: vi.fn(), compileShader: vi.fn(),
    getShaderParameter: vi.fn(() => true), getShaderInfoLog: vi.fn(), deleteShader: vi.fn(),
    createProgram: vi.fn((): object | null => ({})),
    attachShader: vi.fn(), linkProgram: vi.fn(),
    getProgramParameter: vi.fn(() => true), getProgramInfoLog: vi.fn(), deleteProgram: vi.fn(),
    getUniformLocation: vi.fn((_program: object, name: string) => name),
    createBuffer: vi.fn((): object | null => ({})),
    bindBuffer: vi.fn(), bufferData: vi.fn(), deleteBuffer: vi.fn(),
    getAttribLocation: vi.fn(() => 0), enableVertexAttribArray: vi.fn(), vertexAttribPointer: vi.fn(),
    createTexture: vi.fn((): object | null => ({})),
    bindTexture: vi.fn(), texImage2D: vi.fn(), texParameteri: vi.fn(), deleteTexture: vi.fn(),
    createFramebuffer: vi.fn((): object | null => ({})),
    bindFramebuffer: vi.fn(), framebufferTexture2D: vi.fn(), deleteFramebuffer: vi.fn(),
    checkFramebufferStatus: vi.fn(() => 19),
    viewport: vi.fn(), useProgram: vi.fn(), activeTexture: vi.fn(),
    uniform1i: vi.fn(), uniform2f: vi.fn(), uniform1f: vi.fn(), uniform4f: vi.fn(), drawArrays: vi.fn(),
  }
}

describe('fluid shader lifecycle', () => {
  let gl: ReturnType<typeof mockWebGL>
  let canvas: HTMLCanvasElement
  let handle: FluidShaderHandle | undefined
  let frames: Map<number, FrameRequestCallback>
  let motionListener: ((event: MediaQueryListEvent) => void) | undefined
  let resizeCallback: ResizeObserverCallback | undefined
  let reducedMotion: boolean
  let disconnect: ReturnType<typeof vi.fn>
  let removeMotionListener: ReturnType<typeof vi.fn>

  beforeEach(() => {
    gl = mockWebGL()
    canvas = document.createElement('canvas')
    Object.defineProperties(canvas, {
      clientWidth: { configurable: true, value: 800 },
      clientHeight: { configurable: true, value: 600 },
    })
    vi.spyOn(canvas, 'getContext').mockReturnValue(gl as unknown as WebGL2RenderingContext)
    vi.spyOn(console, 'error').mockImplementation(() => {})
    vi.stubGlobal('devicePixelRatio', 1)
    frames = new Map()
    let id = 0
    vi.stubGlobal('requestAnimationFrame', vi.fn((callback: FrameRequestCallback) => {
      frames.set(++id, callback)
      return id
    }))
    vi.stubGlobal('cancelAnimationFrame', vi.fn((id: number) => frames.delete(id)))
    reducedMotion = true
    motionListener = undefined
    removeMotionListener = vi.fn()
    vi.stubGlobal('matchMedia', vi.fn((query: string) => ({
      matches: query.includes('reduced-motion') && reducedMotion,
      addEventListener: (_type: string, handler: (event: MediaQueryListEvent) => void) => {
        if (query.includes('reduced-motion')) motionListener = handler
      },
      removeEventListener: removeMotionListener,
    })))
    disconnect = vi.fn()
    vi.stubGlobal('ResizeObserver', class {
      constructor(callback: ResizeObserverCallback) { resizeCallback = callback }
      observe = vi.fn()
      disconnect = disconnect
    })
  })

  afterEach(() => {
    handle?.dispose()
    handle = undefined
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })

  it('draws once and applies palette changes immediately with reduced motion', () => {
    handle = attachFluidShader(canvas, SITE_FLUID_PARAMS)
    expect(gl.drawArrays).toHaveBeenCalledTimes(2)
    expect(frames.size).toBe(0)

    handle.setParams({ ...SITE_FLUID_PARAMS, color1: '#FF0000' })

    expect(gl.drawArrays).toHaveBeenCalledTimes(4)
    expect(gl.uniform4f).toHaveBeenCalledWith('u_color1', 1, 0, 0, 1)
    expect(gl.uniform1f.mock.calls.filter(([name]) => name === 'u_time')).toEqual([
      ['u_time', 0], ['u_time', 0],
    ])
    expect(frames.size).toBe(0)
  })

  it('redraws and resizes both the canvas and flow textures without starting animation', () => {
    handle = attachFluidShader(canvas, SITE_FLUID_PARAMS)
    Object.defineProperty(canvas, 'clientWidth', { configurable: true, value: 400 })
    window.dispatchEvent(new Event('resize'))

    expect(canvas.width).toBe(400)
    expect(gl.viewport).toHaveBeenLastCalledWith(0, 0, 400, 600)
    expect(gl.texImage2D).toHaveBeenLastCalledWith(gl.TEXTURE_2D, 0, gl.RGBA, 100, 150, 0, gl.RGBA, gl.UNSIGNED_BYTE, expect.any(Uint8Array))
    expect(gl.drawArrays).toHaveBeenCalledTimes(4)

    Object.defineProperty(canvas, 'clientHeight', { configurable: true, value: 300 })
    resizeCallback?.([], {} as ResizeObserver)
    expect(canvas.height).toBe(300)
    expect(gl.drawArrays).toHaveBeenCalledTimes(6)
    expect(frames.size).toBe(0)
  })

  it('starts and stops the animation when reduced motion changes', () => {
    reducedMotion = false
    handle = attachFluidShader(canvas, SITE_FLUID_PARAMS)
    expect(frames.size).toBe(1)

    const [id, callback] = [...frames.entries()][0]
    frames.delete(id)
    callback(performance.now() + 40)
    expect(gl.drawArrays).toHaveBeenCalledTimes(4)
    expect(frames.size).toBe(1)

    motionListener?.({ matches: true } as MediaQueryListEvent)
    expect(frames.size).toBe(0)
    expect(gl.drawArrays).toHaveBeenCalledTimes(6)
    motionListener?.({ matches: false } as MediaQueryListEvent)
    expect(frames.size).toBe(1)
    motionListener?.({ matches: false } as MediaQueryListEvent)
    expect(frames.size).toBe(1)
  })

  it('uses nonzero texture dimensions for a hidden or not yet sized canvas', () => {
    Object.defineProperty(canvas, 'clientWidth', { configurable: true, value: 0 })
    Object.defineProperty(canvas, 'clientHeight', { configurable: true, value: 0 })
    handle = attachFluidShader(canvas, SITE_FLUID_PARAMS)
    expect(canvas.width).toBe(1)
    expect(canvas.height).toBe(1)
    expect(gl.texImage2D).toHaveBeenCalledWith(gl.TEXTURE_2D, 0, gl.RGBA, 1, 1, 0, gl.RGBA, gl.UNSIGNED_BYTE, expect.any(Uint8Array))
  })

  it('releases every resource and listener once and ignores calls after disposal', () => {
    reducedMotion = false
    handle = attachFluidShader(canvas, SITE_FLUID_PARAMS)
    const removeListener = vi.spyOn(window, 'removeEventListener')
    handle.dispose()
    handle.dispose()
    expect(gl.deleteShader).toHaveBeenCalledTimes(4)
    expect(gl.deleteProgram).toHaveBeenCalledTimes(2)
    expect(gl.deleteBuffer).toHaveBeenCalledTimes(1)
    expect(gl.deleteFramebuffer).toHaveBeenCalledTimes(2)
    expect(gl.deleteTexture).toHaveBeenCalledTimes(2)
    expect(disconnect).toHaveBeenCalledTimes(1)
    expect(removeMotionListener).toHaveBeenCalledWith('change', motionListener)
    expect(removeListener).toHaveBeenCalledWith('resize', expect.any(Function))
    expect(removeListener).toHaveBeenCalledWith('mousemove', expect.any(Function))
    expect(frames.size).toBe(0)

    gl.drawArrays.mockClear()
    handle.setParams(SITE_FLUID_PARAMS)
    handle.stir(0, 0, 1, 1)
    window.dispatchEvent(new Event('resize'))
    resizeCallback?.([], {} as ResizeObserver)
    motionListener?.({ matches: false } as MediaQueryListEvent)
    expect(gl.drawArrays).not.toHaveBeenCalled()
    expect(frames.size).toBe(0)
  })

  it.each(['shader', 'compile', 'program', 'link', 'buffer', 'texture', 'framebuffer', 'incomplete'] as const)(
    'cleans up partial initialization after a %s failure', (failure) => {
      if (failure === 'shader') gl.createShader.mockReturnValueOnce(null)
      if (failure === 'compile') gl.getShaderParameter.mockReturnValueOnce(false)
      if (failure === 'program') gl.createProgram.mockReturnValueOnce(null)
      if (failure === 'link') gl.getProgramParameter.mockReturnValueOnce(false)
      if (failure === 'buffer') gl.createBuffer.mockReturnValueOnce(null)
      if (failure === 'texture') gl.createTexture.mockReturnValueOnce({}).mockReturnValueOnce(null)
      if (failure === 'framebuffer') gl.createFramebuffer.mockReturnValueOnce({}).mockReturnValueOnce(null)
      if (failure === 'incomplete') gl.checkFramebufferStatus.mockReturnValueOnce(0)
      handle = attachFluidShader(canvas, SITE_FLUID_PARAMS)
      handle.dispose()

      for (const [create, remove] of [
        [gl.createShader, gl.deleteShader], [gl.createProgram, gl.deleteProgram],
        [gl.createBuffer, gl.deleteBuffer], [gl.createTexture, gl.deleteTexture],
        [gl.createFramebuffer, gl.deleteFramebuffer],
      ]) {
        for (const { value } of create.mock.results) {
          if (value !== null) expect(remove.mock.calls.filter(([resource]) => resource === value)).toHaveLength(1)
        }
      }
      expect(gl.drawArrays).not.toHaveBeenCalled()
      expect(frames.size).toBe(0)
    },
  )

  it('falls back without throwing when WebGL is unavailable', () => {
    vi.mocked(canvas.getContext).mockReturnValue(null)
    handle = attachFluidShader(canvas, SITE_FLUID_PARAMS)
    expect(() => {
      handle?.setParams(SITE_FLUID_PARAMS)
      handle?.stir(0, 0, 1, 1)
      handle?.dispose()
    }).not.toThrow()
    expect(console.error).not.toHaveBeenCalled()
  })
})
