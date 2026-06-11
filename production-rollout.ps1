[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$RemoteHost,
    [int]$RemotePort = 22,
    [string]$RemoteUser = "root",
    [string]$RemoteUploadDir = "/opt/sub2api-rollout",
    [ValidateSet("full", "test", "promote")]
    [string]$Mode = "full",
    [string]$BuildScript = "build-linux.bat",
    [string]$RemoteScriptSource = "deploy/remote-production-flow.sh",
    [string]$RemoteScriptName = "remote-production-flow.sh",
    [string]$BinaryName = "sub2api-linux-amd64",
    [string]$TestService = "sub2api-test.service",
    [string]$ProductionService = "sub2api.service",
    [string]$TestBinaryPath = "/opt/sub2api-test/sub2api",
    [string]$ProductionBinaryPath = "/opt/sub2api/sub2api",
    [string]$TestConfigPath = "/opt/sub2api-test/data/config.yaml",
    [string]$ProductionConfigPath = "/app/data/config.yaml",
    [int]$TestPort = 18808,
    [int]$ProductionPort = 8808,
    [switch]$SkipBuild,
    [switch]$KeepTestService
)

$ErrorActionPreference = "Stop"

function Write-Step {
    param([string]$Message)

    Write-Host ""
    Write-Host ("==> {0}" -f $Message) -ForegroundColor Cyan
}

function Resolve-CommandPath {
    param(
        [string[]]$Candidates,
        [string]$Label
    )

    foreach ($candidate in $Candidates) {
        if ([string]::IsNullOrWhiteSpace($candidate)) {
            continue
        }

        if (Test-Path $candidate) {
            return (Resolve-Path $candidate).Path
        }

        try {
            $command = Get-Command $candidate -ErrorAction Stop
            if ($command.Source) {
                return $command.Source
            }

            if ($command.Path) {
                return $command.Path
            }
        } catch {
        }
    }

    throw ("Missing required command: {0}" -f $Label)
}

function Invoke-Checked {
    param(
        [string]$FilePath,
        [string[]]$Arguments,
        [string]$Label = $FilePath
    )

    & $FilePath @Arguments
    $exitCode = $LASTEXITCODE

    if (-not $?) {
        throw ("{0} failed" -f $Label)
    }

    if (($null -ne $exitCode) -and ($exitCode -ne 0)) {
        throw ("{0} failed with exit code {1}" -f $Label, $exitCode)
    }
}

function Invoke-LocalBuildScript {
    param([string]$ScriptPath)

    $extension = [System.IO.Path]::GetExtension($ScriptPath).ToLowerInvariant()

    if ($extension -eq ".sh") {
        $bashPath = Resolve-CommandPath -Candidates @(
            "bash",
            "C:\Program Files\Git\bin\bash.exe",
            "C:\Program Files\Git\usr\bin\bash.exe"
        ) -Label "bash"
        Invoke-Checked -FilePath $bashPath -Arguments @($ScriptPath) -Label ("bash {0}" -f $ScriptPath)
        return
    }

    if (($extension -eq ".bat") -or ($extension -eq ".cmd")) {
        $cmdPath = Resolve-CommandPath -Candidates @(
            "cmd",
            "C:\Windows\System32\cmd.exe"
        ) -Label "cmd"
        Invoke-Checked -FilePath $cmdPath -Arguments @("/c", $ScriptPath) -Label ("cmd /c {0}" -f $ScriptPath)
        return
    }

    Invoke-Checked -FilePath $ScriptPath -Arguments @() -Label $ScriptPath
}

function Quote-Bash {
    param([string]$Value)

    $singleQuote = [string][char]39
    $replacement = [string]::Concat([char]39, [char]34, [char]39, [char]34, [char]39)
    $escapedValue = $Value.Replace($singleQuote, $replacement)
    return ([string]::Format("{0}{1}{0}", [char]39, $escapedValue))
}

function Join-RemotePath {
    param(
        [string]$Base,
        [string]$Child
    )

    if ($Base.EndsWith("/")) {
        return ("{0}{1}" -f $Base, $Child)
    }

    return ("{0}/{1}" -f $Base, $Child)
}

$repoRoot = $PSScriptRoot
$uploadDir = Join-Path $repoRoot "dist\upload"
$binaryPath = Join-Path $uploadDir $BinaryName
$buildScriptPath = if ([System.IO.Path]::IsPathRooted($BuildScript)) { $BuildScript } else { Join-Path $repoRoot $BuildScript }
$remoteScriptLocalPath = if ([System.IO.Path]::IsPathRooted($RemoteScriptSource)) { $RemoteScriptSource } else { Join-Path $repoRoot $RemoteScriptSource }

if (-not (Test-Path $remoteScriptLocalPath)) {
    throw ("Missing remote flow script: {0}" -f $remoteScriptLocalPath)
}

$sshPath = Resolve-CommandPath -Candidates @(
    "ssh",
    "C:\Windows\System32\OpenSSH\ssh.exe",
    "C:\Program Files\Git\usr\bin\ssh.exe"
) -Label "ssh"
$scpPath = Resolve-CommandPath -Candidates @(
    "scp",
    "C:\Windows\System32\OpenSSH\scp.exe",
    "C:\Program Files\Git\usr\bin\scp.exe"
) -Label "scp"

if (-not $SkipBuild) {
    if (-not (Test-Path $buildScriptPath)) {
        throw ("Missing build script: {0}" -f $buildScriptPath)
    }

    Write-Step "Build local Linux amd64 release package"
    Invoke-LocalBuildScript -ScriptPath $buildScriptPath
}

if (-not (Test-Path $binaryPath)) {
    throw ("Missing binary artifact: {0}" -f $binaryPath)
}

$packagePath = $null
$packageCandidates = @(
    (Join-Path $uploadDir ("{0}.tar.zst" -f $BinaryName)),
    (Join-Path $uploadDir ("{0}.tar.gz" -f $BinaryName))
)

foreach ($candidate in $packageCandidates) {
    if (Test-Path $candidate) {
        $packagePath = $candidate
        break
    }
}

if (-not $packagePath) {
    throw ("Missing package artifact under {0}" -f $uploadDir)
}

$binaryHash = (Get-FileHash -Algorithm SHA256 $binaryPath).Hash.ToLowerInvariant()
$packageHash = (Get-FileHash -Algorithm SHA256 $packagePath).Hash.ToLowerInvariant()
$remotePackagePath = Join-RemotePath -Base $RemoteUploadDir -Child (Split-Path $packagePath -Leaf)
$remoteScriptPath = Join-RemotePath -Base $RemoteUploadDir -Child $RemoteScriptName
$remoteTarget = ("{0}@{1}" -f $RemoteUser, $RemoteHost)

Write-Step "Prepare remote staging directory"
Invoke-Checked -FilePath $sshPath -Arguments @(
    "-p",
    ([string]$RemotePort),
    $remoteTarget,
    ("mkdir -p {0}" -f (Quote-Bash $RemoteUploadDir))
) -Label "ssh mkdir"

Write-Step "Upload release package and remote flow script"
Invoke-Checked -FilePath $scpPath -Arguments @(
    "-P",
    ([string]$RemotePort),
    $packagePath,
    $remoteScriptLocalPath,
    ("{0}:{1}/" -f $remoteTarget, $RemoteUploadDir)
) -Label "scp upload"

$keepTestServiceValue = if ($KeepTestService) { "1" } else { "0" }
$envAssignments = @(
    ("MODE={0}" -f (Quote-Bash $Mode)),
    ("STAGE_DIR={0}" -f (Quote-Bash $RemoteUploadDir)),
    ("PACKAGE_PATH={0}" -f (Quote-Bash $remotePackagePath)),
    ("EXPECTED_BINARY_SHA256={0}" -f (Quote-Bash $binaryHash)),
    ("EXPECTED_PACKAGE_SHA256={0}" -f (Quote-Bash $packageHash)),
    ("BINARY_NAME={0}" -f (Quote-Bash $BinaryName)),
    ("TEST_SERVICE={0}" -f (Quote-Bash $TestService)),
    ("PRODUCTION_SERVICE={0}" -f (Quote-Bash $ProductionService)),
    ("TEST_BINARY_PATH={0}" -f (Quote-Bash $TestBinaryPath)),
    ("PRODUCTION_BINARY_PATH={0}" -f (Quote-Bash $ProductionBinaryPath)),
    ("TEST_CONFIG_PATH={0}" -f (Quote-Bash $TestConfigPath)),
    ("PRODUCTION_CONFIG_PATH={0}" -f (Quote-Bash $ProductionConfigPath)),
    ("TEST_PORT={0}" -f (Quote-Bash ([string]$TestPort))),
    ("PRODUCTION_PORT={0}" -f (Quote-Bash ([string]$ProductionPort))),
    ("KEEP_TEST_SERVICE={0}" -f (Quote-Bash $keepTestServiceValue))
)
$remoteCommandParts = @(
    ("chmod +x {0}" -f (Quote-Bash $remoteScriptPath)),
    ("env {0} bash {1} {2}" -f ($envAssignments -join " "), (Quote-Bash $remoteScriptPath), (Quote-Bash $Mode))
)
$remoteCommand = $remoteCommandParts -join " && "

Write-Step ("Run remote production flow ({0})" -f $Mode)
Invoke-Checked -FilePath $sshPath -Arguments @(
    "-p",
    ([string]$RemotePort),
    $remoteTarget,
    $remoteCommand
) -Label "ssh rollout"

Write-Step "Done"
Write-Host ("Remote package: {0}" -f $remotePackagePath)
Write-Host ("Binary SHA256: {0}" -f $binaryHash)
Write-Host ("Package SHA256: {0}" -f $packageHash)
