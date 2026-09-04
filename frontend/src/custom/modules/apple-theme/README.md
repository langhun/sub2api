# Apple Theme Module

这是一个受 Apple 设计启发的主题模块，为 sub2api 前端提供玻璃态效果和流畅动画。

## 功能特性

- **双主题系统**：支持原始默认主题和 Apple Glass 主题
- **明暗模式**：支持浅色、深色和自动跟随系统
- **玻璃态效果**：毛玻璃背景和半透明卡片
- **环境渐变**：动态背景渐变和呼吸动画
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
├── __tests__/                       # 测试文件目录（待添加）
├── settings.ts                      # 模块设置配置
├── theme.css                        # Apple 主题样式
├── ThemeSwitcher.vue               # 主题切换组件
└── useTheme.ts                     # 主题状态管理
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

1. **玻璃态背景**
   - 毛玻璃模糊效果（14px blur）
   - 半透明卡片（42% opacity）
   - backdrop-filter 支持

2. **配色方案**
   - 浅色模式：
     - 背景：#F4F8FD（浅蓝灰色）
     - 主色：#0071e3（Apple 蓝）
     - 卡片：白色半透明
   
   - 深色模式：
     - 背景：#0C121B（深蓝黑色）
     - 主色：#2997ff（亮蓝色）
     - 卡片：深灰半透明

3. **环境效果**
   - 动态径向渐变背景
   - 呼吸动画（9秒循环）
   - 响应式布局

4. **圆角半径**
   - 全局：0.625rem (10px)
   - 统一的视觉语言

## 技术实现

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
const { themeMode, themeStyle, effectiveMode, setThemeMode, setThemeStyle } = useTheme()
```

- `themeMode`: 'light' | 'dark' | 'auto'
- `themeStyle`: 'default' | 'apple'
- `effectiveMode`: 实际生效的模式（auto 会解析为 light 或 dark）

### 本地存储

主题选择自动保存到 localStorage：

- `theme-mode`: 保存用户的模式选择
- `theme-style`: 保存用户的风格选择

## 兼容性

- 需要浏览器支持 `backdrop-filter`
- 需要 CSS 变量支持
- 需要 `prefers-color-scheme` 媒体查询支持（自动模式）

对于不支持的浏览器，会优雅降级到不透明背景。

## 扩展开发

### 添加新主题风格

1. 在 `theme.css` 中添加新的 CSS 变量定义：

```css
:root[data-theme='your-theme'] {
  /* 定义变量 */
}
```

2. 在 `useTheme.ts` 中添加类型：

```typescript
export type ThemeStyle = 'default' | 'apple' | 'your-theme'
```

3. 在 `ThemeSwitcher.vue` 中添加选项：

```typescript
const themeStyles = [
  // ... existing styles
  { value: 'your-theme' as ThemeStyle, label: 'theme.styleYourTheme', icon: 'sparkles' as const },
]
```

4. 添加国际化文本到 `locales/` 文件中

### 测试

TODO: 添加单元测试和 E2E 测试

## 注意事项

1. **不要修改布局**：本模块只改变配色和视觉效果，不改动原有布局结构
2. **保留原主题**：默认主题始终可用，用户可以随时切换回去
3. **性能考虑**：`backdrop-filter` 可能影响性能，建议在性能较好的设备上使用
4. **渐进增强**：不支持的浏览器会降级到纯色背景

## 参考资料

- 设计灵感来源：[c3api](https://github.com/is7Qin/c3api)
- Apple Human Interface Guidelines
- Material Design Glass Morphism

## 更新日志

### v1.0.0 (2026-09-04)

- 初始版本
- 支持 Apple Glass 主题
- 支持明暗模式切换
- 完整的国际化支持
- 模块化设计实现
