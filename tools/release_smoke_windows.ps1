param(
    [string]$Version = "",
    [switch]$SkipFrontendTests,
    [switch]$SkipPortableInstall
)

$ErrorActionPreference = "Stop"

function Write-Step {
    param([string]$Message)
    Write-Host ""
    Write-Host "==> $Message" -ForegroundColor Cyan
}

function Wait-Port {
    param(
        [int]$Port,
        [int]$TimeoutSec = 20
    )

    $deadline = (Get-Date).AddSeconds($TimeoutSec)
    while ((Get-Date) -lt $deadline) {
        try {
            $client = New-Object System.Net.Sockets.TcpClient
            $iar = $client.BeginConnect("127.0.0.1", $Port, $null, $null)
            if ($iar.AsyncWaitHandle.WaitOne(500) -and $client.Connected) {
                $client.EndConnect($iar)
                $client.Close()
                return $true
            }
            $client.Close()
        } catch {
        }
        Start-Sleep -Milliseconds 300
    }
    return $false
}

function Stop-ProcIfRunning {
    param($Proc)
    if ($null -ne $Proc -and -not $Proc.HasExited) {
        Stop-Process -Id $Proc.Id -Force
    }
}

$repoRoot = Split-Path -Parent $PSScriptRoot
$backendDir = Join-Path $repoRoot "backend"
$frontendDir = Join-Path $repoRoot "frontend"
$git = "C:\Program Files\Git\cmd\git.exe"

Write-Step "Resolve version metadata"
$commit = (& $git -C $repoRoot rev-parse --short HEAD).Trim()
if ([string]::IsNullOrWhiteSpace($Version)) {
    $Version = ((Get-Content (Join-Path $backendDir "cmd\server\VERSION") -Raw).Trim())
}
$buildDate = Get-Date -Format "yyyy-MM-ddTHH:mm:ssK"

Write-Step "Run backend tests"
Push-Location $backendDir
go test ./...
Pop-Location

if (-not $SkipFrontendTests) {
    Write-Step "Run frontend typecheck"
    Push-Location $frontendDir
    & "node_modules\.bin\vue-tsc.CMD" --noEmit

    Write-Step "Run frontend tests"
    & "node_modules\.bin\vitest.CMD" run

    Write-Step "Build frontend"
    & "node_modules\.bin\vite.CMD" build
    Pop-Location
}

Write-Step "Build release-style backend binary"
$releaseBin = Join-Path $env:TEMP "sub2api-release-smoke.exe"
if (Test-Path $releaseBin) {
    Remove-Item $releaseBin -Force
}
Push-Location $backendDir
go build -tags embed -ldflags "-X 'main.Commit=$commit' -X 'main.Date=$buildDate' -X 'main.BuildType=release' -X 'main.Version=$Version'" -o $releaseBin ./cmd/server
Pop-Location

Write-Step "Check binary version output"
& $releaseBin -version

Write-Step "Smoke test setup mode root/setup/assets"
$smokeData = Join-Path $env:TEMP "sub2api-smoke-windows"
if (Test-Path $smokeData) {
    Remove-Item $smokeData -Recurse -Force
}
New-Item -ItemType Directory -Path $smokeData | Out-Null
$env:DATA_DIR = $smokeData
$env:SERVER_HOST = "127.0.0.1"
$env:SERVER_PORT = "18110"
$setupProc = Start-Process -FilePath $releaseBin -ArgumentList @() -WindowStyle Hidden -PassThru
Start-Sleep -Seconds 3
try {
    $root = Invoke-WebRequest -UseBasicParsing "http://127.0.0.1:18110/" -TimeoutSec 5
    $setup = Invoke-WebRequest -UseBasicParsing "http://127.0.0.1:18110/setup" -TimeoutSec 5
    $asset = Invoke-WebRequest -UseBasicParsing "http://127.0.0.1:18110/assets/index-BLQ1PWy0.js" -TimeoutSec 5
    if ($root.StatusCode -ne 200 -or $setup.StatusCode -ne 200 -or $asset.StatusCode -ne 200) {
        throw "Setup mode smoke test failed"
    }
} finally {
    Stop-ProcIfRunning $setupProc
    Remove-Item Env:DATA_DIR -ErrorAction SilentlyContinue
    Remove-Item Env:SERVER_HOST -ErrorAction SilentlyContinue
    Remove-Item Env:SERVER_PORT -ErrorAction SilentlyContinue
}

if (-not $SkipPortableInstall) {
    Write-Step "Portable PostgreSQL 18 + Redis install success smoke"
    $portableBase = Join-Path $env:TEMP "sub2api-portable-smoke"
    if (Test-Path $portableBase) {
        Remove-Item $portableBase -Recurse -Force
    }
    New-Item -ItemType Directory -Path $portableBase | Out-Null

    $pgZip = Join-Path $portableBase "postgres18.zip"
    $redisZip = Join-Path $portableBase "redis.zip"
    Invoke-WebRequest -UseBasicParsing "https://get.enterprisedb.com/postgresql/postgresql-18.4-1-windows-x64-binaries.zip" -OutFile $pgZip
    Invoke-WebRequest -UseBasicParsing "https://github.com/redis-windows/redis-windows/releases/download/7.2.14/Redis-7.2.14-Windows-x64-msys2.zip" -OutFile $redisZip
    tar -xf $pgZip -C $portableBase
    tar -xf $redisZip -C $portableBase

    $pgBin = Join-Path $portableBase "pgsql\bin"
    $pgData = Join-Path $portableBase "pgdata"
    New-Item -ItemType Directory -Path $pgData | Out-Null
    $env:PATH = "$pgBin;$env:PATH"
    & (Join-Path $pgBin "initdb.exe") -D $pgData -U postgres -A trust --locale=C --encoding=UTF8 | Out-Null

    $pgProc = Start-Process -FilePath (Join-Path $pgBin "postgres.exe") -ArgumentList @("-D", $pgData, "-p", "5438", "-h", "127.0.0.1") -WindowStyle Hidden -PassThru

    $redisDir = Join-Path $portableBase "Redis-7.2.14-Windows-x64-msys2"
    $redisConf = Join-Path $redisDir "redis-smoke.conf"
    $redisData = Join-Path $redisDir "data"
    New-Item -ItemType Directory -Path $redisData | Out-Null
    Set-Content -Path $redisConf -Value @(
        'port 6382',
        'bind 127.0.0.1',
        'save ""',
        'appendonly no',
        'dir ./data'
    )
    $redisProc = Start-Process -FilePath (Join-Path $redisDir "redis-server.exe") -ArgumentList @("redis-smoke.conf") -WorkingDirectory $redisDir -WindowStyle Hidden -PassThru

    if (-not (Wait-Port -Port 5438 -TimeoutSec 25)) {
        throw "Portable PostgreSQL failed to start"
    }
    if (-not (Wait-Port -Port 6382 -TimeoutSec 25)) {
        throw "Portable Redis failed to start"
    }

    $installData = Join-Path $portableBase "appdata"
    New-Item -ItemType Directory -Path $installData | Out-Null
    $env:DATA_DIR = $installData
    $env:SERVER_HOST = "127.0.0.1"
    $env:SERVER_PORT = "18111"
    $setupInstallProc = Start-Process -FilePath $releaseBin -ArgumentList @() -WindowStyle Hidden -PassThru
    if (-not (Wait-Port -Port 18111 -TimeoutSec 20)) {
        throw "Setup install server failed to start"
    }

    try {
        $installBody = '{"database":{"host":"127.0.0.1","port":5438,"user":"postgres","password":"","dbname":"sub2api_release_smoke","sslmode":"disable"},"redis":{"host":"127.0.0.1","port":6382,"password":"","db":0,"enable_tls":false},"admin":{"email":"admin@example.com","password":"StrongPass123"},"server":{"host":"127.0.0.1","port":18112,"mode":"release"}}'
        $installResp = Invoke-WebRequest -UseBasicParsing "http://127.0.0.1:18111/setup/install" -Method POST -ContentType "application/json" -Body $installBody -TimeoutSec 40
        $statusResp = Invoke-WebRequest -UseBasicParsing "http://127.0.0.1:18111/setup/status" -TimeoutSec 10
        if ($installResp.StatusCode -ne 200 -or $statusResp.Content -notmatch '"needs_setup":false') {
            throw "Portable install success smoke failed"
        }
    } finally {
        Stop-ProcIfRunning $setupInstallProc
        Remove-Item Env:DATA_DIR -ErrorAction SilentlyContinue
        Remove-Item Env:SERVER_HOST -ErrorAction SilentlyContinue
        Remove-Item Env:SERVER_PORT -ErrorAction SilentlyContinue
        Stop-ProcIfRunning $pgProc
        Stop-ProcIfRunning $redisProc
    }
}

Write-Step "Done"
Write-Host "Release smoke completed successfully." -ForegroundColor Green
