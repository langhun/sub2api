[CmdletBinding()]
param(
    [Parameter(Mandatory)]
    [string]$ProductBefore,
    [Parameter(Mandatory)]
    [string]$ProductAfter,
    [Parameter(Mandatory)]
    [string]$OldUpstream,
    [Parameter(Mandatory)]
    [string]$NewUpstream,
    [string]$PreReport,
    [string]$PostReport,
    [switch]$GenerateReports,
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
    if ($command) { return $command.Source }

    $bundledGit = 'C:\Users\L\.cache\codex-runtimes\codex-primary-runtime\dependencies\native\git\cmd\git.exe'
    if (Test-Path -LiteralPath $bundledGit -PathType Leaf) { return $bundledGit }
    throw 'Git executable was not found. Pass -GitPath with an absolute git.exe path.'
}

function Resolve-Commit {
    param([string]$Ref)

    $result = & $script:GitExecutable rev-parse --verify "$Ref^{commit}"
    if ($LASTEXITCODE -ne 0) { throw "Cannot resolve commit: $Ref" }
    return [string]$result
}

function Test-Ancestor {
    param([string]$Ancestor, [string]$Descendant)

    & $script:GitExecutable merge-base --is-ancestor $Ancestor $Descendant
    if ($LASTEXITCODE -eq 0) { return $true }
    if ($LASTEXITCODE -eq 1) { return $false }
    throw "git merge-base --is-ancestor failed with exit code $LASTEXITCODE"
}

function Read-BoundaryReport {
    param([string]$Path, [string]$ExpectedBase, [string]$ExpectedTarget)

    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) {
        throw "Boundary report does not exist: $Path"
    }
    $report = Get-Content -Raw -LiteralPath $Path | ConvertFrom-Json
    if ($report.Mode -ne 'OverlayBoundary') {
        throw "Boundary report must use OverlayBoundary mode: $Path"
    }
    if ($report.BaseCommit -ne $ExpectedBase -or $report.TargetCommit -ne $ExpectedTarget) {
        throw "Boundary report refs do not match the sync inputs: $Path"
    }
    return @($report.Violations | Where-Object { $_.Bucket -in @('shared', 'immutable-migration-change') } | ForEach-Object { $_.Path } | Sort-Object -Unique)
}

function New-BoundaryReport {
    param(
        [string]$UpstreamRef,
        [string]$TargetRef,
        [string]$ReportPath
    )

    $boundaryScript = Join-Path $PSScriptRoot 'check-custom-overlay-boundaries.ps1'
    if (-not (Test-Path -LiteralPath $boundaryScript -PathType Leaf)) {
        throw "Boundary checker does not exist: $boundaryScript"
    }

    $output = @(& $boundaryScript `
        -Mode OverlayBoundary `
        -UpstreamRef $UpstreamRef `
        -TargetRef $TargetRef `
        -ReportPath $ReportPath `
        -GitPath $script:GitExecutable *>&1)
    $exitCode = $LASTEXITCODE

    # Exit 1 is expected when the product has historical shared paths. The
    # pre/post comparator below decides whether this sync introduced new ones.
    if ($exitCode -notin @(0, 1) -or -not (Test-Path -LiteralPath $ReportPath -PathType Leaf)) {
        $details = ($output | Select-Object -Last 20) -join [Environment]::NewLine
        throw "Boundary report generation failed for $TargetRef (exit $exitCode).$([Environment]::NewLine)$details"
    }
}

$script:GitExecutable = Resolve-GitPath -RequestedPath $GitPath
$beforeCommit = Resolve-Commit -Ref $ProductBefore
$afterCommit = Resolve-Commit -Ref $ProductAfter
$oldUpstreamCommit = Resolve-Commit -Ref $OldUpstream
$newUpstreamCommit = Resolve-Commit -Ref $NewUpstream

if (-not $PreReport) {
    $PreReport = Join-Path (Get-Location) 'artifacts/overlay-pre.json'
}
if (-not $PostReport) {
    $PostReport = Join-Path (Get-Location) 'artifacts/overlay-post.json'
}

if ($GenerateReports) {
    New-BoundaryReport -UpstreamRef $oldUpstreamCommit -TargetRef $beforeCommit -ReportPath $PreReport
    New-BoundaryReport -UpstreamRef $newUpstreamCommit -TargetRef $afterCommit -ReportPath $PostReport
}

if (-not (Test-Ancestor -Ancestor $oldUpstreamCommit -Descendant $beforeCommit)) {
    throw "Old upstream $oldUpstreamCommit is not an ancestor of pre-sync product $beforeCommit."
}
if (-not (Test-Ancestor -Ancestor $newUpstreamCommit -Descendant $afterCommit)) {
    throw "New upstream $newUpstreamCommit is not an ancestor of post-sync product $afterCommit."
}

$preViolations = Read-BoundaryReport -Path $PreReport -ExpectedBase $oldUpstreamCommit -ExpectedTarget $beforeCommit
$postViolations = Read-BoundaryReport -Path $PostReport -ExpectedBase $newUpstreamCommit -ExpectedTarget $afterCommit
$newViolations = @($postViolations | Where-Object { $_ -notin $preViolations })

# A true merge must retain the new upstream as a direct parent, so accidental
# rebases or arbitrary branch parents cannot be presented as a reviewed sync.
$mergeCandidates = @(& $script:GitExecutable rev-list --parents "$newUpstreamCommit..$afterCommit")
if ($LASTEXITCODE -ne 0) { throw 'Cannot inspect post-sync merge ancestry.' }
$syncMerge = $mergeCandidates | Where-Object {
    $parts = $_ -split ' '
    $parts.Count -ge 3 -and $parts[1..($parts.Count - 1)] -contains $newUpstreamCommit
} | Select-Object -First 1
if (-not $syncMerge) {
    throw "No merge commit between $newUpstreamCommit and $afterCommit has the reviewed upstream as a direct parent."
}

$syncCommit = ($syncMerge -split ' ')[0]
$subject = [string](& $script:GitExecutable log -1 --format=%s $syncCommit)
if ($LASTEXITCODE -ne 0) { throw "Cannot inspect sync commit $syncCommit" }
if ($subject -notmatch '^chore\(sync\):') {
    throw "Sync merge $syncCommit must use a chore(sync): subject; found: $subject"
}

$upstreamBase = [string](& $script:GitExecutable merge-base $beforeCommit $newUpstreamCommit)
if ($LASTEXITCODE -ne 0) { throw 'Cannot calculate upstream merge base.' }
$upstreamPaths = @(& $script:GitExecutable diff --name-only "$upstreamBase..$newUpstreamCommit")
if ($LASTEXITCODE -ne 0) { throw 'Cannot list upstream changes.' }

Write-Host "Existing shared inventory before sync (informational): $($preViolations.Count)"
Write-Host "Existing shared inventory after sync (informational): $($postViolations.Count)"
Write-Host "New shared paths introduced by this sync: $($newViolations.Count)"
Write-Host "Reviewed sync merge: $syncCommit ($subject)"
Write-Host "Upstream-only changed paths: $($upstreamPaths.Count)"
if ($newViolations.Count -gt 0) {
    Write-Host 'Sync introduced new shared paths:'
    $newViolations | ForEach-Object { Write-Host "  $_" }
    exit 1
}

Write-Host 'Upstream sync gate passed: no new shared Overlay paths were introduced. Existing inventory is not a sync failure.'
