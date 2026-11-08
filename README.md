# Switchboard

A local chat UI for any OpenAI-compatible server (Ollama, LM Studio, vLLM, llama.cpp, OpenRouter, OpenAI…).
The frontend and backend are separate: the backend owns the chats, context and provider keys (SQLite);
the frontend is static files that only talk to the backend's `/api`.

```
switchboard/
├── backend/            Python (FastAPI) — API, SQLite, calls to providers
│   ├── switchboard/
│   │   ├── app.py      HTTP routes
│   │   ├── db.py       SQLite schema and queries
│   │   ├── llm.py      GET {base}/models, streaming POST {base}/chat/completions
│   │   └── context.py  decides which messages fit the context window
│   ├── tests/          API tests + a fake OpenAI server
│   └── data/           switchboard.db is created here on first run
└── frontend/           static UI (no build step)
    ├── index.html
    ├── js/api.js       the only file that talks to the backend
    ├── js/api.mock.js  in-memory stand-in, for UI work without a backend
    └── vendor/         markdown, sanitiser, highlighter, fonts (works offline)
```

## Run it

Needs Python 3.10+.

**macOS / Linux**
```bash
./run.sh
```

**Windows**
```bat
run.bat
```

Both create a virtual environment in `backend/.venv` the first time, install three packages,
and start the server. Then open **http://localhost:8787**.

Manual equivalent:
```bash
cd backend
python -m venv .venv && source .venv/bin/activate   # Windows: .venv\Scripts\activate
pip install -r requirements.txt
python -m switchboard
```

## Add a provider

Open **Providers & settings** (bottom left) → **Add provider**, then enter a name and the base URL.
The base URL is the part before `/models` and `/chat/completions`:

| Provider   | Base URL                        | API key |
|------------|---------------------------------|---------|
| Ollama     | `http://localhost:11434/v1`     | none    |
| LM Studio  | `http://localhost:1234/v1`      | none    |
| llama.cpp  | `http://localhost:8080/v1`      | none    |
| OpenRouter | `https://openrouter.ai/api/v1`  | yes     |
| OpenAI     | `https://api.openai.com/v1`     | yes     |

Models are loaded from `{base}/models`; use **Reload** in the model picker (Ctrl+K) after pulling new ones.
API keys are kept in the SQLite database and never sent to the browser.

## Options

| Flag           | Environment variable    | Default                     |
|----------------|-------------------------|-----------------------------|
| `--port`       | `SWITCHBOARD_PORT`      | `8787`                      |
| `--host`       | `SWITCHBOARD_HOST`      | `127.0.0.1` (this computer only) |
| `--db`         | `SWITCHBOARD_DB`        | `backend/data/switchboard.db` |
| `--frontend`   | `SWITCHBOARD_FRONTEND`  | `../frontend` (`""` = API only) |
|                | `SWITCHBOARD_CORS`      | `*`                         |

Example: `python -m switchboard --port 9000 --db ~/chats.db`

## Running the frontend separately

The backend serves the frontend by default. To host the UI elsewhere (another port, a dev server),
start the backend with `--frontend ""` and set the backend address before `js/api.js` loads:

```html
<script>window.SWITCHBOARD_API = "http://localhost:8787";</script>
<script src="js/api.js"></script>
```

To work on the UI with no backend at all, load `js/api.mock.js` instead of `js/api.js`.

## How context is handled

Every request sends the chat's system prompt plus as many recent messages as fit in
75% of the chat's context window (the rest is left for the reply). Tokens are estimated at
about four characters each. Older messages stay in the database and in the UI; they're just not sent.
Set the window per chat from the ring in the top bar.

## API

Interactive docs are at **http://localhost:8787/api/docs**.

| Method | Path | |
|---|---|---|
| GET/POST | `/api/providers` | list / add (`name`, `baseUrl`, `apiKey?`) |
| PATCH/DELETE | `/api/providers/{id}` | edit / remove (`apiKey: ""` clears the key) |
| GET | `/api/providers/{id}/models` | models from the provider |
| GET/POST | `/api/chats` | list / create |
| GET/PATCH/DELETE | `/api/chats/{id}` | read with messages / update settings or title / delete |
| POST | `/api/chats/{id}/restore` | undo a delete (kept for 24 hours) |
| POST | `/api/chats/{id}/messages` | send; streams Server-Sent Events: `user`, `token`, `done` or `error` |

`/messages` body: `{content, providerId, model}`, plus `editMessageId` to replace a message
and everything after it, or `regenerate: true` to replace the last reply.
Closing the connection stops generation; the partial reply is saved and marked as stopped.

## Tests

```bash
cd backend && pip install pytest && python -m pytest tests
```

`tests/fake_openai.py` is a small fake provider you can also point the UI at
(`python tests/fake_openai.py 9100` → base URL `http://127.0.0.1:9100/v1`).
