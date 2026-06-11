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
    Write-Host "==> $Message" -ForegroundColor Cyan
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

    throw "Missing required command: $Label"
}

function Invoke-Checked {
    param(
        [string]$FilePath,
        [string[]]$Arguments,
        [string]$Label = $FilePath
    )

    & $FilePath @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "$Label failed with exit code $LASTEXITCODE"
    }
}

function Invoke-LocalBuildScript {
    param([string]$ScriptPath)

    $extension = [System.IO.Path]::GetExtension($ScriptPath).ToLowerInvariant()

    switch ($extension) {
        ".sh" {
            $bashPath = Resolve-CommandPath -Candidates @(
                "bash",
                "C:\Program Files\Git\bin\bash.exe",
                "C:\Program Files\Git\usr\bin\bash.exe"
            ) -Label "bash"
            Invoke-Checked -FilePath $bashPath -Arguments @($ScriptPath) -Label "bash $ScriptPath"
            return
        }
        ".bat" {
            $cmdPath = Resolve-CommandPath -Candidates @(
                "cmd",
                "C:\Windows\System32\cmd.exe"
            ) -Label "cmd"
            Invoke-Checked -FilePath $cmdPath -Arguments @("/c", $ScriptPath) -Label "cmd /c $ScriptPath"
            return
        }
        ".cmd" {
            $cmdPath = Resolve-CommandPath -Candidates @(
                "cmd",
                "C:\Windows\System32\cmd.exe"
            ) -Label "cmd"
            Invoke-Checked -FilePath $cmdPath -Arguments @("/c", $ScriptPath) -Label "cmd /c $ScriptPath"
            return
        }
        default {
            Invoke-Checked -FilePath $ScriptPath -Arguments @() -Label $ScriptPath
            return
        }
    }
}

function Quote-Bash {
    param([string]$Value)

    return "'" + $Value.Replace("'", "'""'""'") + "'"
}

function Join-RemotePath {
    param(
        [string]$Base,
        [string]$Child
    )

    if ($Base.EndsWith("/")) {
        return "$Base$Child"
    }

    return "$Base/$Child"
}

$repoRoot = $PSScriptRoot
$uploadDir = Join-Path $repoRoot "dist\upload"
$binaryPath = Join-Path $uploadDir $BinaryName
$buildScriptPath = if ([System.IO.Path]::IsPathRooted($BuildScript)) { $BuildScript } else { Join-Path $repoRoot $BuildScript }
$remoteScriptLocalPath = if ([System.IO.Path]::IsPathRooted($RemoteScriptSource)) { $RemoteScriptSource } else { Join-Path $repoRoot $RemoteScriptSource }

if (-not (Test-Path $remoteScriptLocalPath)) {
    throw "Missing remote flow script: $remoteScriptLocalPath"
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
        throw "Missing build script: $buildScriptPath"
    }

    Write-Step "本地编译 Linux amd64 发布包"
    Invoke-LocalBuildScript -ScriptPath $buildScriptPath
}

if (-not (Test-Path $binaryPath)) {
    throw "Missing binary artifact: $binaryPath"
}

$packagePath = @(
    (Join-Path $uploadDir "$BinaryName.tar.zst")
    (Join-Path $uploadDir "$BinaryName.tar.gz")
) | Where-Object { Test-Path $_ } | Select-Object -First 1

if (-not $packagePath) {
    throw "Missing package artifact under $uploadDir"
}

$binaryHash = (Get-FileHash -Algorithm SHA256 $binaryPath).Hash.ToLowerInvariant()
$packageHash = (Get-FileHash -Algorithm SHA256 $packagePath).Hash.ToLowerInvariant()
$remotePackagePath = Join-RemotePath -Base $RemoteUploadDir -Child (Split-Path $packagePath -Leaf)
$remoteScriptPath = Join-RemotePath -Base $RemoteUploadDir -Child $RemoteScriptName
$remoteTarget = "$RemoteUser@$RemoteHost"

Write-Step "准备远端目录"
Invoke-Checked -FilePath $sshPath -Arguments @(
    "-p",
    "$RemotePort",
    $remoteTarget,
    "mkdir -p $(Quote-Bash $RemoteUploadDir)"
) -Label "ssh mkdir"

Write-Step "上传压缩包和远端流程脚本"
Invoke-Checked -FilePath $scpPath -Arguments @(
    "-P",
    "$RemotePort",
    $packagePath,
    $remoteScriptLocalPath,
    "${remoteTarget}:$RemoteUploadDir/"
) -Label "scp upload"

$keepTestServiceValue = if ($KeepTestService) { "1" } else { "0" }
$envAssignments = @(
    "MODE=$(Quote-Bash $Mode)",
    "STAGE_DIR=$(Quote-Bash $RemoteUploadDir)",
    "PACKAGE_PATH=$(Quote-Bash $remotePackagePath)",
    "EXPECTED_BINARY_SHA256=$(Quote-Bash $binaryHash)",
    "EXPECTED_PACKAGE_SHA256=$(Quote-Bash $packageHash)",
    "BINARY_NAME=$(Quote-Bash $BinaryName)",
    "TEST_SERVICE=$(Quote-Bash $TestService)",
    "PRODUCTION_SERVICE=$(Quote-Bash $ProductionService)",
    "TEST_BINARY_PATH=$(Quote-Bash $TestBinaryPath)",
    "PRODUCTION_BINARY_PATH=$(Quote-Bash $ProductionBinaryPath)",
    "TEST_CONFIG_PATH=$(Quote-Bash $TestConfigPath)",
    "PRODUCTION_CONFIG_PATH=$(Quote-Bash $ProductionConfigPath)",
    "TEST_PORT=$(Quote-Bash $TestPort.ToString())",
    "PRODUCTION_PORT=$(Quote-Bash $ProductionPort.ToString())",
    "KEEP_TEST_SERVICE=$(Quote-Bash $keepTestServiceValue)"
)
$remoteCommand = 'chmod +x {0} && env {1} bash {0} {2}' -f `
    (Quote-Bash $remoteScriptPath), `
    ($envAssignments -join ' '), `
    (Quote-Bash $Mode)

Write-Step "执行远端生产流程 ($Mode)"
Invoke-Checked -FilePath $sshPath -Arguments @(
    "-p",
    "$RemotePort",
    $remoteTarget,
    $remoteCommand
) -Label "ssh rollout"

Write-Step "完成"
Write-Host "Remote package: $remotePackagePath"
Write-Host "Binary SHA256: $binaryHash"
Write-Host "Package SHA256: $packageHash"
