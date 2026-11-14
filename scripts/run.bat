@echo off
rem Builds Klean and starts it at http://localhost:8787, opening the browser.
rem Flags pass through, e.g. scripts\run.bat --port 9000 --no-browser
cd /d "%~dp0.."
where go >nul 2>nul || goto :nogo
where npm >nul 2>nul || goto :nonode
pushd frontend
call npm ci || goto :uierror
call npm run build || goto :uierror
popd
go build -o klean.exe ./backend || goto :error
klean.exe %*
goto :eof
:nogo
echo Go isn't installed. Get Go 1.26 or newer from https://go.dev/dl/
pause
goto :eof
:nonode
echo Node.js isn't installed. Get Node.js 24 or newer from https://nodejs.org/
pause
goto :eof
:uierror
popd
:error
echo Build failed.
pause
