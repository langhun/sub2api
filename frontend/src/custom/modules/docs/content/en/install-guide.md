<!-- beginner -->
## Step 1: Prepare the terminal

Confirm the system clock is correct and install required runtimes. Gemini CLI currently requires Node.js 20+. Prefer the official Codex and Claude Code installers over old global npm instructions.

## Step 2: Install a client

| Client | Windows PowerShell | macOS / Linux |
| --- | --- | --- |
| Codex | `irm https://chatgpt.com/codex/install.ps1 | iex` | `curl -fsSL https://chatgpt.com/codex/install.sh | sh` |
| Claude Code | `irm https://claude.ai/install.ps1 | iex` or WinGet | `curl -fsSL https://claude.ai/install.sh | bash` |
| Gemini CLI | `npm install -g @google/gemini-cli` | npm, Homebrew, or an official package |
| Grok CLI | `irm https://x.ai/cli/install.ps1 | iex` | `curl -fsSL https://x.ai/cli/install.sh | bash` |

Install commands change over time. Recheck the official source linked by this topic before running one.

## Step 3: Verify executables

```text
codex --version
claude --version
gemini --version
grok --version
```

If a command is missing, reopen the terminal and check whether the install directory is on `PATH`. Do not run daily AI clients as an administrator.

## Step 4: Configure this site

Installation alone does not connect the client. Continue to the OpenAI, Anthropic, Gemini, or Grok CLI topic to configure the Base URL, environment variable, and model.

<!-- reference -->
## Configuration locations

The Windows home is normally `%USERPROFILE%`; macOS/Linux uses `$HOME`. Keep configuration in the user directory and out of repositories. Before uninstalling, remove user configuration and environment variables containing credentials.
