$ErrorActionPreference = 'Stop'

Add-Type -AssemblyName System.Drawing

$root = Split-Path -Parent $PSScriptRoot
$fontFamily = 'Microsoft YaHei UI'

function New-Canvas {
  param([string]$Title, [string]$Subtitle)
  $bitmap = [System.Drawing.Bitmap]::new(1440, 900)
  $graphics = [System.Drawing.Graphics]::FromImage($bitmap)
  $graphics.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::AntiAlias
  $graphics.TextRenderingHint = [System.Drawing.Text.TextRenderingHint]::ClearTypeGridFit
  $graphics.Clear([System.Drawing.ColorTranslator]::FromHtml('#F3F5F7'))

  $graphics.FillRectangle([System.Drawing.SolidBrush]::new([System.Drawing.ColorTranslator]::FromHtml('#FFFFFF')), 0, 0, 1440, 84)
  $graphics.DrawLine([System.Drawing.Pen]::new([System.Drawing.ColorTranslator]::FromHtml('#D9DEE5'), 2), 0, 84, 1440, 84)
  $graphics.DrawString($Title, [System.Drawing.Font]::new($fontFamily, 24, [System.Drawing.FontStyle]::Bold), [System.Drawing.Brushes]::Black, 44, 20)
  $graphics.DrawString($Subtitle, [System.Drawing.Font]::new($fontFamily, 11), [System.Drawing.SolidBrush]::new([System.Drawing.ColorTranslator]::FromHtml('#667085')), 46, 54)

  return @{ Bitmap = $bitmap; Graphics = $graphics }
}

function Draw-Panel {
  param($Graphics, [int]$X, [int]$Y, [int]$Width, [int]$Height)
  $Graphics.FillRectangle([System.Drawing.SolidBrush]::new([System.Drawing.Color]::White), $X, $Y, $Width, $Height)
  $Graphics.DrawRectangle([System.Drawing.Pen]::new([System.Drawing.ColorTranslator]::FromHtml('#D0D5DD'), 2), $X, $Y, $Width, $Height)
}

function Draw-Field {
  param($Graphics, [string]$Label, [string]$Value, [int]$X, [int]$Y, [int]$Width = 760)
  $Graphics.DrawString($Label, [System.Drawing.Font]::new($fontFamily, 12, [System.Drawing.FontStyle]::Bold), [System.Drawing.SolidBrush]::new([System.Drawing.ColorTranslator]::FromHtml('#344054')), $X, $Y)
  $Graphics.FillRectangle([System.Drawing.SolidBrush]::new([System.Drawing.ColorTranslator]::FromHtml('#FAFBFC')), $X, $Y + 28, $Width, 56)
  $Graphics.DrawRectangle([System.Drawing.Pen]::new([System.Drawing.ColorTranslator]::FromHtml('#C9D1DA'), 2), $X, $Y + 28, $Width, 56)
  $Graphics.DrawString($Value, [System.Drawing.Font]::new('Consolas', 12), [System.Drawing.SolidBrush]::new([System.Drawing.ColorTranslator]::FromHtml('#1D2939')), $X + 16, $Y + 46)
}

function Draw-Step {
  param($Graphics, [int]$Number, [int]$X, [int]$Y, [int]$Width, [int]$Height, [string]$Text)
  $red = [System.Drawing.ColorTranslator]::FromHtml('#D92D20')
  $Graphics.DrawRectangle([System.Drawing.Pen]::new($red, 4), $X, $Y, $Width, $Height)
  $Graphics.FillEllipse([System.Drawing.SolidBrush]::new($red), $X - 19, $Y - 19, 42, 42)
  $format = [System.Drawing.StringFormat]::new()
  $format.Alignment = [System.Drawing.StringAlignment]::Center
  $format.LineAlignment = [System.Drawing.StringAlignment]::Center
  $Graphics.DrawString([string]$Number, [System.Drawing.Font]::new($fontFamily, 14, [System.Drawing.FontStyle]::Bold), [System.Drawing.Brushes]::White, [System.Drawing.RectangleF]::new($X - 19, $Y - 19, 42, 42), $format)
  $Graphics.DrawString($Text, [System.Drawing.Font]::new($fontFamily, 11, [System.Drawing.FontStyle]::Bold), [System.Drawing.SolidBrush]::new($red), $X + $Width + 18, $Y + [Math]::Max(0, ($Height / 2) - 10))
}

function Draw-Button {
  param($Graphics, [string]$Text, [int]$X, [int]$Y, [int]$Width, [string]$Color = '#1570EF')
  $Graphics.FillRectangle([System.Drawing.SolidBrush]::new([System.Drawing.ColorTranslator]::FromHtml($Color)), $X, $Y, $Width, 52)
  $format = [System.Drawing.StringFormat]::new()
  $format.Alignment = [System.Drawing.StringAlignment]::Center
  $format.LineAlignment = [System.Drawing.StringAlignment]::Center
  $Graphics.DrawString($Text, [System.Drawing.Font]::new($fontFamily, 12, [System.Drawing.FontStyle]::Bold), [System.Drawing.Brushes]::White, [System.Drawing.RectangleF]::new($X, $Y, $Width, 52), $format)
}

function Save-Canvas {
  param($Canvas, [string]$RelativePath)
  $path = Join-Path $root $RelativePath
  $directory = Split-Path -Parent $path
  New-Item -ItemType Directory -Force -Path $directory | Out-Null
  $Canvas.Bitmap.Save($path, [System.Drawing.Imaging.ImageFormat]::Png)
  $Canvas.Graphics.Dispose()
  $Canvas.Bitmap.Dispose()
}

$canvas = New-Canvas '创建 API Key' '通用界面示意图 - 示例值均已脱敏'
$g = $canvas.Graphics
Draw-Panel $g 210 124 1020 700
Draw-Field $g '名称' 'sillytavern-home' 290 190 650
Draw-Field $g '分组' 'Grok - Standard' 290 305 650
Draw-Field $g '额度' '$5.00' 290 420 650
Draw-Field $g 'IP 规则' 'Leave empty for the first test' 290 535 650
Draw-Button $g '创建密钥' 790 690 150 '#1570EF'
Draw-Step $g 1 280 180 680 94 '填写用途名称'
Draw-Step $g 2 280 295 680 94 '选择分组和模型'
Draw-Step $g 3 280 410 680 94 '设置消费额度'
Draw-Step $g 4 280 525 680 94 '首次测试可留空'
Save-Canvas $canvas 'assets/common/api-key-create.png'

$canvas = New-Canvas 'Grok API 接入' '通用界面示意图 - 客户端会自动拼接接口路径'
$g = $canvas.Graphics
Draw-Panel $g 170 124 1100 700
Draw-Field $g 'Base URL' 'https://api.example.com/v1' 250 185 720
Draw-Field $g 'API Key' 'sk-demo-replace-me' 250 315 720
Draw-Field $g 'Model' 'select an ID returned by /models' 250 445 720
Draw-Button $g '测试请求' 790 635 180 '#087A5B'
Draw-Step $g 1 240 175 750 94 '复制本站 api_base_url'
Draw-Step $g 2 240 305 750 94 '使用专用 Key'
Draw-Step $g 3 240 435 750 94 '选择可见模型 ID'
Draw-Step $g 4 780 625 200 72 '返回 OK 即成功'
Save-Canvas $canvas 'assets/grok/grok-api-fields.png'

$canvas = New-Canvas 'Grok CLI 配置' 'config.toml 通用示意图 - 仅写入用户配置'
$g = $canvas.Graphics
Draw-Panel $g 140 120 1160 710
$g.FillRectangle([System.Drawing.SolidBrush]::new([System.Drawing.ColorTranslator]::FromHtml('#111827')), 190, 170, 800, 600)
$lines = @('[model."site-grok"]','model = "<grok-model-id>"','base_url = "https://api.example.com/v1"','name = "Site Grok"','env_key = "XAI_API_KEY"','api_backend = "responses"','','[models]','default = "site-grok"')
for ($i = 0; $i -lt $lines.Count; $i++) { $g.DrawString($lines[$i], [System.Drawing.Font]::new('Consolas', 14), [System.Drawing.SolidBrush]::new([System.Drawing.ColorTranslator]::FromHtml('#E5E7EB')), 220, 205 + ($i * 50)) }
Draw-Step $g 1 205 290 770 48 '填写完整 Base URL'
Draw-Step $g 2 205 390 770 48 'Key 从环境变量读取'
Draw-Step $g 3 205 440 770 48 '保持 Responses 协议'
Save-Canvas $canvas 'assets/grok/grok-cli-config.png'

$canvas = New-Canvas 'Codex 自定义 Provider' 'config.toml 通用示意图 - 使用 Responses'
$g = $canvas.Graphics
Draw-Panel $g 140 120 1160 710
$g.FillRectangle([System.Drawing.SolidBrush]::new([System.Drawing.ColorTranslator]::FromHtml('#111827')), 190, 170, 800, 600)
$lines = @('model_provider = "site-grok"','model = "<grok-model-id>"','','[model_providers.site-grok]','base_url = "https://api.example.com/v1"','env_key = "SUB2API_API_KEY"','wire_api = "responses"','requires_openai_auth = false','supports_websockets = false')
for ($i = 0; $i -lt $lines.Count; $i++) { $g.DrawString($lines[$i], [System.Drawing.Font]::new('Consolas', 13), [System.Drawing.SolidBrush]::new([System.Drawing.ColorTranslator]::FromHtml('#E5E7EB')), 220, 195 + ($i * 56)) }
Draw-Step $g 1 205 405 770 48 '复制本站 api_base_url'
Draw-Step $g 2 205 461 770 48 '从环境变量读取 Key'
Draw-Step $g 3 205 517 770 48 '保持 Responses 协议'
Save-Canvas $canvas 'assets/grok/codex-provider.png'

$canvas = New-Canvas 'Claude Code 网关变量' '终端通用示意图 - 使用 Messages 协议'
$g = $canvas.Graphics
Draw-Panel $g 140 120 1160 710
$g.FillRectangle([System.Drawing.SolidBrush]::new([System.Drawing.ColorTranslator]::FromHtml('#111827')), 190, 170, 850, 500)
$lines = @('$env:ANTHROPIC_BASE_URL="https://api.example.com"','$env:ANTHROPIC_AUTH_TOKEN="sk-demo-replace-me"','$env:ANTHROPIC_MODEL="<grok-model-id>"','$env:ANTHROPIC_DEFAULT_SONNET_MODEL="<grok-model-id>"','$env:ANTHROPIC_DEFAULT_HAIKU_MODEL="<grok-model-id>"','claude --model <grok-model-id> -p "Reply only OK"')
for ($i = 0; $i -lt $lines.Count; $i++) { $g.DrawString($lines[$i], [System.Drawing.Font]::new('Consolas', 12), [System.Drawing.SolidBrush]::new([System.Drawing.ColorTranslator]::FromHtml('#E5E7EB')), 220, 205 + ($i * 68)) }
Draw-Step $g 1 205 190 820 58 '使用不带 /v1 的站点根地址'
Draw-Step $g 2 205 258 820 58 '使用专用 Bearer Token'
Draw-Step $g 3 205 326 820 58 '映射可见模型 ID'
Draw-Step $g 4 205 530 820 58 '发送最小测试'
Save-Canvas $canvas 'assets/grok/claude-code-env.png'

$canvas = New-Canvas 'OpenCode Provider 配置' 'opencode.json 通用示意图 - 使用 Responses'
$g = $canvas.Graphics
Draw-Panel $g 140 120 1160 710
$g.FillRectangle([System.Drawing.SolidBrush]::new([System.Drawing.ColorTranslator]::FromHtml('#111827')), 190, 155, 850, 620)
$lines = @('{','  "provider": {','    "site": {','      "npm": "@ai-sdk/openai",','      "options": {','        "baseURL": "https://api.example.com/v1",','        "apiKey": "{env:SUB2API_API_KEY}"','      },','      "models": { "<grok-model-id>": {} }','    }','  }','}')
for ($i = 0; $i -lt $lines.Count; $i++) { $g.DrawString($lines[$i], [System.Drawing.Font]::new('Consolas', 12), [System.Drawing.SolidBrush]::new([System.Drawing.ColorTranslator]::FromHtml('#E5E7EB')), 220, 180 + ($i * 45)) }
Draw-Step $g 1 205 315 820 48 '选择 Responses Provider 包'
Draw-Step $g 2 205 405 820 93 '填写 Base URL 和环境变量 Key'
Draw-Step $g 3 205 540 820 48 '填写精确模型 ID'
Save-Canvas $canvas 'assets/grok/opencode-provider.png'

$canvas = New-Canvas 'SillyTavern API 连接' 'SillyTavern 1.18 通用连接界面'
$g = $canvas.Graphics
Draw-Panel $g 150 112 1140 730
Draw-Field $g 'API Type' 'Chat Completion' 225 165 760
Draw-Field $g 'Chat Completion Source' 'Custom (OpenAI-compatible)' 225 270 760
Draw-Field $g 'Custom Endpoint (Base URL)' 'https://api.example.com/v1' 225 375 760
Draw-Field $g 'Custom API Key' 'sk-demo-replace-me' 225 480 760
Draw-Field $g 'Enter a Model ID' 'select an ID returned by /models' 225 585 760
Draw-Button $g 'Connect' 1025 627 120 '#087A5B'
Draw-Button $g 'Test Message' 1025 692 160 '#1570EF'
Draw-Step $g 1 215 155 790 92 '选择 Chat Completion'
Draw-Step $g 2 215 260 790 92 '选择 Custom 来源'
Draw-Step $g 3 215 365 790 92 '不要追加 /chat/completions'
Draw-Step $g 4 215 470 790 92 '填写专用 Key'
Draw-Step $g 5 215 575 790 92 '填写精确模型 ID'
Save-Canvas $canvas 'assets/sillytavern/custom-openai.png'

$canvas = New-Canvas 'TavernAI 兼容客户端' '通用字段对照图 - 不同版本的字段名称可能不同'
$g = $canvas.Graphics
Draw-Panel $g 170 118 1100 720
Draw-Field $g 'Provider / Backend' 'Custom OpenAI-compatible or Reverse Proxy' 245 175 730
Draw-Field $g 'Base URL / Proxy URL' 'https://api.example.com/v1' 245 295 730
Draw-Field $g 'API Key / Token / Password' 'sk-demo-replace-me' 245 415 730
Draw-Field $g 'Model / Custom model' 'select an ID returned by /models' 245 535 730
Draw-Button $g 'Test' 805 695 170 '#6941C6'
Draw-Step $g 1 235 165 760 94 '选择自定义 Provider'
Draw-Step $g 2 235 285 760 94 '复制本站 Base URL'
Draw-Step $g 3 235 405 760 94 '使用专用 Key'
Draw-Step $g 4 235 525 760 94 '填写精确模型 ID'
Draw-Step $g 5 795 685 190 72 '发送最小消息'
Save-Canvas $canvas 'assets/tavernai/field-map.png'

Write-Output '已生成中文标注的脱敏文档示意图。'
