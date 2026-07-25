[CmdletBinding()]
param(
    # HistoricalAudit keeps the original product-freeze audit. OverlayBoundary
    # compares a product commit to one reviewed, immutable upstream commit.
    [ValidateSet('HistoricalAudit', 'OverlayBoundary')]
    [string]$Mode = 'HistoricalAudit',
    # This is the named Overlay freeze point, not origin/main. Pass a later
    # reviewed checkpoint explicitly when auditing an incremental migration.
    [string]$BaselineRef = 'baseline/v0.1.164-before-overlay',
    # Required by OverlayBoundary. Pass a resolved commit SHA, never a floating
    # remote-tracking branch such as origin/main.
    [string]$UpstreamRef,
    [string]$TargetRef = 'HEAD',
    # Include staged, unstaged, and untracked files relative to TargetRef.
    # This is required before accepting an in-progress migration.
    [switch]$IncludeWorktree,
    # Optional machine-readable result for a sync workflow or CI comparison.
    [string]$ReportPath,
    [string]$GitPath
)

$ErrorActionPreference = 'Stop'

function Resolve-GitPath {
    param([string]$RequestedPath)

    if ($RequestedPath) {
        if (-not (Test-Path -LiteralPath $RequestedPath -PathType Leaf)) {
            throw "Git executable does not exist: $RequestedPath"
        }
        return (Resolve-Path -LiteralPath $RequestedPath).Path
    }

    $command = Get-Command git -ErrorAction SilentlyContinue
    if ($command) {
        return $command.Source
    }

    $bundledGit = 'C:\Users\L\.cache\codex-runtimes\codex-primary-runtime\dependencies\native\git\cmd\git.exe'
    if (Test-Path -LiteralPath $bundledGit -PathType Leaf) {
        return $bundledGit
    }

    throw 'Git executable was not found. Pass -GitPath with an absolute git.exe path.'
}

function Invoke-Git {
    param([string[]]$Arguments)

    $output = & $script:GitExecutable @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "git $($Arguments -join ' ') failed with exit code $LASTEXITCODE"
    }
    return @($output)
}

function Test-GitAncestor {
    param(
        [string]$Ancestor,
        [string]$Descendant
    )

    & $script:GitExecutable merge-base --is-ancestor $Ancestor $Descendant
    if ($LASTEXITCODE -eq 0) {
        return $true
    }
    if ($LASTEXITCODE -eq 1) {
        return $false
    }
    throw "git merge-base --is-ancestor failed with exit code $LASTEXITCODE"
}

function Get-OverlayBucket {
    param([string]$Path)

    if ($Path -like 'backend/internal/custom/*') { return 'custom-backend' }
    if ($Path -like 'frontend/src/custom/*' -or $Path -like 'frontend/public/custom/*') { return 'custom-frontend' }
    if ($Path -like 'backend/ent/*') { return 'ent-generated-or-schema' }
    if ($Path -like 'backend/migrations/*_custom_*.sql' -or $Path -like 'backend/migrations/*_custom_*_test.go') { return 'custom-migration' }
    if ($script:PublishedHistoryAllowlist.Contains($Path)) { return 'published-history' }
    if ($Path -like 'docs/*') { return 'documentation' }
    if ($script:IntegrationAllowlist.Contains($Path)) { return 'fixed-integration' }
    if ($script:CleanupAllowlist.Contains($Path)) { return 'shared-cleanup' }
    if ($Path -like 'tools/check-custom-overlay-boundaries.*') { return 'boundary-gate' }
    return 'shared'
}

function Get-AddedLineCount {
    param(
        [string[]]$Arguments,
        [string]$Path
    )

    $numstat = Invoke-Git ($Arguments + @('--', $Path))
    $added = 0
    foreach ($line in $numstat) {
        if ($line -match '^(?<added>\d+)\t') {
            $added += [int]$Matches.added
            continue
        }

        # Binary changes or an unexpected numstat format must be reviewed, not grandfathered.
        return $null
    }
    return $added
}

function Get-ChangedRows {
    param(
        [string]$Scope,
        [string[]]$DiffArguments
    )

    $paths = Invoke-Git ($DiffArguments + @('--name-only', '--diff-filter=ACMR', '--find-renames'))
    foreach ($path in $paths | Where-Object { $_ }) {
        $bucket = Get-OverlayBucket -Path $path
        if ($bucket -eq 'shared') {
            $addedLines = Get-AddedLineCount -Arguments ($DiffArguments + @('--numstat', '--find-renames')) -Path $path
            if ($null -ne $addedLines -and $addedLines -eq 0) {
                $bucket = 'shared-cleanup'
            }
        }

        [pscustomobject]@{
            Scope  = $Scope
            Bucket = $bucket
            Path   = $path
        }
    }
}

function Get-UntrackedWorktreeRows {
    $paths = Invoke-Git @('ls-files', '--others', '--exclude-standard')
    foreach ($path in $paths | Where-Object { $_ }) {
        # An untracked file is necessarily an addition. Do not classify it as
        # deletion-only cleanup merely because it has no numstat entry yet.
        [pscustomobject]@{
            Scope  = 'worktree-untracked'
            Bucket = Get-OverlayBucket -Path $path
            Path   = $path
        }
    }
}

function Write-AuditResult {
    param(
        [string]$Scope,
        [object[]]$Rows
    )

    if ($Rows.Count -eq 0) {
        Write-Host "${Scope}: no added or modified paths to classify."
        return @()
    }

    $Rows | Group-Object Bucket | Sort-Object Name | ForEach-Object {
        Write-Host ("{0}: {1}" -f $_.Name, $_.Count)
    }

    $violations = @($Rows | Where-Object { $_.Bucket -eq 'shared' })
    if ($violations.Count -eq 0) {
        Write-Host "${Scope}: boundary classification passed."
        return @()
    }

    Write-Host "${Scope}: boundary classification failed. Added shared paths:"
    $violations | Select-Object -ExpandProperty Path | Sort-Object -Unique | ForEach-Object {
        Write-Host "  $_"
    }
    return $violations
}

$script:GitExecutable = Resolve-GitPath -RequestedPath $GitPath
$script:IntegrationAllowlist = [System.Collections.Generic.HashSet[string]]::new([string[]]@(
    'backend/cmd/server/wire.go',
    'backend/cmd/server/wire_gen.go',
    'backend/cmd/server/wire_gen_test.go',
    'backend/internal/server/http.go',
    'backend/internal/server/router.go',
    'backend/internal/server/routes/common.go',
    'backend/internal/server/routes/user.go',
    'backend/internal/web/embed_test.go',
    'backend/internal/handler/settingsext/mount.go',
    'backend/internal/handler/settingsext/mount_test.go',
    'backend/internal/handler/setting_handler.go',
    'backend/internal/handler/admin/setting_handler.go',
    'backend/internal/handler/admin/setting_handler_update.go',
    'backend/internal/handler/admin/setting_handler_custom_settings_test.go',
    'backend/internal/handler/dto/settings.go',
    'backend/internal/handler/wire.go',
    'backend/internal/handler/gateway_handler.go',
    'backend/internal/handler/gateway_handler_usage_settings_test.go',
    'backend/internal/service/setting_service.go',
    'frontend/src/components/layout/AppHeader.vue',
    'frontend/src/components/layout/AppSidebar.vue',
    'frontend/src/components/layout/__tests__/AppSidebar.spec.ts',
    'frontend/src/i18n/__tests__/localesNoKeyCollision.spec.ts',
    'frontend/src/i18n/locales/en/admin/index.ts',
    'frontend/src/i18n/locales/en/index.ts',
    'frontend/src/i18n/locales/zh/admin/index.ts',
    'frontend/src/i18n/locales/zh/index.ts',
    'frontend/src/router/__tests__/feature-access.spec.ts',
    'frontend/src/router/index.ts',
    'frontend/src/router/meta.d.ts',
    'frontend/src/stores/__tests__/app.spec.ts',
    'frontend/src/stores/app.ts',
    'frontend/src/api/admin/settings.ts',
    'frontend/src/types/index.ts',
    'frontend/src/utils/featureFlags.ts',
    'frontend/src/views/admin/SettingsView.vue'
))
$script:PublishedHistoryAllowlist = [System.Collections.Generic.HashSet[string]]::new([string[]]@(
    'backend/migrations/187_move_game_hall_user_access.sql',
    'backend/migrations/game_hall_migrations_regression_test.go'
))
$script:CleanupAllowlist = [System.Collections.Generic.HashSet[string]]::new([string[]]@(
    'backend/internal/handler/admin/user_handler.go',
    'backend/internal/handler/dto/types.go',
    'backend/internal/repository/user_repo.go',
    'backend/internal/service/admin_service.go',
    'backend/internal/service/code_format_settings.go',
    'backend/internal/service/code_format_settings_test.go',
    'backend/internal/service/setting_parse.go',
    'backend/internal/service/setting_public.go',
    'backend/internal/service/setting_service_backend_mode_test.go',
    'backend/internal/service/setting_service_platform_quota_test.go',
    'backend/internal/service/setting_service_update_test.go',
    'backend/internal/service/settings_view.go',
    'frontend/src/components/admin/user/UserEditModal.vue'
))

# rev-parse produces exactly one line. Do not index the scalar result: in
# PowerShell that would select the first character of the SHA.
$targetCommit = [string](Invoke-Git @('rev-parse', '--verify', "$TargetRef^{commit}"))
$auditBaseLabel = $BaselineRef
if ($Mode -eq 'OverlayBoundary') {
    if (-not $UpstreamRef) {
        throw 'OverlayBoundary requires -UpstreamRef with a reviewed upstream commit SHA.'
    }
    if ($UpstreamRef -match '(^|/)origin/main$') {
        throw 'OverlayBoundary does not accept a floating origin/main reference. Resolve and pass its commit SHA.'
    }
    $baselineCommit = [string](Invoke-Git @('rev-parse', '--verify', "$UpstreamRef^{commit}"))
    if (-not (Test-GitAncestor -Ancestor $baselineCommit -Descendant $targetCommit)) {
        throw "OverlayBoundary requires upstream $baselineCommit to be an ancestor of target $targetCommit."
    }
    $auditBaseLabel = "reviewed upstream $baselineCommit"
} else {
    $baselineCommit = [string](Invoke-Git @('rev-parse', '--verify', "$BaselineRef^{commit}"))
}

$range = "$baselineCommit..$targetCommit"
if ($baselineCommit -eq $targetCommit) {
    Write-Host "Committed audit: no commits after $auditBaseLabel. This is not a worktree audit."
    $committedRows = @()
    $committedViolations = @()
} else {
    $committedRows = @(Get-ChangedRows -Scope "committed $range" -DiffArguments @('diff', $range))
    $committedViolations = @(Write-AuditResult -Scope "Committed audit ($range)" -Rows $committedRows)
}

$worktreeViolations = @()
if ($IncludeWorktree) {
    $trackedWorktreeRows = @(Get-ChangedRows -Scope 'worktree-tracked' -DiffArguments @('diff', $targetCommit))
    $untrackedWorktreeRows = @(Get-UntrackedWorktreeRows)
    $worktreeRows = @($trackedWorktreeRows + $untrackedWorktreeRows)
    $worktreeViolations = @(Write-AuditResult -Scope "Worktree audit (against $TargetRef)" -Rows $worktreeRows)
} else {
    Write-Host 'Worktree audit skipped. Use -IncludeWorktree before accepting uncommitted migration work.'
}

if (($committedViolations.Count + $worktreeViolations.Count) -gt 0) {
    if ($ReportPath) {
        $report = [pscustomobject]@{
            Mode = $Mode; BaseCommit = $baselineCommit; TargetCommit = $targetCommit
            CommittedRows = @($committedRows); WorktreeRows = @($worktreeRows)
            Violations = @($committedViolations + $worktreeViolations)
        }
        $reportDirectory = Split-Path -Parent $ReportPath
        if ($reportDirectory) { New-Item -ItemType Directory -Force -Path $reportDirectory | Out-Null }
        $report | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath $ReportPath -Encoding UTF8
    }
    Write-Host 'Move business code into custom/, keep shared changes deletion-only for debt cleanup, or obtain a reviewed allowlist change.'
    exit 1
}

if ($ReportPath) {
    $report = [pscustomobject]@{
        Mode = $Mode; BaseCommit = $baselineCommit; TargetCommit = $targetCommit
        CommittedRows = @($committedRows); WorktreeRows = @($worktreeRows)
        Violations = @()
    }
    $reportDirectory = Split-Path -Parent $ReportPath
    if ($reportDirectory) { New-Item -ItemType Directory -Force -Path $reportDirectory | Out-Null }
    $report | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath $ReportPath -Encoding UTF8
}

Write-Host 'Overlay boundary gate completed without shared additions in the selected audit scope.'
