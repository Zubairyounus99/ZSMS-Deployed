# ==============================================================================
# ZSMS — Local Development Launch Script (PowerShell)
# ==============================================================================

Write-Host ">>> Starting ZSMS Local Development Environment..." -ForegroundColor Cyan

if (-not (Test-Path ".env")) {
    Write-Host ">>> .env not found, copying from .env.example..." -ForegroundColor Yellow
    Copy-Item ".env.example" ".env"
}

if (Get-Command docker -ErrorAction SilentlyContinue) {
    Write-Host ">>> Launching Docker Compose stack (dev)..." -ForegroundColor Green
    docker compose -f docker-compose.dev.yml up -d
} else {
    Write-Host ">>> Docker not found in PATH. Please ensure Docker Desktop is running." -ForegroundColor Red
}
