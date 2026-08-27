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
  $accent = [System.Drawing.ColorTranslator]::FromHtml('#0F766E')
  $Graphics.DrawRectangle([System.Drawing.Pen]::new($accent, 2), $X, $Y, $Width, $Height)
  $Graphics.FillEllipse([System.Drawing.SolidBrush]::new($accent), $X - 19, $Y - 19, 42, 42)
  $format = [System.Drawing.StringFormat]::new()
  $format.Alignment = [System.Drawing.StringAlignment]::Center
  $format.LineAlignment = [System.Drawing.StringAlignment]::Center
  $Graphics.DrawString([string]$Number, [System.Drawing.Font]::new($fontFamily, 14, [System.Drawing.FontStyle]::Bold), [System.Drawing.Brushes]::White, [System.Drawing.RectangleF]::new($X - 19, $Y - 19, 42, 42), $format)
  $Graphics.DrawString($Text, [System.Drawing.Font]::new($fontFamily, 11, [System.Drawing.FontStyle]::Bold), [System.Drawing.SolidBrush]::new($accent), $X + $Width + 18, $Y + [Math]::Max(0, ($Height / 2) - 10))
}

function Draw-NumberedNote {
  param($Graphics, [int]$Number, [int]$X, [int]$Y, [int]$Width, [string]$Title, [string]$Description)
  $accent = [System.Drawing.ColorTranslator]::FromHtml('#0F766E')
  $background = [System.Drawing.ColorTranslator]::FromHtml('#F0FDFA')
  $border = [System.Drawing.ColorTranslator]::FromHtml('#99F6E4')
  $text = [System.Drawing.ColorTranslator]::FromHtml('#134E4A')
  $Graphics.FillRectangle([System.Drawing.SolidBrush]::new($background), $X, $Y, $Width, 94)
  $Graphics.DrawRectangle([System.Drawing.Pen]::new($border, 1), $X, $Y, $Width, 94)
  $Graphics.FillEllipse([System.Drawing.SolidBrush]::new($accent), $X + 16, $Y + 18, 34, 34)
  $format = [System.Drawing.StringFormat]::new()
  $format.Alignment = [System.Drawing.StringAlignment]::Center
  $format.LineAlignment = [System.Drawing.StringAlignment]::Center
  $Graphics.DrawString([string]$Number, [System.Drawing.Font]::new($fontFamily, 12, [System.Drawing.FontStyle]::Bold), [System.Drawing.Brushes]::White, [System.Drawing.RectangleF]::new($X + 16, $Y + 18, 34, 34), $format)
  $Graphics.DrawString($Title, [System.Drawing.Font]::new($fontFamily, 12, [System.Drawing.FontStyle]::Bold), [System.Drawing.SolidBrush]::new($text), $X + 64, $Y + 18)
  $Graphics.DrawString($Description, [System.Drawing.Font]::new($fontFamily, 10), [System.Drawing.SolidBrush]::new($text), $X + 64, $Y + 49)
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

$canvas = New-Canvas '创建 API Key' 'DaiGua 脱敏配置参考图 - 示例值均已脱敏'
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

$canvas = New-Canvas 'Grok API 接入' 'DaiGua 脱敏配置参考图 - 客户端会自动拼接接口路径'
$g = $canvas.Graphics
Draw-Panel $g 170 124 1100 700
Draw-Field $g 'Base URL' 'https://api.daigua.icu/v1' 250 185 720
Draw-Field $g 'API Key' 'sk-demo-replace-me' 250 315 720
Draw-Field $g 'Model' 'select an ID returned by /models' 250 445 720
Draw-Button $g '测试请求' 790 635 180 '#087A5B'
Draw-Step $g 1 240 175 750 94 '复制本站 api_base_url'
Draw-Step $g 2 240 305 750 94 '使用专用 Key'
Draw-Step $g 3 240 435 750 94 '选择可见模型 ID'
Draw-Step $g 4 780 625 200 72 '返回 OK 即成功'
Save-Canvas $canvas 'assets/grok/grok-api-fields.png'

$canvas = New-Canvas 'Grok CLI 配置' 'DaiGua 脱敏配置参考图 - 仅写入用户配置'
$g = $canvas.Graphics
Draw-Panel $g 140 120 1160 710
$g.FillRectangle([System.Drawing.SolidBrush]::new([System.Drawing.ColorTranslator]::FromHtml('#111827')), 190, 170, 800, 600)
$lines = @('[model."site-grok"]','model = "<grok-model-id>"','base_url = "https://api.daigua.icu/v1"','name = "Site Grok"','env_key = "XAI_API_KEY"','api_backend = "responses"','','[models]','default = "site-grok"')
for ($i = 0; $i -lt $lines.Count; $i++) { $g.DrawString($lines[$i], [System.Drawing.Font]::new('Consolas', 14), [System.Drawing.SolidBrush]::new([System.Drawing.ColorTranslator]::FromHtml('#E5E7EB')), 220, 205 + ($i * 50)) }
Draw-Step $g 1 205 290 770 48 '填写完整 Base URL'
Draw-Step $g 2 205 390 770 48 'Key 从环境变量读取'
Draw-Step $g 3 205 440 770 48 '保持 Responses 协议'
Save-Canvas $canvas 'assets/grok/grok-cli-config.png'

$canvas = New-Canvas 'Codex 自定义 Responses Provider' '本机核验 Codex CLI 0.147.0 - 字段值已脱敏'
$g = $canvas.Graphics
Draw-Panel $g 140 120 1160 710
$g.FillRectangle([System.Drawing.SolidBrush]::new([System.Drawing.ColorTranslator]::FromHtml('#F8FAFC')), 190, 170, 760, 590)
$g.FillRectangle([System.Drawing.SolidBrush]::new([System.Drawing.ColorTranslator]::FromHtml('#2563EB')), 190, 170, 8, 590)
$g.DrawString('config.toml', [System.Drawing.Font]::new($fontFamily, 12, [System.Drawing.FontStyle]::Bold), [System.Drawing.SolidBrush]::new([System.Drawing.ColorTranslator]::FromHtml('#1D4ED8')), 224, 196)
$lines = @('model_provider = "daigua"','model = "<model-id>"','','[model_providers.daigua]','base_url = "https://api.daigua.icu/v1"','env_key = "SUB2API_API_KEY"','wire_api = "responses"','requires_openai_auth = false','supports_websockets = false')
for ($i = 0; $i -lt $lines.Count; $i++) { $g.DrawString($lines[$i], [System.Drawing.Font]::new('Consolas', 12), [System.Drawing.SolidBrush]::new([System.Drawing.ColorTranslator]::FromHtml('#172B4D')), 224, 240 + ($i * 52)) }
Draw-NumberedNote $g 1 990 185 260 'Base URL' '使用本站完整 /v1 地址'
Draw-NumberedNote $g 2 990 300 260 'Responses' 'wire_api 保持 responses'
Draw-NumberedNote $g 3 990 415 260 'Key 来源' '只保存环境变量名称'
Draw-NumberedNote $g 4 990 530 260 'WebSocket' '兼容路由默认关闭'
Save-Canvas $canvas 'assets/grok/codex-provider.png'

$canvas = New-Canvas 'Claude Code 配置' 'DaiGua 脱敏配置示意图 - 使用 Messages 协议'
$g = $canvas.Graphics
Draw-Panel $g 140 120 1160 710
$g.FillRectangle([System.Drawing.SolidBrush]::new([System.Drawing.ColorTranslator]::FromHtml('#F8FAFC')), 190, 170, 760, 590)
$g.FillRectangle([System.Drawing.SolidBrush]::new([System.Drawing.ColorTranslator]::FromHtml('#0F766E')), 190, 170, 8, 590)
$g.DrawString('PowerShell', [System.Drawing.Font]::new($fontFamily, 12, [System.Drawing.FontStyle]::Bold), [System.Drawing.SolidBrush]::new([System.Drawing.ColorTranslator]::FromHtml('#0F766E')), 224, 198)
$lines = @('$env:ANTHROPIC_BASE_URL="https://api.daigua.icu"','$env:ANTHROPIC_API_KEY="sk-demo-replace-me"','$env:ANTHROPIC_MODEL="<model-id>"','$env:ANTHROPIC_DEFAULT_SONNET_MODEL="<model-id>"','$env:ANTHROPIC_DEFAULT_HAIKU_MODEL="<model-id>"','claude --model <model-id> -p "Reply only OK"')
for ($i = 0; $i -lt $lines.Count; $i++) { $g.DrawString($lines[$i], [System.Drawing.Font]::new('Consolas', 12), [System.Drawing.SolidBrush]::new([System.Drawing.ColorTranslator]::FromHtml('#172B4D')), 224, 252 + ($i * 68)) }
Draw-NumberedNote $g 1 990 185 260 '站点根地址' '这里不要带 /v1'
Draw-NumberedNote $g 2 990 300 260 '优先 API Key' '不要同时设置 Token'
Draw-NumberedNote $g 3 990 415 260 '模型映射' '填写 /models 返回的 ID'
Draw-NumberedNote $g 4 990 530 260 '最小测试' '先确认返回 OK'
Save-Canvas $canvas 'assets/grok/claude-code-env.png'

$canvas = New-Canvas 'OpenCode Provider 配置' 'DaiGua 脱敏配置示意图 - 仅用于 Chat Completions'
$g = $canvas.Graphics
Draw-Panel $g 140 120 1160 710
$g.FillRectangle([System.Drawing.SolidBrush]::new([System.Drawing.ColorTranslator]::FromHtml('#F8FAFC')), 190, 170, 760, 590)
$g.FillRectangle([System.Drawing.SolidBrush]::new([System.Drawing.ColorTranslator]::FromHtml('#2563EB')), 190, 170, 8, 590)
$g.DrawString('opencode.json', [System.Drawing.Font]::new($fontFamily, 12, [System.Drawing.FontStyle]::Bold), [System.Drawing.SolidBrush]::new([System.Drawing.ColorTranslator]::FromHtml('#1D4ED8')), 224, 198)
$lines = @('{','  "provider": {','    "daigua": {','      "npm": "@ai-sdk/openai-compatible",','      "options": {','        "baseURL": "https://api.daigua.icu/v1",','        "apiKey": "{env:SUB2API_API_KEY}"','      },','      "models": { "<model-id>": {} }','    }','  }','}')
for ($i = 0; $i -lt $lines.Count; $i++) { $g.DrawString($lines[$i], [System.Drawing.Font]::new('Consolas', 11), [System.Drawing.SolidBrush]::new([System.Drawing.ColorTranslator]::FromHtml('#172B4D')), 224, 244 + ($i * 39)) }
Draw-NumberedNote $g 1 990 190 260 '兼容 Provider' '选择 openai-compatible'
Draw-NumberedNote $g 2 990 305 260 '完整 Base URL' '使用本站 /v1 地址'
Draw-NumberedNote $g 3 990 420 260 '环境变量 Key' '不要写入配置文件'
Draw-NumberedNote $g 4 990 535 260 '模型 ID' '先通过 Chat 最小测试'
Save-Canvas $canvas 'assets/grok/opencode-provider.png'

$canvas = New-Canvas 'OpenClaw 自定义 Provider' '官方字段参考 - DaiGua 地址与示例值已脱敏'
$g = $canvas.Graphics
Draw-Panel $g 150 116 1140 720
Draw-Field $g 'Provider ID' 'daigua' 230 165 720
Draw-Field $g 'Base URL' 'https://api.daigua.icu/v1' 230 280 720
Draw-Field $g 'API' 'openai-completions' 230 395 720
Draw-Field $g 'Model catalog' '<model-id>  /  DaiGua Model' 230 510 720
Draw-Button $g 'Test connection' 740 675 210 '#087A5B'
Draw-NumberedNote $g 1 990 170 250 '添加 Provider' '名称建议使用 daigua'
Draw-NumberedNote $g 2 990 285 250 '填写 Base URL' '地址以 /v1 结尾'
Draw-NumberedNote $g 3 990 400 250 '选择协议' '首次使用 Chat Completions'
Draw-NumberedNote $g 4 990 515 250 '登记模型' '必须写入模型列表'
Save-Canvas $canvas 'assets/clients/openclaw-provider.png'

$canvas = New-Canvas 'Hermes Agent Custom Provider' '官方向导字段参考 - 运行 hermes model'
$g = $canvas.Graphics
Draw-Panel $g 170 120 1100 710
Draw-Field $g 'Provider type' 'Custom / OpenAI-compatible' 245 180 720
Draw-Field $g 'Provider name' 'daigua' 245 295 720
Draw-Field $g 'Base URL' 'https://api.daigua.icu/v1' 245 410 720
Draw-Field $g 'API Key' 'sk-demo-replace-me' 245 525 720
Draw-Field $g 'Model' '<model-id>' 245 640 720
Draw-Step $g 1 235 170 750 94 '选择 Custom Provider'
Draw-Step $g 2 235 285 750 94 '设置识别名称'
Draw-Step $g 3 235 400 750 94 '复制本站 /v1 地址'
Draw-Step $g 4 235 515 750 94 '使用客户端专用 Key'
Draw-Step $g 5 235 630 750 94 '填写 /models 返回 ID'
Save-Canvas $canvas 'assets/clients/hermes-custom-provider.png'

$canvas = New-Canvas 'WorkBuddy 自定义 API' '官方模型设置字段参考 - 默认自动补全 Chat 端点'
$g = $canvas.Graphics
Draw-Panel $g 150 116 1140 720
Draw-Field $g '供应商' '自定义 API' 230 165 720
Draw-Field $g 'URL' 'https://api.daigua.icu/v1' 230 280 720
Draw-Field $g 'API Key' 'sk-demo-replace-me' 230 395 720
Draw-Field $g '模型名' '<model-id>' 230 510 720
Draw-Field $g '自定义协议' '关闭 - 自动追加 /chat/completions' 230 625 720
Draw-NumberedNote $g 1 990 170 250 '选择自定义 API' '不要使用内置套餐入口'
Draw-NumberedNote $g 2 990 285 250 '填写 /v1 地址' '不要重复写完整端点'
Draw-NumberedNote $g 3 990 400 250 '专用 Key' '仅保存在本地配置'
Draw-NumberedNote $g 4 990 515 250 '精确模型 ID' '来自当前 Key 的 /models'
Draw-NumberedNote $g 5 990 630 250 '保持关闭' '让客户端自动补全路径'
Save-Canvas $canvas 'assets/clients/workbuddy-custom-api.png'

$canvas = New-Canvas 'ZCode 自定义供应商' '官方模型设置字段参考 - 选择 OpenAI-compatible'
$g = $canvas.Graphics
Draw-Panel $g 160 116 1120 720
Draw-Field $g '供应商名称' 'DaiGua' 240 175 710
Draw-Field $g '协议' 'OpenAI-compatible' 240 295 710
Draw-Field $g 'API 基础 URL' 'https://api.daigua.icu/v1' 240 415 710
Draw-Field $g 'API Key' 'sk-demo-replace-me' 240 535 710
Draw-Field $g '模型 ID' '<model-id>' 240 655 710
Draw-Step $g 1 230 165 740 94 '添加自定义供应商'
Draw-Step $g 2 230 285 740 94 '选择 OpenAI 协议'
Draw-Step $g 3 230 405 740 94 '复制本站 /v1 地址'
Draw-Step $g 4 230 525 740 94 '填写客户端专用 Key'
Draw-Step $g 5 230 645 740 94 '手动登记模型 ID'
Save-Canvas $canvas 'assets/clients/zcode-custom-provider.png'

$canvas = New-Canvas 'OpenAI SDK 与 curl' 'DaiGua 最小 Responses 请求 - 示例值均已脱敏'
$g = $canvas.Graphics
Draw-Panel $g 140 120 1160 710
$g.FillRectangle([System.Drawing.SolidBrush]::new([System.Drawing.ColorTranslator]::FromHtml('#F8FAFC')), 190, 170, 800, 590)
$g.FillRectangle([System.Drawing.SolidBrush]::new([System.Drawing.ColorTranslator]::FromHtml('#0F766E')), 190, 170, 8, 590)
$g.DrawString('curl', [System.Drawing.Font]::new($fontFamily, 12, [System.Drawing.FontStyle]::Bold), [System.Drawing.SolidBrush]::new([System.Drawing.ColorTranslator]::FromHtml('#0F766E')), 224, 198)
$lines = @('curl "https://api.daigua.icu/v1/responses" \','  -H "Authorization: Bearer sk-demo-replace-me" \','  -H "Content-Type: application/json" \','  -d ''{','    "model": "<model-id>",','    "input": "Reply only OK"','  }''')
for ($i = 0; $i -lt $lines.Count; $i++) { $g.DrawString($lines[$i], [System.Drawing.Font]::new('Consolas', 12), [System.Drawing.SolidBrush]::new([System.Drawing.ColorTranslator]::FromHtml('#172B4D')), 224, 250 + ($i * 60)) }
Draw-NumberedNote $g 1 1020 185 230 '完整端点' 'Responses 使用 /v1/responses'
Draw-NumberedNote $g 2 1020 300 230 '示例 Key' '提交前只保留 sk-demo'
Draw-NumberedNote $g 3 1020 415 230 '动态模型' '复制 /models 的精确 ID'
Draw-NumberedNote $g 4 1020 530 230 '最小输入' '先验证纯文本返回 OK'
Save-Canvas $canvas 'assets/sdk/openai-curl.png'

$canvas = New-Canvas 'SillyTavern API 连接' 'DaiGua 脱敏字段布局参考图 - SillyTavern 1.18'
$g = $canvas.Graphics
Draw-Panel $g 150 112 1140 730
Draw-Field $g 'API Type' 'Chat Completion' 225 165 760
Draw-Field $g 'Chat Completion Source' 'Custom (OpenAI-compatible)' 225 270 760
Draw-Field $g 'Custom Endpoint (Base URL)' 'https://api.daigua.icu/v1' 225 375 760
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

$canvas = New-Canvas 'TavernAI 兼容客户端' 'DaiGua 脱敏字段布局参考图 - 不同版本名称可能不同'
$g = $canvas.Graphics
Draw-Panel $g 170 118 1100 720
Draw-Field $g 'Provider / Backend' 'Custom OpenAI-compatible or Reverse Proxy' 245 175 730
Draw-Field $g 'Base URL / Proxy URL' 'https://api.daigua.icu/v1' 245 295 730
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
