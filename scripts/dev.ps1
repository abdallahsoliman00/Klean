#!/usr/bin/env pwsh
#Requires -Version 5.1
# Runs Klean for development: the Go backend plus the Vite dev server with hot reload.
#
#   ./scripts/dev.ps1                     backend on :8787, UI on http://localhost:5173
#   ./scripts/dev.ps1 --mock              UI only, backed by the in-memory mock (no Go server)
#   ./scripts/dev.ps1 -- --host 0.0.0.0   extra arguments go to the Vite dev server
#
# Ctrl+C stops both. The dev database and binary live in .dev/ (gitignored).
# Needs Go 1.26+ (https://go.dev/dl/) and Node.js 24+ (https://nodejs.org/).
$ErrorActionPreference = 'Stop'
Set-Location (Join-Path $PSScriptRoot '..')

function Die([string]$Message) {
    [Console]::Error.WriteLine($Message)
    exit 1
}

$Mock = $false
$ViteArgs = @($args)
if ($ViteArgs.Count -gt 0 -and $ViteArgs[0] -eq '--mock') {
    $Mock = $true
    $ViteArgs = @($ViteArgs | Select-Object -Skip 1)
}
if ($ViteArgs.Count -gt 0 -and $ViteArgs[0] -eq '--') {
    $ViteArgs = @($ViteArgs | Select-Object -Skip 1)
}

if (-not (Get-Command npm -ErrorAction SilentlyContinue)) {
    Die "Node.js isn't installed. Get Node.js 24 or newer from https://nodejs.org/"
}
if (-not $Mock -and -not (Get-Command go -ErrorAction SilentlyContinue)) {
    Die "Go isn't installed. Get Go 1.26 or newer from https://go.dev/dl/"
}

if (-not (Test-Path 'frontend/node_modules')) {
    Write-Host 'Installing frontend dependencies...'
    Push-Location frontend
    try {
        npm ci
        if ($LASTEXITCODE -ne 0) { Die 'npm ci failed.' }
    } finally {
        Pop-Location
    }
}

if ($Mock) {
    Write-Host 'UI:      http://localhost:5173  (mock backend, no Go server)'
    Push-Location frontend
    try {
        $npmArgs = @('run', 'dev:mock', '--') + $ViteArgs
        & npm @npmArgs
        exit $LASTEXITCODE
    } finally {
        Pop-Location
    }
}

# The Go binary embeds frontend/dist and won't compile without it.
if (-not (Test-Path 'frontend/dist/index.html')) {
    Write-Host 'Building the frontend once, so the backend can embed it...'
    Push-Location frontend
    try {
        npm run build
        if ($LASTEXITCODE -ne 0) { Die 'npm run build failed.' }
    } finally {
        Pop-Location
    }
}

New-Item -ItemType Directory .dev -Force | Out-Null
$exe = if ($env:OS -eq 'Windows_NT') { '.dev/klean.exe' } else { '.dev/klean' }
go build -o $exe ./backend
if ($LASTEXITCODE -ne 0) { Die 'go build failed.' }

$db = Join-Path '.dev' 'klean.db'
$backend = Start-Process -FilePath (Resolve-Path $exe).Path `
    -ArgumentList @('--headless', '--db', "`"$db`"") -PassThru -NoNewWindow

Write-Host ''
Write-Host 'backend: http://localhost:8787'
Write-Host 'UI:      http://localhost:5173'
Write-Host ''

$code = 0
Push-Location frontend
try {
    $npmArgs = @('run', 'dev', '--') + $ViteArgs
    & npm @npmArgs
    $code = $LASTEXITCODE
} finally {
    Pop-Location
    if ($backend -and -not $backend.HasExited) {
        Stop-Process -Id $backend.Id -Force -ErrorAction SilentlyContinue
    }
}
exit $code
