# 测试 Apple Glass 主题

## 如何启用主题

### 方法 1：通过导航栏切换器（推荐）

1. 启动开发服务器：
```bash
cd frontend
npm run dev
```

2. 打开浏览器访问 http://localhost:5173

3. 点击导航栏右上角的**主题按钮**（太阳/月亮图标）

4. 在下拉菜单中：
   - **主题风格** → 选择 **"Apple Glass"**
   - **外观模式** → 选择 **"浅色" / "深色" / "自动"**

5. 页面应该立即显示：
   - 🌊 **WebGL 流体动画背景**（会随鼠标移动）
   - 💎 **半透明玻璃卡片**（可以看到背后的流体）
   - ✨ **动态渐变效果**

### 方法 2：通过浏览器控制台（快速测试）

打开浏览器控制台（F12），输入：

```javascript
// 设置 Apple 主题
localStorage.setItem('theme-style', 'apple')

// 刷新页面
location.reload()
```

### 方法 3：通过管理后台（全局配置）

1. 以管理员身份登录
2. 进入 **管理后台** → **站点设置**
3. 找到 **"Apple Theme"** 设置项
4. 开启 **"启用 Apple 主题"**
5. 设置 **默认主题模式**
6. 保存设置

## 验证效果

### 检查 WebGL 流体是否工作

1. 打开浏览器控制台（F12）

2. 查看是否有以下日志：
```
[FluidCanvas] Initializing fluid shader...
[FluidShader] Attaching to canvas: <canvas>
[FluidShader] WebGL2 context created successfully
[FluidShader] Shaders compiled and linked successfully
```

3. 如果看到错误，检查：
   - 浏览器是否支持 WebGL2（Chrome 56+, Firefox 51+, Safari 15+）
   - 是否有硬件加速

### 检查玻璃效果

1. 页面上的卡片应该是**半透明**的
2. 通过卡片可以看到背后的**流体动画**
3. 卡片边缘有**毛玻璃模糊**效果

### 测试鼠标交互

1. 在页面上移动鼠标
2. 流体应该**跟随鼠标**产生涡流效果
3. 移动速度越快，效果越明显

## 调试问题

### 流体背景不显示

检查控制台是否有错误：

```javascript
// 检查主题设置
console.log('Theme Style:', localStorage.getItem('theme-style'))
console.log('Theme Mode:', localStorage.getItem('theme-mode'))

// 检查 HTML 属性
console.log('data-theme:', document.documentElement.getAttribute('data-theme'))
console.log('dark class:', document.documentElement.classList.contains('dark'))
```

### 玻璃效果不透明

检查浏览器是否支持 `backdrop-filter`：

```javascript
// 检查支持情况
console.log('backdrop-filter supported:', 
  CSS.supports('backdrop-filter', 'blur(10px)') ||
  CSS.supports('-webkit-backdrop-filter', 'blur(10px)')
)
```

### WebGL 不工作

```javascript
// 检查 WebGL2 支持
const canvas = document.createElement('canvas')
const gl = canvas.getContext('webgl2')
console.log('WebGL2 supported:', gl !== null)
```

## 预期效果对比

### Apple Glass 主题（浅色模式）
- 背景：淡蓝灰色（#F4F8FD）
- 流体：青绿色渐变流动
- 卡片：白色半透明玻璃
- 主色：Apple 蓝（#0071e3）

### Apple Glass 主题（深色模式）
- 背景：深蓝黑色（#0C121B）
- 流体：深蓝色渐变流动
- 卡片：深灰半透明玻璃
- 主色：亮蓝色（#2997ff）

## 性能说明

- 流体动画限制在 **30fps**
- 使用 **GPU 低功耗模式**
- 流体模拟分辨率为显示分辨率的 **1/4**
- 支持 **prefers-reduced-motion**（系统设置禁用动画时自动关闭）
- 触摸设备和 Windows 系统**默认禁用鼠标跟随**

## 截图对比

请拍摄启用 Apple Glass 主题后的截图，应该能看到：
1. 背景有流动的渐变效果
2. 卡片是半透明的
3. 鼠标移动时流体有响应
