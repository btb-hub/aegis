param(
    [Parameter(Position=0)]
    [ValidateSet('up','down','status','logs','spike','dip','healthy','alert')]
    [string]$Action = 'up',
    [string]$ModulePath
)
$ErrorActionPreference = 'Stop'
$taskRepoRoot = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot '..')).Path
$taskCompose = Join-Path $taskRepoRoot 'devtools/anomaly-sandbox/compose.yml'
if (-not $ModulePath -and $env:AEGIS_ANOMALY_SOURCE_DIR) { $ModulePath = $env:AEGIS_ANOMALY_SOURCE_DIR }
if (-not $ModulePath) {
    foreach ($candidate in @((Join-Path $taskRepoRoot '.worktrees/aegis-anomaly-detector'), (Join-Path $taskRepoRoot '../aegis-anomaly-detector'))) {
        if (Test-Path -LiteralPath (Join-Path $candidate 'go.mod')) { $ModulePath = $candidate; break }
    }
}
if ($ModulePath -and $Action -eq 'up') {
    $env:AEGIS_ANOMALY_SOURCE_DIR = (Resolve-Path -LiteralPath $ModulePath).Path
}
if ($Action -eq 'up' -and (-not $ModulePath -or -not (Test-Path -LiteralPath (Join-Path $ModulePath 'Dockerfile')))) {
    throw 'Clone the private aegis-anomaly-detector repository, then pass -ModulePath <checkout>.'
}
if ($Action -in @('spike','dip','healthy','alert')) {
    $taskEndpoint = if ($Action -eq 'alert') { 'alert' } else { 'scenario' }
    $taskPayload = if ($Action -eq 'alert') { '{}' } else { @{scenario=$Action} | ConvertTo-Json -Compress }
    Invoke-RestMethod -Method Post -Uri "http://localhost:18081/api/$taskEndpoint" -ContentType 'application/json' -Body $taskPayload
    exit
}
switch ($Action) {
    'up' {
        docker compose -f $taskCompose up -d --build
        if ($LASTEXITCODE -ne 0) { throw 'Sandbox startup failed. Inspect Docker Compose output.' }
        docker compose -f $taskCompose wait bootstrap
        if ($LASTEXITCODE -ne 0) { throw 'Bootstrap failed. Run ./scripts/anomaly-sandbox.ps1 logs.' }
        Write-Output 'Controls http://localhost:18081 | Aegis http://127.0.0.1:18080'
    }
    'down' { docker compose -f $taskCompose down }
    'status' { docker compose -f $taskCompose ps -a }
    'logs' { docker compose -f $taskCompose logs --tail 100 }
}
if ($LASTEXITCODE -ne 0) { throw "Docker Compose $Action failed." }
