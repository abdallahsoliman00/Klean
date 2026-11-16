#!/usr/bin/env pwsh
#Requires -Version 5.1
# Builds a single-file Klean for each platform into dist/.
# Each binary carries the UI inside it; copy it anywhere and double-click it.
$ErrorActionPreference = 'Stop'
Set-Location (Join-Path $PSScriptRoot '..')

function Die([string]$Message) {
    [Console]::Error.WriteLine($Message)
    exit 1
}

function Assert-LastExit([string]$What) {
    if ($LASTEXITCODE -ne 0) { Die "$What failed." }
}

# the UI is built once and embedded into every binary
Push-Location frontend
try {
    npm ci
    Assert-LastExit 'npm ci'
    npm run build
    Assert-LastExit 'npm run build'
} finally {
    Pop-Location
}

Remove-Item -Recurse -Force dist -ErrorAction SilentlyContinue
New-Item -ItemType Directory dist | Out-Null

foreach ($target in 'linux/amd64', 'linux/arm64', 'darwin/amd64', 'darwin/arm64', 'windows/amd64', 'windows/arm64') {
    $os, $arch = $target -split '/'
    $out = "dist/klean-$os-$arch"
    if ($os -eq 'windows') { $out += '.exe' }
    Write-Host "building $out"
    $env:CGO_ENABLED = '0'
    $env:GOOS = $os
    $env:GOARCH = $arch
    go build -trimpath -ldflags '-s -w' -o $out ./backend
    Assert-LastExit "go build $out"
}
