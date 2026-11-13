@echo off
rem Checks the whole repo the way CI does: formatting, lint, types, build and tests.
setlocal
cd /d "%~dp0.."

set FIX=
if /i "%~1"=="--fix" set FIX=1
if defined FIX shift

where go >nul 2>nul || goto :nogo
where npm >nul 2>nul || goto :nonode

echo.
echo ==^> Go formatting (gofmt)
set UNFORMATTED=
for /f "delims=" %%d in ('go list -f "{{.Dir}}" ./...') do (
  if defined FIX (
    gofmt -w "%%d" || goto :error
  ) else (
    for /f "delims=" %%f in ('gofmt -l "%%d"') do (
      echo %%f
      set UNFORMATTED=1
    )
  )
)
if defined UNFORMATTED (
  echo Not gofmt'd; run scripts\check.bat --fix
  goto :error
)

if not exist frontend\node_modules (
  echo.
  echo ==^> Installing frontend dependencies (npm ci)
  cmd /c "cd frontend && npm ci" || goto :error
)

echo.
echo ==^> Frontend lint (oxlint)
if defined FIX (
  cmd /c "cd frontend && npm run lint -- --fix" || goto :error
) else (
  cmd /c "cd frontend && npm run lint" || goto :error
)

echo.
echo ==^> Frontend type check (tsgo)
cmd /c "cd frontend && npm run typecheck" || goto :error

rem go vet and go test compile the embedded UI, so frontend\dist must exist first.
echo.
echo ==^> Frontend build (vite)
cmd /c "cd frontend && npm run build" || goto :error

echo.
echo ==^> Go vet
go vet ./... || goto :error

echo.
echo ==^> Go tests
go test ./... || goto :error

echo.
echo All checks passed.
exit /b 0

:nogo
echo Go isn't installed. Get Go 1.26 or newer from https://go.dev/dl/
exit /b 1
:nonode
echo Node.js isn't installed. Get Node.js 24 or newer from https://nodejs.org/
exit /b 1
:error
exit /b 1
