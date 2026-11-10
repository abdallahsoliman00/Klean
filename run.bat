@echo off
rem Starts Switchboard at http://localhost:8787 (pass flags through, e.g. run.bat --port 9000)
cd /d "%~dp0backend"
where go >nul 2>nul || goto :nogo
go build -o switchboard.exe . || goto :error
switchboard.exe %*
goto :eof
:nogo
echo Go isn't installed. Get Go 1.26 or newer from https://go.dev/dl/
pause
goto :eof
:error
echo Build failed.
pause
