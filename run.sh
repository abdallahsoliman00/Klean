#!/usr/bin/env sh
# Starts Switchboard at http://localhost:8787 (pass flags through, e.g. ./run.sh --port 9000)
set -e
cd "$(dirname "$0")/backend"
PY=python3; command -v python3 >/dev/null 2>&1 || PY=python
if [ ! -d .venv ]; then
  echo "First run: setting up Python environment..."
  $PY -m venv .venv
  .venv/bin/pip install -q --upgrade pip
  .venv/bin/pip install -q -r requirements.txt
fi
exec .venv/bin/python -m switchboard "$@"
