param(
    [Parameter(Position = 0)]
    [ValidateSet("setup", "setup-local", "up", "up-detached", "down", "logs", "ps", "dev-db", "dev-db-down", "dev-api", "dev-worker", "dev-web")]
    [string]$Command = "help"
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

function Ensure-EnvFile {
    param([string]$Template)
    if (-not (Test-Path ".env")) {
        Copy-Item $Template ".env"
        Write-Host "Created .env from $Template"
    }
}

function Install-Deps {
    Push-Location pkg; go mod download; Pop-Location
    Push-Location apps/api; go mod download; Pop-Location
    Push-Location apps/worker; go mod download; Pop-Location
    Push-Location apps/web; npm install; Pop-Location
}

function Invoke-WithEnv {
    param([string[]]$Args)
    & powershell -NoProfile -ExecutionPolicy Bypass -File (Join-Path $PSScriptRoot "load-env.ps1") @Args
}

switch ($Command) {
    "setup" {
        Ensure-EnvFile "deploy\.env.example"
        Install-Deps
    }
    "setup-local" {
        Ensure-EnvFile "deploy\.env.local.example"
        Install-Deps
    }
    "up" {
        docker compose -f deploy/docker-compose.yml up --build
    }
    "up-detached" {
        docker compose -f deploy/docker-compose.yml up --build -d
    }
    "down" {
        docker compose -f deploy/docker-compose.yml down
    }
    "logs" {
        docker compose -f deploy/docker-compose.yml logs -f
    }
    "ps" {
        docker compose -f deploy/docker-compose.yml ps
    }
    "dev-db" {
        docker compose -f deploy/docker-compose.dev.yml up -d
    }
    "dev-db-down" {
        docker compose -f deploy/docker-compose.dev.yml down
    }
    "dev-api" {
        Invoke-WithEnv @("go", "run", "./apps/api/cmd/api")
    }
    "dev-worker" {
        Invoke-WithEnv @("go", "run", "./apps/worker/cmd/worker")
    }
    "dev-web" {
        Push-Location apps/web
        npm run dev
        Pop-Location
    }
    default {
        Write-Host @"
Usage: .\scripts\dev.ps1 <command>

Commands:
  setup          Create .env from deploy/.env.example and install dependencies
  setup-local    Create .env from deploy/.env.local.example and install dependencies
  up             Start full Docker stack (foreground)
  up-detached    Start full Docker stack (background)
  down           Stop full Docker stack
  logs           Follow Docker stack logs
  ps             List Docker stack services
  dev-db         Start Postgres + migrations for native dev
  dev-db-down    Stop Postgres dev stack
  dev-api        Run API on the host (requires dev-db)
  dev-worker     Run worker on the host (requires dev-db)
  dev-web        Run Vite dev server on the host
"@
        exit 1
    }
}
