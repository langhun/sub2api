<!-- beginner -->
## 步骤 1：准备终端

确认系统时间正确，并安装客户端要求的运行时。Gemini CLI 当前要求 Node.js 20+；Codex 与 Claude Code 优先使用官方安装器，避免旧教程中的全局 npm 方案。

## 步骤 2：安装客户端

| 客户端 | Windows PowerShell | macOS / Linux |
| --- | --- | --- |
| Codex | `irm https://chatgpt.com/codex/install.ps1 | iex` | `curl -fsSL https://chatgpt.com/codex/install.sh | sh` |
| Claude Code | `irm https://claude.ai/install.ps1 | iex` 或 WinGet | `curl -fsSL https://claude.ai/install.sh | bash` |
| Gemini CLI | `npm install -g @google/gemini-cli` | npm、Homebrew 或官方包 |
| Grok CLI | `irm https://x.ai/cli/install.ps1 | iex` | `curl -fsSL https://x.ai/cli/install.sh | bash` |

安装命令会随官方版本变化；执行前打开本文来源中的官方页面复核。

## 步骤 3：验证可执行文件

```text
codex --version
claude --version
gemini --version
grok --version
```

命令找不到时，关闭并重新打开终端，再检查安装目录是否进入 `PATH`。不要用管理员权限运行日常 AI 客户端。

## 步骤 4：配置本站

安装成功不代表已接入本站。继续阅读 OpenAI、Anthropic、Gemini 或 Grok CLI 指南，配置 Base URL、环境变量和模型。

<!-- reference -->
## 配置目录

Windows 用户目录通常是 `%USERPROFILE%`；macOS/Linux 是 `$HOME`。配置文件只放在用户目录，不要提交到项目仓库。卸载前先移除包含密钥的用户配置和环境变量。
