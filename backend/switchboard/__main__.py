"""Run with:  python -m switchboard  (from the backend folder)

Settings (environment variables or flags):
  --host / SWITCHBOARD_HOST          default 127.0.0.1
  --port / SWITCHBOARD_PORT          default 8787
  --db   / SWITCHBOARD_DB            default ./data/switchboard.db
  --frontend / SWITCHBOARD_FRONTEND  default ../frontend ("" to serve the API only)
"""
import argparse
import os
from pathlib import Path

import uvicorn

from .app import create_app

HERE = Path(__file__).resolve().parent.parent


def main():
    ap = argparse.ArgumentParser(prog="switchboard")
    ap.add_argument("--host", default=os.environ.get("SWITCHBOARD_HOST", "127.0.0.1"))
    ap.add_argument("--port", type=int, default=int(os.environ.get("SWITCHBOARD_PORT", "8787")))
    ap.add_argument("--db", default=os.environ.get("SWITCHBOARD_DB", str(HERE / "data" / "switchboard.db")))
    ap.add_argument("--frontend", default=os.environ.get("SWITCHBOARD_FRONTEND", str(HERE.parent / "frontend")))
    args = ap.parse_args()

    app = create_app(args.db, args.frontend or None)
    shown = "localhost" if args.host in ("127.0.0.1", "0.0.0.0") else args.host
    print(f"\n  Switchboard is running at http://{shown}:{args.port}\n  Database: {args.db}\n")
    uvicorn.run(app, host=args.host, port=args.port, log_level="warning")


if __name__ == "__main__":
    main()
