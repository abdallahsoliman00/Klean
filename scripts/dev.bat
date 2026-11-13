@echo off
rem Runs Klean for development: the Go backend plus the Vite dev server with hot reload.
rem   dev.bat                     backend on :8787, UI on http://localhost:5173
rem   dev.bat --mock              UI only, backed by the in-memory mock (no Go server)
rem   dev.bat -- --host 0.0.0.0   extra arguments go to the Vite dev server
rem Ctrl+C in this window, then close the backend window, stops everything.
setlocal
cd /d "%~dp0.."

set MOCK=
if /i "%~1"=="--mock" (
  set MOCK=1
  shift
)

where npm >nul 2>nul || goto :nonode
if not defined MOCK (
  where go >nul 2>nul || goto :nogo
)

if not exist frontend\node_modules (
  echo Installing frontend dependencies...
  cmd /c "cd frontend && npm ci" || goto :error
)

if defined MOCK (
  echo UI:      http://localhost:5173  (mock backend, no Go server)
  cmd /c "cd frontend && npm run dev:mock -- %*"
  goto :eof
)

rem The Go binary embeds frontend\dist and won't compile without it.
if not exist frontend\dist\index.html (
  echo Building the frontend once, so the backend can embed it...
  cmd /c "cd frontend && npm run build" || goto :error
)

if not exist .dev mkdir .dev
go build -o .dev\klean.exe ./backend || goto :error
start "Klean dev backend" cmd /c ".dev\klean.exe --headless --db .dev\klean.db"
echo.
echo backend: http://localhost:8787
echo UI:      http://localhost:5173
echo.

cmd /c "cd frontend && npm run dev -- %*"

rem Stop the backend window this script started.
taskkill /FI "WINDOWTITLE eq Klean dev backend*" /T /F >nul 2>nul
goto :eof

:nogo
echo Go isn't installed. Get Go 1.26 or newer from https://go.dev/dl/
exit /b 1
:nonode
echo Node.js isn't installed. Get Node.js 24 or newer from https://nodejs.org/
exit /b 1
:error
exit /b 1
