# WebGL流体背景性能优化方案

## 问题分析

当前 `fluidShader.ts` 中的渲染循环以60fps全速运行,在以下情况下会造成性能问题:
1. 低端移动设备
2. 多标签页场景
3. 电池供电设备

## 优化方案

### 方案1: FPS限制器 (推荐)

限制渲染帧率到30fps,肉眼几乎无感,但GPU占用降低50%。

```typescript
// fluidShader.ts 修改
export function attachFluidShader(canvas: HTMLCanvasElement, params: FluidParams) {
  // ... 现有代码

  let lastFrameTime = 0
  const TARGET_FPS = 30
  const FRAME_INTERVAL = 1000 / TARGET_FPS

  function render(currentTime: number) {
    if (!canvas.isConnected) return

    const delta = currentTime - lastFrameTime
    if (delta < FRAME_INTERVAL) {
      rafId = requestAnimationFrame(render)
      return
    }
    lastFrameTime = currentTime - (delta % FRAME_INTERVAL)

    // 原有渲染逻辑
    gl.bindFramebuffer(gl.FRAMEBUFFER, pingpongFBO)
    gl.viewport(0, 0, internalWidth, internalHeight)
    gl.useProgram(flowProgram)
    // ... 其余代码不变

    rafId = requestAnimationFrame(render)
  }

  rafId = requestAnimationFrame(render)
  
  // ...
}
```

**预期效果**:
- GPU占用: 60% → 30%
- 功耗: 减少约40%
- 视觉: 几乎无差异 (流体动画本身很慢)

---

### 方案2: 空闲暂停 (激进优化)

用户无操作5秒后暂停渲染,鼠标移动/点击时恢复。

```typescript
export function attachFluidShader(canvas: HTMLCanvasElement, params: FluidParams) {
  // ... 现有代码

  let isPaused = false
  let idleTimer: number | undefined
  const IDLE_DELAY = 5000 // 5秒无操作暂停

  function resetIdleTimer() {
    if (isPaused) {
      isPaused = false
      rafId = requestAnimationFrame(render)
    }
    if (idleTimer !== undefined) window.clearTimeout(idleTimer)
    idleTimer = window.setTimeout(() => {
      isPaused = true
    }, IDLE_DELAY)
  }

  function render(currentTime: number) {
    if (!canvas.isConnected || isPaused) return

    // 原有渲染逻辑
    // ...

    rafId = requestAnimationFrame(render)
  }

  // 监听用户活动
  window.addEventListener('mousemove', resetIdleTimer, { passive: true })
  window.addEventListener('mousedown', resetIdleTimer, { passive: true })
  window.addEventListener('touchstart', resetIdleTimer, { passive: true })
  window.addEventListener('wheel', resetIdleTimer, { passive: true })

  rafId = requestAnimationFrame(render)
  resetIdleTimer()

  return {
    setParams,
    dispose: () => {
      // 清理监听器
      window.removeEventListener('mousemove', resetIdleTimer)
      window.removeEventListener('mousedown', resetIdleTimer)
      window.removeEventListener('touchstart', resetIdleTimer)
      window.removeEventListener('wheel', resetIdleTimer)
      if (idleTimer !== undefined) window.clearTimeout(idleTimer)
      // ... 原有dispose逻辑
    }
  }
}
```

**预期效果**:
- 空闲时GPU占用: 0%
- 活跃时与原版相同
- 适合长时间打开但不操作的场景

---

### 方案3: 设备性能分级 (智能优化)

检测设备性能,自动调整策略:
- 高端设备: 60fps
- 中端设备: 30fps
- 低端设备: 禁用流体背景

```typescript
function detectPerformanceTier(): 'high' | 'medium' | 'low' {
  // 检测硬件并发数 (CPU核心)
  const cores = navigator.hardwareConcurrency || 2
  
  // 检测内存 (如果可用)
  const memory = (navigator as any).deviceMemory || 4
  
  // 检测GPU (WebGL参数)
  const canvas = document.createElement('canvas')
  const gl = canvas.getContext('webgl2')
  if (!gl) return 'low'
  
  const debugInfo = gl.getExtension('WEBGL_debug_renderer_info')
  const renderer = debugInfo ? gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL) : ''
  
  // 简单分级逻辑
  if (cores >= 8 && memory >= 8) return 'high'
  if (cores >= 4 && memory >= 4) return 'medium'
  return 'low'
}

export function attachFluidShader(canvas: HTMLCanvasElement, params: FluidParams) {
  const tier = detectPerformanceTier()
  
  if (tier === 'low') {
    // 返回空实现,不启用流体背景
    return {
      setParams: () => {},
      dispose: () => {}
    }
  }

  const TARGET_FPS = tier === 'high' ? 60 : 30
  const FRAME_INTERVAL = 1000 / TARGET_FPS

  // ... 其余代码
}
```

---

## 实施建议

### 阶段1: 立即实施 (30分钟)
✅ **方案1: FPS限制器**  
- 改动最小,风险最低
- 效果明显,兼容性好

### 阶段2: 观察1周后实施 (1小时)
⚙️ **方案3: 设备性能分级**  
- 收集用户反馈
- 分析真实设备数据
- 调整分级阈值

### 阶段3: 可选实施 (2小时)
💡 **方案2: 空闲暂停**  
- 适合追求极致省电的场景
- 需要充分测试用户体验

---

## 性能对比预测

| 方案 | GPU占用 | 功耗 | 视觉差异 | 实施难度 |
|------|--------|------|---------|---------|
| 当前 (60fps) | 100% | 100% | 基准 | - |
| 方案1 (30fps) | 50% | 60% | 几乎无 | ⭐ 简单 |
| 方案2 (空闲暂停) | 5%* | 20%* | 无 | ⭐⭐ 中等 |
| 方案3 (分级) | 25-100%** | 40-100%** | 无 | ⭐⭐⭐ 复杂 |

\*: 假设80%时间处于空闲  
\*\*: 取决于设备分布

---

## 监控指标

实施后建议监控:

```javascript
// 添加性能监控
let frameCount = 0
let lastFpsTime = performance.now()

function render(currentTime: number) {
  // ... 渲染逻辑

  frameCount++
  if (currentTime - lastFpsTime >= 1000) {
    const fps = Math.round((frameCount * 1000) / (currentTime - lastFpsTime))
    console.log(`[FluidCanvas] FPS: ${fps}`)
    frameCount = 0
    lastFpsTime = currentTime
  }

  rafId = requestAnimationFrame(render)
}
```

---

## 结论

**推荐立即实施方案1 (FPS限制器)**:
- ✅ 代码改动 < 10行
- ✅ 性能提升约50%
- ✅ 几乎无副作用

后续根据数据决定是否实施方案2或方案3。
