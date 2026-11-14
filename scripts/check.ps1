#!/usr/bin/env pwsh
#Requires -Version 5.1
# Checks the whole repo the way CI does: formatting, lint, types, build and tests.
#
#   ./scripts/check.ps1          report problems and exit non-zero
#   ./scripts/check.ps1 --fix    reformat and auto-fix what can be fixed, then check
#
# Run it before committing, and AI agents should run it before saying they're done.
# Needs Go 1.26+ (https://go.dev/dl/) and Node.js 24+ (https://nodejs.org/).
$ErrorActionPreference = 'Stop'
Set-Location (Join-Path $PSScriptRoot '..')

$Fix = $false
if ($args.Count -gt 0) {
    if ($args.Count -eq 1 -and $args[0] -eq '--fix') {
        $Fix = $true
    } else {
        [Console]::Error.WriteLine('usage: check.ps1 [--fix]')
        exit 2
    }
}

function Step([string]$Name) {
    Write-Host "`n==> $Name"
}

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

# The directory of every Go package, so gofmt only sees Go code (not node_modules or dist).
$goDirs = @(& go list -f '{{.Dir}}' ./...)
if ($LASTEXITCODE -ne 0) { Die 'go list failed.' }

Step 'Go formatting (gofmt)'
if ($Fix) {
    foreach ($dir in $goDirs) {
        & gofmt -w $dir
        if ($LASTEXITCODE -ne 0) { Die "gofmt -w $dir failed." }
    }
} else {
    $unformatted = @()
    foreach ($dir in $goDirs) {
        $files = @(& gofmt -l $dir)
        if ($LASTEXITCODE -ne 0) { Die "gofmt -l $dir failed." }
        $unformatted += $files
    }
    if ($unformatted.Count -gt 0) {
        Die "Not gofmt'd; run ./scripts/check.ps1 --fix:`n$($unformatted -join "`n")"
    }
}

Push-Location frontend
try {
    if (-not (Test-Path node_modules)) {
        Step 'Installing frontend dependencies (npm ci)'
        npm ci
        if ($LASTEXITCODE -ne 0) { Die 'npm ci failed.' }
    }

    Step 'Frontend lint (oxlint)'
    $lintArgs = if ($Fix) { @('run', 'lint', '--', '--fix') } else { @('run', 'lint') }
    & npm @lintArgs
    if ($LASTEXITCODE -ne 0) { Die 'Frontend lint failed.' }

    Step 'Frontend type check (tsgo)'
    npm run typecheck
    if ($LASTEXITCODE -ne 0) { Die 'Frontend type check failed.' }

    # go vet and go test compile the embedded UI, so frontend/dist must exist first.
    Step 'Frontend build (vite)'
    npm run build
    if ($LASTEXITCODE -ne 0) { Die 'Frontend build failed.' }
} finally {
    Pop-Location
}

Step 'Go vet'
go vet ./...
if ($LASTEXITCODE -ne 0) { Die 'go vet failed.' }

Step 'Go tests'
go test ./...
if ($LASTEXITCODE -ne 0) { Die 'go test failed.' }

Write-Host "`nAll checks passed."
exit 0
