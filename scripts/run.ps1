#!/usr/bin/env pwsh
#Requires -Version 5.1
# Builds Klean and starts it at http://localhost:8787, opening the browser.
# Flags pass through, e.g. ./scripts/run.ps1 --port 9000 --no-browser
$ErrorActionPreference = 'Stop'
Set-Location (Join-Path $PSScriptRoot '..')

function Die([string]$Message) {
    [Console]::Error.WriteLine($Message)
    exit 1
}

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Die "Go isn't installed. Get Go 1.26 or newer from https://go.dev/dl/"
}
if (-not (Get-Command npm -ErrorAction SilentlyContinue)) {
    Die "Node.js isn't installed. Get Node.js 24 or newer from https://nodejs.org/"
}

Push-Location frontend
try {
    npm ci
    if ($LASTEXITCODE -ne 0) { Die 'npm ci failed.' }
    npm run build
    if ($LASTEXITCODE -ne 0) { Die 'npm run build failed.' }
} finally {
    Pop-Location
}

$exe = if ($env:OS -eq 'Windows_NT') { 'klean.exe' } else { 'klean' }
go build -o $exe ./backend
if ($LASTEXITCODE -ne 0) { Die 'go build failed.' }

& (Join-Path $PWD $exe) @args
exit $LASTEXITCODE
