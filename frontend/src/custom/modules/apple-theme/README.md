# Apple Theme Module

这是一个受 Apple 设计启发的主题模块，为 sub2api 前端提供**玻璃态效果**、**WebGL 流体动画**和**流畅交互**。

## 功能特性

- **双主题系统**：支持原始默认主题和 Apple Glass 主题
- **明暗模式**：支持浅色、深色和自动跟随系统
- **玻璃态效果**：毛玻璃背景和半透明卡片
- **🌊 WebGL 流体动画**：实时流体模拟背景特效
- **鼠标交互**：跟随鼠标的流体扰动效果
- **环境渐变**：动态背景渐变和呼吸动画
- **性能优化**：30fps 限制、低功耗模式、响应式设计
- **可访问性**：支持 prefers-reduced-motion
- **完整国际化**：中英文支持
- **模块化设计**：符合项目的模块组织规范

## 文件结构

```
frontend/src/custom/modules/apple-theme/
├── admin/
│   └── AppleThemeSettingsPanel.vue  # 管理员设置面板
├── locales/
│   ├── en.ts                        # 英文翻译
│   └── zh.ts                        # 中文翻译
├── __tests__/                       # 测试文件
│   ├── useTheme.spec.ts
│   ├── ThemeSwitcher.spec.ts
│   └── AppleThemeSettingsPanel.spec.ts
├── settings.ts                      # 模块设置配置
├── theme.css                        # Apple 主题样式
├── ThemeSwitcher.vue               # 主题切换组件
├── FluidCanvas.vue                 # WebGL 流体 Canvas 组件
├── fluidShader.ts                  # WebGL 流体着色器实现
├── useTheme.ts                     # 主题状态管理
└── README.md                       # 本文档
```

## 使用方式

### 1. 管理员配置

在管理后台的"站点设置"中：

1. 找到"Apple Theme"设置项
2. 开启"启用 Apple 主题"开关
3. 选择默认主题模式（自动/浅色/深色）
4. 保存设置

### 2. 用户切换主题

用户可以在页面顶部导航栏中：

1. 点击主题切换按钮（太阳/月亮图标）
2. 在下拉菜单中选择：
   - **主题风格**：默认 / Apple Glass
   - **外观模式**：浅色 / 深色 / 自动

用户的主题选择会保存到浏览器本地存储。

## 主题特性

### Apple Glass 主题特点

1. **🌊 WebGL 流体动画**
   - 完整的 WebGL2 流体模拟
   - 实时计算的流体物理
   - 鼠标/触控交互响应
   - 自动跟随系统主题切换配色
   - 性能优化：
     - 30fps 刷新率限制
     - 低功耗模式（WebGL low-power）
     - 响应式分辨率（最高 1.5x DPR）
     - prefers-reduced-motion 支持

2. **玻璃态背景**
   - 毛玻璃模糊效果（14px blur）
   - 半透明卡片（42% opacity）
   - backdrop-filter 支持

3. **配色方案**
   - 浅色模式：
     - 背景：#F4F8FD（浅蓝灰色）
     - 主色：#0071e3（Apple 蓝）
     - 流体：青绿色渐变
     - 卡片：白色半透明

   - 深色模式：
     - 背景：#0C121B（深蓝黑色）
     - 主色：#2997ff（亮蓝色）
     - 流体：深蓝色渐变
     - 卡片：深灰半透明

4. **环境效果**
   - 动态径向渐变背景
   - 呼吸动画（9秒循环）
   - 响应式布局

5. **圆角半径**
   - 全局：0.625rem (10px)
   - 统一的视觉语言

## 技术实现

### WebGL 流体模拟

基于 WebGL2 实现的实时流体模拟系统：

**着色器系统：**
- **Flow Shader**：流体物理模拟
  - 速度场传播
  - 衰减和扩散
  - 鼠标/触控输入响应

- **Display Shader**：视觉渲染
  - 噪声生成和扭曲
  - 涡旋效果
  - 多色渐变混合
  - 基于 flowmap 的动态响应

**性能优化：**
```typescript
// 30fps 刷新率限制
const step = 1000 / 30

// 设备像素比上限
const dprCap = Math.min(window.devicePixelRatio || 1, 1.5)

// 流体模拟分辨率（主分辨率的 1/4）
flowWidth = Math.round(width / 4)
flowHeight = Math.round(height / 4)

// 低功耗模式
powerPreference: 'low-power'
```

**可访问性：**
```typescript
// 禁用动画偏好检测
if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
  // 渲染一帧后停止动画
  frame(performance.now())
  cancelAnimationFrame(raf)
}
```

### CSS 变量系统

主题使用 CSS 变量实现，通过 `data-theme` 属性切换：

```css
:root[data-theme='apple'] {
  --glass-blur: 14px;
  --apple-primary: #0071e3;
  /* ... more variables */
}

:root[data-theme='apple'].dark {
  --apple-primary: #2997ff;
  /* ... dark mode variables */
}
```

### 状态管理

使用 Vue 3 Composition API：

```typescript
const { themeMode, themeStyle, effectiveMode, setThemeMode } = useTheme()
```

- `themeMode`: 'light' | 'dark' | 'auto'
- `themeStyle`: 'default' | 'apple'
- `effectiveMode`: 实际生效的模式（auto 会解析为 light 或 dark）

### 配置所有权

主题风格由管理后台的公开设置统一控制：

- `enable_apple_theme`: 是否启用 Apple Theme
- `default_theme_mode`: 没有用户偏好时的默认明暗模式
- `theme_style`: 仅用于兼容旧版本的 `glass/classic` 设置

浏览器只在 `localStorage.theme` 中保存用户选择的明暗模式，不保存主题风格。

## 兼容性

- **WebGL2**：需要浏览器支持 WebGL2（大部分现代浏览器）
- **Backdrop Filter**：需要支持 `backdrop-filter`（Safari 9+，Chrome 76+，Firefox 103+）
- **CSS 变量**：需要 CSS 自定义属性支持
- **媒体查询**：需要 `prefers-color-scheme` 和 `prefers-reduced-motion` 支持

对于不支持的浏览器：
- WebGL2 不可用时会优雅降级到静态渐变背景
- backdrop-filter 不支持时会降级到不透明背景
- 所有降级都是自动的，不影响基本使用

## 性能考虑

1. **WebGL 流体**
   - 30fps 刷新率限制（避免过度消耗）
   - 低功耗模式（GPU 节能）
   - 流体模拟分辨率为显示分辨率的 1/4
   - 自动响应 prefers-reduced-motion
   - 触摸设备和粗指针设备默认禁用鼠标跟随，Windows 桌面精确指针保持鼠标交互

2. **渲染优化**
   - 使用 CSS 变量避免重绘
   - backdrop-filter 硬件加速
   - 固定位置元素避免回流

3. **内存管理**
   - WebGL 资源正确释放
   - 组件卸载时清理监听器
   - 帧缓冲区复用

## 扩展开发

### 调整全局主题风格

1. 在 `theme.css` 中添加新的 CSS 变量定义：

```css
:root[data-theme='your-theme'] {
  /* 定义变量 */
}
```

2. 在 `useTheme.ts` 中更新服务端设置到主题样式的映射。

```typescript
export type ThemeStyle = 'default' | 'apple' | 'your-theme'
```

3. 在后台设置模块中增加对应选项和持久化字段；不要添加用户侧主题风格切换入口。

### 自定义流体参数

可以在 `FluidCanvas.vue` 中调整流体效果：

```typescript
const LIGHT_PARAMS: FluidParams = {
  mouseRadius: 0.22,      // 鼠标影响半径
  mouseStrength: 1.1,     // 鼠标影响强度
  decay: 0.96,            // 衰减速度（越小衰减越快）
  speed: 18,              // 流体速度
  distortion: 22,         // 扭曲强度
  swirl: 14,              // 涡旋强度
  color1: '#4A9E8E',      // 流体颜色 1
  color2: '#7EC8BB',      // 流体颜色 2
  color3: '#E0F2EF',      // 流体颜色 3
  // ... 更多参数
}
```

### 测试

```bash
# 运行单元测试
npm run test

# 测试覆盖率
npm run test:coverage
```

已包含的测试：
- `useTheme.spec.ts` - 主题状态管理测试
- `ThemeSwitcher.spec.ts` - 主题切换组件测试
- `AppleThemeSettingsPanel.spec.ts` - 设置面板测试

## 注意事项

1. **不修改布局**：本模块只改变配色、视觉效果和背景，不改动原有布局结构
2. **保留原主题**：默认主题始终可用，用户可以随时切换回去
3. **性能考虑**：
   - WebGL 流体在移动设备上可能影响电池续航
   - 可以在设置中提供"性能模式"选项
   - 自动响应 prefers-reduced-motion 设置
4. **渐进增强**：
   - WebGL2 不可用时自动降级到静态渐变
   - backdrop-filter 不支持时降级到纯色背景
5. **浏览器兼容性**：
   - 推荐使用 Chrome 76+、Safari 14+、Firefox 103+
   - IE 浏览器不支持，会显示默认主题

## 参考资料

- 设计灵感来源：[c3api](https://github.com/is7Qin/c3api)
- WebGL 流体模拟：[Fluid Simulation for Dummies](https://mikeash.com/pyblog/fluid-simulation-for-dummies.html)
- Apple Human Interface Guidelines
- Material Design Glass Morphism

## 更新日志

### v1.1.0 (2026-09-04)

- ✨ 添加 WebGL 流体动画特效
- ✨ 支持鼠标/触控交互
- ✨ 性能优化和可访问性支持
- 🔧 添加完整的单元测试
- 📝 完善文档和注释

### v1.0.0 (2026-09-04)

- 初始版本
- 支持 Apple Glass 主题
- 支持明暗模式切换
- 完整的国际化支持
- 模块化设计实现
