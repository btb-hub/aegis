param(
    [Parameter(Mandatory = $true, ValueFromRemainingArguments = $true)]
    [string[]]$Command
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
$EnvFile = Join-Path $Root ".env"

if (-not (Test-Path $EnvFile)) {
    Write-Error ".env not found. Run 'make setup' or 'make setup-local' first."
}

Get-Content $EnvFile | ForEach-Object {
    $line = $_.Trim()
    if ($line -eq "" -or $line.StartsWith("#")) {
        return
    }
    $eq = $line.IndexOf("=")
    if ($eq -lt 1) {
        return
    }
    $name = $line.Substring(0, $eq).Trim()
    $value = $line.Substring($eq + 1).Trim()
    Set-Item -Path "env:$name" -Value $value
}

Set-Location $Root

if ($Command.Count -eq 1) {
    Invoke-Expression $Command[0]
    exit $LASTEXITCODE
}

& $Command[0] @($Command[1..($Command.Count - 1)])
exit $LASTEXITCODE
