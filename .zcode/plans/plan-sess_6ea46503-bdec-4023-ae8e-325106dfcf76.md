# 执行 Apple 级液态玻璃与原主题完全隔离方案

## 1. 还原上游共享组件
- 还原 `AppLayout.vue`、`AuthLayout.vue`、`AppSidebar.vue`、`AppHeader.vue` 的模板与原生 class（包括 `bg-mesh-gradient`、`w-64`、`lg:ml-64`、`rounded-2xl` 等），完全保留上游经典主题视觉，消除组件级硬编码侵入。

## 2. 重构流体着色器与纯净配色
- 更新 `FluidCanvas.vue`：
  - 浅色模式（晨曦极光蓝紫）：`color1: '#3A88E9'`、`color2: '#8E7BF5'`、`color3: '#EAF2FF'`
  - 深色模式（深空星云蓝紫）：`color1: '#0D3880'`、`color2: '#2C1A6E'`、`color3: '#121E36'`
- 更新 `lib/fluid-shader.ts`：开放全平台鼠标搅拌交互，调整流体阻尼与弹性，打造果冻般的通透流动感。

## 3. 重构 `style.css` 中的 `html[data-glass]` 视觉体系
- 严格将所有玻璃拟态样式收敛在 `html[data-glass]` 选择器作用域下。
- 注入 Apple 拟态材质规范：1px 顶部内高光微边（`inset 0 1px 1px rgba(255,255,255,0.85)`）、次级微漫射阴影、`backdrop-filter: blur(20px) saturate(180%)`。
- 修复 Sticky 表格固定列横向滑动重影穿透问题（设置 92% 高不透明度玻璃）。
- 修复表单 Input 全透明线框化缺陷（半透明立体底色 + Focus 外发光环）。
- 恢复成功/警告/危险/标签的原生红绿黄蓝语义色彩。
- 合并并清理 `style.css` 中重复的全局覆盖块。

## 4. 验证与构建
- 运行前端单元测试（`pnpm test:run`）与构建检查（`pnpm build`）。
- 验证后台主题切换（`classic` 100% 还原上游官方界面，`glass` 呈现纯正 Apple 液态玻璃）。