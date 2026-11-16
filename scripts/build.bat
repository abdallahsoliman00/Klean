@echo off
setlocal

rem Run from the repository root (the parent of this script's directory).
cd /d "%~dp0.." || goto :fail

rem Build the UI once; it is embedded into each Go binary.
pushd frontend || goto :fail
npm ci
if errorlevel 1 goto :npm_ci_failed
npm run build
if errorlevel 1 goto :frontend_build_failed
popd

if exist dist rmdir /s /q dist
if errorlevel 1 goto :dist_cleanup_failed
mkdir dist
if errorlevel 1 goto :dist_create_failed

for %%T in (linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64) do call :build_target %%T || goto :fail

echo Build completed successfully.
exit /b 0

:build_target
for /f "tokens=1,2 delims=/" %%A in ("%~1") do (
    set "GOOS=%%A"
    set "GOARCH=%%B"
)
set "OUT=dist\klean-%GOOS%-%GOARCH%"
if /i "%GOOS%"=="windows" set "OUT=%OUT%.exe"
echo building %OUT%
set "CGO_ENABLED=0"
go build -trimpath -ldflags "-s -w" -o "%OUT%" ./backend
if errorlevel 1 (
    >&2 echo go build %OUT% failed.
    exit /b 1
)
exit /b 0

:npm_ci_failed
popd
>&2 echo npm ci failed.
goto :fail

:frontend_build_failed
popd
>&2 echo npm run build failed.
goto :fail

:dist_cleanup_failed
>&2 echo Failed to remove dist.
goto :fail

:dist_create_failed
>&2 echo Failed to create dist.
goto :fail

:fail
>&2 echo Build failed.
exit /b 1
