[CmdletBinding()]
param(
    # IncrementalOverlay audits the reviewed product checkpoint. HistoricalAudit
    # keeps the original product-freeze inventory. OverlayBoundary compares a
    # product commit to one reviewed, immutable upstream commit.
    [ValidateSet('IncrementalOverlay', 'HistoricalAudit', 'OverlayBoundary')]
    [string]$Mode = 'IncrementalOverlay',
    # Optional for non-upstream modes. When omitted, the mode-specific immutable
    # baseline is selected below; do not pass a floating origin/main ref here.
    [string]$BaselineRef,
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

# These are review checkpoints, not moving branch names. Keep the historical
# inventory separate from the current migration gate so identity/admin debt
# inherited before the latest reviewed production checkpoint is not reported as
# a new Overlay regression.
$script:HistoricalBaselineRef = 'baseline/v0.1.164-before-overlay'
$script:IncrementalBaselineRef = '54dcc7eacebe5075077ce6ab5bfed18321e05c67'

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

function Test-VerificationPath {
    param([string]$Path)

    return $Path -match '(_test\.go$|\.(spec|test)\.[cm]?[jt]sx?$)'
}

function Get-OverlayBucket {
    param(
        [string]$Path,
        [string]$ChangeKind
    )

    if ($script:PublishedImmutableMigrationPaths.Contains($Path)) {
        if ($ChangeKind -eq 'A') { return 'published-immutable-migration' }
        return 'immutable-migration-change'
    }
    if (Test-VerificationPath -Path $Path) { return 'verification-test' }
    if ($Path -like 'backend/internal/custom/*') { return 'custom-backend' }
    if ($Path -like 'frontend/src/custom/*' -or $Path -like 'frontend/public/custom/*') { return 'custom-frontend' }
    # go.sum is a Go toolchain lock file. It may change when Ent/Wire generation
    # resolves its own command dependencies, without adding product behavior.
    if ($Path -eq 'backend/go.sum') { return 'build-metadata' }
    if ($Path -like 'backend/ent/*') { return 'ent-generated-or-schema' }
    if ($Path -like 'backend/migrations/*_custom_*.sql' -or $Path -like 'backend/migrations/*_custom_*_test.go') { return 'custom-migration' }
    if ($Path -like 'docs/*') { return 'documentation' }
    if ($script:FixedMountAllowlist.Contains($Path)) { return 'fixed-mount' }
    if ($script:CompositionRootAllowlist.Contains($Path)) { return 'composition-root' }
    if ($script:RestrictedPlatformPortAllowlist.Contains($Path)) { return 'restricted-platform-port' }
    if ($Path -like 'tools/check-custom-overlay-boundaries.*') { return 'boundary-gate' }
    if ($Path -eq 'tools/check-upstream-sync.ps1') { return 'boundary-gate' }
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

function Test-ApprovedCleanupTransformation {
    param(
        [string[]]$Arguments,
        [string]$Path
    )

    # c27 removes the no-longer-used handler argument from the common route
    # registrar. Git necessarily represents the shortened declaration as one
    # added line. This is an exact cleanup proof, not a path allowlist: any
    # second added line or a different signature remains a shared violation.
    $patch = Invoke-Git ($Arguments + @('--no-color', '--unified=0', '--', $Path))
    $addedLines = @($patch | Where-Object { $_.StartsWith('+') -and -not $_.StartsWith('+++') })
    if ($Path -eq 'backend/internal/server/routes/common.go') {
        return $addedLines.Count -eq 1 -and $addedLines[0] -eq '+func RegisterCommonRoutes(r *gin.Engine) {'
    }

    # Removing activity-only redeem metadata leaves one syntactic replacement
    # at the end of the fluent update chain. The exact patch shape prevents a
    # broad repository allowlist from masking later feature additions.
    if ($Path -eq 'backend/internal/repository/redeem_code_repo.go') {
        return $addedLines.Count -eq 1 -and $addedLines[0] -eq '+		SetValidityDays(code.ValidityDays)'
    }

    if ($Path -eq 'backend/internal/service/redeem_code.go') {
        return $addedLines.Count -eq 1 -and $addedLines[0] -eq '+import "time"'
    }

    return $false
}

function Test-ExactHistoricalUpstreamRevert {
    param(
        [string]$Path,
        [bool]$AgainstWorktree
    )

    # This proof applies only to the currently reviewed incremental baseline.
    # It accepts a shared path only when its final content is byte-for-byte the
    # same as that immutable upstream commit, so it cannot mask new logic.
    if ($Mode -ne 'IncrementalOverlay') {
        return $false
    }

    if ($AgainstWorktree) {
        & $script:GitExecutable diff --quiet $script:HistoricalUpstreamCommit -- $Path
    } else {
        & $script:GitExecutable diff --quiet $script:HistoricalUpstreamCommit $script:TargetCommit -- $Path
    }

    if ($LASTEXITCODE -eq 0) {
        return $true
    }
    if ($LASTEXITCODE -eq 1) {
        return $false
    }
    throw "Cannot compare $Path with approved historical upstream $($script:HistoricalUpstreamCommit)."
}

function Test-ExactReviewedUpstreamHotfix {
    param(
        [string]$Path,
        [bool]$AgainstWorktree
    )

    # A reviewed shared change is accepted only for its immutable review base
    # and one exact final blob per path. Any later edit returns to shared.
    $reviewedBlobs = $null
    if ($Mode -eq 'OverlayBoundary' -and $script:BaselineCommit -eq $script:ReviewedUpstreamHotfixBase) {
        $reviewedBlobs = $script:ReviewedUpstreamHotfixBlobs
    } elseif ($Mode -eq 'IncrementalOverlay' -and $script:BaselineCommit -eq $script:ReviewedIncrementalHotfixBase) {
        $reviewedBlobs = $script:ReviewedIncrementalHotfixBlobs
    }
    if ($null -eq $reviewedBlobs -or -not $reviewedBlobs.ContainsKey($Path)) {
        return $false
    }

    if ($AgainstWorktree) {
        if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) {
            return $false
        }
        $actualBlob = [string](Invoke-Git @('hash-object', '--', $Path))
    } else {
        $actualBlob = [string](Invoke-Git @('rev-parse', "$($script:TargetCommit):$Path"))
    }

    return $actualBlob -eq $reviewedBlobs[$Path]
}

function Get-ChangedRows {
    param(
        [string]$Scope,
        [string[]]$DiffArguments
    )

    $changes = Invoke-Git ($DiffArguments + @('--name-status', '--diff-filter=ACMRD', '--find-renames'))
    foreach ($change in $changes | Where-Object { $_ }) {
        $parts = $change -split "`t"
        $changeKind = $parts[0]
        if ($parts.Count -lt 2) {
            throw "Unexpected git name-status row: $change"
        }

        # A rename away from a published migration must fail just like an edit.
        # The destination remains the audited row for all ordinary renames.
        if ($changeKind -match '^R' -and $parts.Count -ge 3 -and $script:PublishedImmutableMigrationPaths.Contains($parts[1])) {
            [pscustomobject]@{
                Scope  = $Scope
                Bucket = 'immutable-migration-change'
                Path   = $parts[1]
            }
        }

        $path = if ($changeKind -match '^[RC]') { $parts[$parts.Count - 1] } else { $parts[1] }
        $bucket = Get-OverlayBucket -Path $path -ChangeKind $changeKind
        if ($bucket -eq 'shared') {
            if (Test-ExactHistoricalUpstreamRevert -Path $path -AgainstWorktree ($Scope -eq 'worktree-tracked')) {
                $bucket = 'historical-upstream-revert'
            } elseif ($changeKind -eq 'D') {
                $bucket = 'shared-cleanup'
            } elseif (Test-ExactReviewedUpstreamHotfix -Path $path -AgainstWorktree ($Scope -eq 'worktree-tracked')) {
                $bucket = 'reviewed-upstream-hotfix'
            } else {
                $addedLines = Get-AddedLineCount -Arguments ($DiffArguments + @('--numstat', '--find-renames')) -Path $path
                if (($null -ne $addedLines -and $addedLines -eq 0) -or (Test-ApprovedCleanupTransformation -Arguments $DiffArguments -Path $path)) {
                    $bucket = 'shared-cleanup'
                }
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
            Bucket = Get-OverlayBucket -Path $path -ChangeKind 'A'
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

    $violations = @($Rows | Where-Object { $_.Bucket -in @('shared', 'immutable-migration-change') })
    if ($violations.Count -eq 0) {
        Write-Host "${Scope}: boundary classification passed."
        return @()
    }

    Write-Host "${Scope}: boundary classification failed. Disallowed paths:"
    $violations | Select-Object -ExpandProperty Path | Sort-Object -Unique | ForEach-Object {
        Write-Host "  $_"
    }
    return $violations
}

$script:GitExecutable = Resolve-GitPath -RequestedPath $GitPath
$script:HistoricalUpstreamCommit = '43d4bae2464387817560a1aeb0b023cd0c9b22ee'
$script:ReviewedUpstreamHotfixBase = '396a9d1130c9a8ab977e6a959a4fdd2d9f95dd27'
$script:ReviewedUpstreamHotfixBlobs = @{
    'backend/internal/repository/custom_group_usage_rollup_repo.go' = '1342478ab68e32dac4d1f8708a11decd6f31267a'
    'backend/internal/repository/dashboard_aggregation_repo.go' = '5266aef6f6fa4346619e882feb116d1284757105'
}
$script:ReviewedIncrementalHotfixBase = '54dcc7eacebe5075077ce6ab5bfed18321e05c67'
$script:ReviewedIncrementalHotfixBlobs = @{
    # Smart-routing shared request lifecycle adapters. Provider policy,
    # persistence, retention, strategy presets, and UI normalization remain
    # owned by the backend/frontend custom smart-routing modules.
    'backend/internal/handler/failover_loop.go' = '42471ec7cda1386e08af7f79a27121bbede17098'
    'backend/internal/handler/openai_chat_completions.go' = 'b1437942c9357e79c09f2adec51b18b3dc57b14e'
    'backend/internal/handler/openai_gateway_handler.go' = '6e1f23ef421a332ae0d959957b177ae72c9218ea'
    'backend/internal/handler/smart_routing_retry.go' = '3dd4df3442ba2078301f378cc136ff0627b0d5eb'
    'backend/internal/server/middleware/api_key_auth.go' = '0c73c0f1cfaa0ebd84f9316630d792ed96060feb'
    'backend/internal/service/openai_account_scheduler.go' = '4bcc4ca5815fc76f83a00adef8dfb31db907b299'
    'backend/internal/service/smart_routing_context.go' = '010f517a1cad31e6ece6d20b73bb04ab8ffc8292'
    'backend/internal/service/smart_routing_feedback.go' = 'f7b969131a37851576b12027a78cf32fe158cdd2'
    # Reviewed generic UI fixes and existing presentation text. These hashes
    # prevent the exception from becoming a reusable shared-path allowlist.
    'frontend/src/components/common/DateRangePicker.vue' = '78c70c3e69f5af4c9defe1fcc25f3c0f7a06d401'
    'frontend/src/components/user/dashboard/UserDashboardCharts.vue' = '6a662cfd8e65555f37005ba6694433a41af07864'
    'frontend/src/features/channel-monitor-v2/monitorFormat.ts' = 'a1100bfb789775840144342f5899536792a9f894'
    'frontend/src/i18n/locales/en/dashboard.ts' = '521da623a750db18fa73eb841a48401b89c29c03'
    'frontend/src/i18n/locales/zh/dashboard.ts' = '0299b31aae3e8ccc663550509f36f351b115c7ce'
    'frontend/src/views/admin/DashboardView.vue' = 'df8c62fc28b9c1482a3e3cc7588008a9abc41a91'
    'frontend/src/views/admin/UsageView.vue' = 'cb28e2add30d4789d3670748581d25ab1edcd32c'
    'frontend/src/views/user/UsageView.vue' = 'b5dda9378f7796d8a2676e33a3fec3a6309b2ff3'
}
$script:FixedMountAllowlist = [System.Collections.Generic.HashSet[string]]::new([string[]]@(
    'backend/internal/server/http.go',
    'backend/internal/server/router.go',
    'backend/internal/handler/settingsext/mount.go',
    'backend/internal/handler/setting_handler.go',
    'backend/internal/handler/admin/setting_handler.go',
    'backend/internal/handler/admin/setting_handler_update.go',
    'backend/internal/handler/dto/settings.go',
    'frontend/src/components/layout/AppHeader.vue',
    'frontend/src/components/layout/AppSidebar.vue',
    'frontend/src/i18n/locales/en/admin/index.ts',
    'frontend/src/i18n/locales/en/index.ts',
    'frontend/src/i18n/locales/zh/admin/index.ts',
    'frontend/src/i18n/locales/zh/index.ts',
    'frontend/src/router/index.ts',
    'frontend/src/router/meta.d.ts',
    'frontend/src/stores/app.ts',
    'frontend/src/api/admin/settings.ts',
    'frontend/src/types/index.ts',
    'frontend/src/utils/featureFlags.ts',
    'frontend/src/views/admin/SettingsView.vue',
    # Channel monitor V2 presentation mounts. These files may render generic
    # registered settings/source components and carry generic extension data;
    # provider-specific probe rules remain under frontend/src/custom.
    'frontend/src/api/channelMonitorV2.ts',
    'frontend/src/features/channel-monitor-v2/MonitorSettingsPanel.vue',
    'frontend/src/views/admin/ChannelMonitorView.vue',
    'frontend/src/views/user/ChannelStatusV2View.vue',
    # Presentation-only Overlay adapters. They delegate display classification
    # or formatting to frontend/src/custom and must not contain custom rules.
    'frontend/src/views/admin/UsersView.vue',
    'frontend/src/components/admin/user/UserBalanceHistoryModal.vue',
    'frontend/src/views/user/AffiliateView.vue',
    'frontend/src/views/user/RedeemView.vue'
))
$script:CompositionRootAllowlist = [System.Collections.Generic.HashSet[string]]::new([string[]]@(
    'backend/cmd/server/wire.go',
    'backend/cmd/server/wire_gen.go',
    'backend/internal/handler/wire.go',
    'backend/internal/service/wire.go'
))
$script:RestrictedPlatformPortAllowlist = [System.Collections.Generic.HashSet[string]]::new([string[]]@(
    'backend/internal/service/setting_service.go',
    'backend/internal/service/admin_service.go',
    'backend/internal/service/admin_user.go',
    'backend/internal/service/code_generator.go',
    'backend/internal/service/redeem_service.go',
    # Generic channel-monitor scheduling, observation, and V2 result ports.
    # Concrete active-probe settings, persistence, and merge policy remain in
    # backend/internal/custom/modules/channel-monitor-v2-active-probes.
    'backend/internal/service/channel_monitor_runner.go',
    'backend/internal/service/channel_monitor_service.go',
    'backend/internal/service/channel_monitor_v2.go'
))
$script:PublishedImmutableMigrationPaths = [System.Collections.Generic.HashSet[string]]::new([string[]]@(
    'backend/migrations/173_port_balance_features.sql',
    'backend/migrations/174_add_entry_feature_switches.sql',
    'backend/migrations/175_add_game_hall_dedicated_tables.sql',
    'backend/migrations/176_backfill_game_hall_dedicated_balances.sql',
    'backend/migrations/177_migrate_legacy_checkin_records.sql',
    'backend/migrations/178_add_game_hall_rounds.sql',
    'backend/migrations/179_repair_legacy_checkin_migration.sql',
    'backend/migrations/180_add_user_game_hall_disabled.sql',
    'backend/migrations/181_create_reward_deliveries.sql',
    'backend/migrations/185_widen_large_balance_amount_columns.sql',
    'backend/migrations/187_move_game_hall_user_access.sql',
    # Historical repair migration. It is keyed by full filename in
    # schema_migrations and must stay byte-for-byte immutable after release.
    'backend/migrations/192_restore_ops_monitoring_schema.sql'
))

# rev-parse produces exactly one line. Do not index the scalar result: in
# PowerShell that would select the first character of the SHA.
$targetCommit = [string](Invoke-Git @('rev-parse', '--verify', "$TargetRef^{commit}"))
$script:TargetCommit = $targetCommit
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
    if ([string]::IsNullOrWhiteSpace($BaselineRef)) {
        $BaselineRef = if ($Mode -eq 'HistoricalAudit') { $script:HistoricalBaselineRef } else { $script:IncrementalBaselineRef }
    }
    $baselineCommit = [string](Invoke-Git @('rev-parse', '--verify', "$BaselineRef^{commit}"))
    $auditBaseLabel = "reviewed $Mode baseline $baselineCommit"
}
$script:BaselineCommit = $baselineCommit

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
    Write-Host 'Move business code into custom/, restore immutable migrations, or obtain a reviewed mount, composition-root, or platform-port allowlist change.'
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
