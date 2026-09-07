# ==============================================================================
# ZSMS — Build Verification Script (PowerShell)
# ==============================================================================

Write-Host ">>> Building ZSMS Monorepo Artifacts..." -ForegroundColor Cyan

# 1. Frontend Build Check
if (Get-Command npm.cmd -ErrorAction SilentlyContinue) {
    Write-Host ">>> Building Nuxt Web Frontend..." -ForegroundColor Green
    Push-Location web
    npm.cmd run build
    Pop-Location
} else {
    Write-Host ">>> npm not found, skipping frontend build check." -ForegroundColor Yellow
}

# 2. Go Backend Build Check
if (Get-Command go -ErrorAction SilentlyContinue) {
    Write-Host ">>> Building Go Backend..." -ForegroundColor Green
    Push-Location backend
    go build ./...
    Pop-Location
} else {
    Write-Host ">>> Go not found in PATH." -ForegroundColor Yellow
}
