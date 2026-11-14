If you are an agent working on the frontend, see ./frontend/AGENTS.md

If you are an agent working on the backend, see ./backend/AGENTS.md

# Before you finish
Run `./scripts/check.sh` (or `scripts\check.bat` / `scripts\check.ps1` on Windows) and make it pass: it checks formatting,
lints, type checks, builds the UI, runs `go vet` and the Go tests. Use `--fix` to reformat Go and
apply lint fixes.

To run the app while working, use `./scripts/dev.sh` (backend on :8787 plus the Vite dev server) or
`./scripts/dev.sh --mock` for UI work with no backend. On Windows use `scripts\dev.bat`, or
`scripts\dev.ps1` in PowerShell.

# Committing Chages
If you are to commit changes, don't make the message too large unless some massive changes have taken place where documenting the changes is actually necessary.
