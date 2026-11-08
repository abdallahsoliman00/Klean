@echo off
rem Starts Switchboard at http://localhost:8787 (pass flags through, e.g. run.bat --port 9000)
cd /d "%~dp0backend"
if not exist .venv (
  echo First run: setting up Python environment...
  python -m venv .venv || goto :error
  .venv\Scripts\python -m pip install -q --upgrade pip
  .venv\Scripts\python -m pip install -q -r requirements.txt || goto :error
)
.venv\Scripts\python -m switchboard %*
goto :eof
:error
echo Setup failed. Make sure Python 3.10+ is installed and on your PATH.
pause
