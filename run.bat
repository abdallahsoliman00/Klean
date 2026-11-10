@echo off
rem Builds Switchboard and starts it at http://localhost:8787, opening the browser.
rem Flags pass through, e.g. run.bat --port 9000 --no-browser
cd /d "%~dp0"
where go >nul 2>nul || goto :nogo
go build -o switchboard.exe ./backend || goto :error
switchboard.exe %*
goto :eof
:nogo
echo Go isn't installed. Get Go 1.26 or newer from https://go.dev/dl/
pause
goto :eof
:error
echo Build failed.
pause
