param(
    [ValidateSet('health', 'pricing', 'monitoring-summary')]
    [string]$Scenario = $(if ($env:SCENARIO) { $env:SCENARIO } else { 'health' }),

    [string]$BaseUrl = $(if ($env:BASE_URL) { $env:BASE_URL } else { 'http://127.0.0.1:18808' }),

    [string]$Duration = $(if ($env:VEGETA_DURATION) { $env:VEGETA_DURATION } else { '30s' }),

    [string]$Rate = $(if ($env:VEGETA_RATE) { $env:VEGETA_RATE } else { '20' }),

    [string]$Timeout = $(if ($env:VEGETA_TIMEOUT) { $env:VEGETA_TIMEOUT } else { '5s' })
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

function Get-ScenarioPath {
    param([string]$Name)

    switch ($Name) {
        'health' { return '/health' }
        'pricing' { return '/api/v1/public/pricing' }
        'monitoring-summary' { return '/api/v1/monitoring/summary' }
        default { throw "Unsupported scenario: $Name" }
    }
}

function ConvertTo-HeaderLines {
    param([hashtable]$HeaderTable)

    $lines = New-Object System.Collections.Generic.List[string]
    foreach ($entry in $HeaderTable.GetEnumerator()) {
        $lines.Add("$($entry.Key): $($entry.Value)")
    }
    return $lines
}

$vegeta = Get-Command vegeta -ErrorAction SilentlyContinue
if (-not $vegeta) {
    throw 'vegeta is not installed or not on PATH. Run vegeta -version first.'
}

$normalizedBaseUrl = $BaseUrl.TrimEnd('/')
$path = Get-ScenarioPath -Name $Scenario
$url = "$normalizedBaseUrl$path"

$headers = @{
    Accept = 'application/json'
}

if ($env:AUTH_TOKEN) {
    $authHeader = if ($env:AUTH_HEADER) { $env:AUTH_HEADER } else { 'Authorization' }
    $authScheme = if ($null -ne $env:AUTH_SCHEME -and $env:AUTH_SCHEME -ne '') { $env:AUTH_SCHEME } else { 'Bearer' }
    if ($authScheme) {
        $headers[$authHeader] = "$authScheme $($env:AUTH_TOKEN)".Trim()
    }
    else {
        $headers[$authHeader] = $env:AUTH_TOKEN
    }
}

if ($env:EXTRA_HEADERS) {
    $extraHeaders = $env:EXTRA_HEADERS | ConvertFrom-Json -AsHashtable
    foreach ($entry in $extraHeaders.GetEnumerator()) {
        $headers[$entry.Key] = [string]$entry.Value
    }
}

$outputDir = Join-Path $PSScriptRoot 'output'
if (-not (Test-Path $outputDir)) {
    New-Item -ItemType Directory -Path $outputDir | Out-Null
}

$stamp = Get-Date -Format 'yyyyMMdd-HHmmss'
$binPath = Join-Path $outputDir "$Scenario-$stamp.bin"
$jsonPath = Join-Path $outputDir "$Scenario-$stamp.json"
$targetFile = Join-Path $outputDir "$Scenario-$stamp.targets.txt"

$targetLines = New-Object System.Collections.Generic.List[string]
$targetLines.Add("GET $url")
$targetLines.AddRange((ConvertTo-HeaderLines -HeaderTable $headers))
$targetLines.Add('')
Set-Content -Path $targetFile -Value $targetLines -Encoding utf8

Write-Host "[perf] scenario=$Scenario"
Write-Host "[perf] url=$url"
Write-Host "[perf] rate=$Rate"
Write-Host "[perf] duration=$Duration"
Write-Host "[perf] timeout=$Timeout"
Write-Host "[perf] targetFile=$targetFile"

Get-Content $targetFile |
    vegeta attack -rate="$Rate" -duration="$Duration" -timeout="$Timeout" |
    Tee-Object -FilePath $binPath |
    vegeta report

Get-Content $binPath |
    vegeta report -type=json | Set-Content -Path $jsonPath -Encoding utf8

Write-Host "[perf] binary report saved: $binPath"
Write-Host "[perf] json report saved: $jsonPath"
