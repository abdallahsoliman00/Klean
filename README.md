# Klean

A local chat UI for any OpenAI-compatible server (Ollama, LM Studio, vLLM, llama.cpp, OpenRouter, OpenAI…).
It ships as **one executable** with the UI built in: copy it anywhere, double-click it, and it opens
in your browser. The backend owns the chats, context and provider keys (SQLite);
the frontend is static files that only talk to the backend's `/api`.

```
klean/
├── go.mod                    one Go module for the whole repo (so the UI can be embedded)
├── backend/                  Go — API, SQLite, calls to providers
│   ├── main.go               flags and startup
│   ├── internal/
│   │   ├── server/           HTTP routes (+ API tests)
│   │   ├── store/            SQLite schema, queries, and where the database lives
│   │   ├── llm/              openai-go SDK: GET {base}/models, streaming POST {base}/responses
│   │   ├── prompt/           decides which messages fit the context window
│   │   ├── launch/           opens the browser, and a terminal window when double-clicked
│   │   └── fakeprovider/     a fake OpenAI-compatible server for tests
│   └── cmd/fakeprovider/     the same fake server as a program, for UI work
└── frontend/                 static UI (no build step), embedded into the executable
    ├── embed.go              tells Go which files to embed
    ├── index.html
    ├── js/api.js             the only file that talks to the backend
    ├── js/api.mock.js        in-memory stand-in, for UI work without a backend
    └── vendor/               markdown, sanitiser, highlighter, fonts (works offline)
```

## Run it

Needs Go 1.26+ to build (https://go.dev/dl/).

**macOS / Linux**
```bash
./run.sh
```

**Windows**
```bat
run.bat
```

Both build `./klean` (dependencies are downloaded the first time), start it, and open
**http://localhost:8787** in your browser. Manual equivalent: `go build -o klean ./backend && ./klean`

### Build for distribution

```bash
./build.sh
```

puts a single self-contained executable per platform in `dist/` (Linux, macOS and Windows; amd64
and arm64). Each one carries the UI inside it and needs nothing else installed: the SQLite driver
is pure Go, so there's no cgo.

### What happens when it starts

- **The browser opens** at the app's address. Use `--no-browser` to skip that.
- **Double-clicked in a file manager**, it opens its own terminal window showing the address and
  where the database is. **Closing that window stops Klean** and frees the port. On Windows,
  Explorer gives it a console window. On macOS, Finder opens it in Terminal. On Linux, it finds your
  desktop's terminal: `$TERMINAL`, `xdg-terminal-exec`, Konsole, Ptyxis, GNOME Terminal and others.
- **Started from a terminal**, it stays in that terminal; Ctrl+C stops it.
- **Already running?** Starting it again just opens the browser at the running copy.
- If an error stops it in its own window, the window waits for Enter so you can read why.
- For a service or autostart entry, use `--headless`: no window, no browser.

## Responses API

Replies use the OpenAI **Responses API** (`POST {base}/responses`, streamed) through the official
[openai-go](https://github.com/openai/openai-go) SDK. The chat's system prompt is sent as `instructions`,
the history as input messages, and `store: false` so the provider keeps no copy. A provider that only
has Chat Completions will answer 404, and the error says so. If that happens, update the server
or check its docs for `/v1/responses` support.

`OPENAI_API_KEY`, `OPENAI_BASE_URL` and the other `OPENAI_*` variables are ignored: each provider
only gets the key saved for it.

## Add a provider

Open **Providers & settings** (bottom left) → **Add provider**, then enter a name and the base URL.
The base URL is the part before `/models` and `/responses`:

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
| `--port`       | `KLEAN_PORT`      | `8787`                      |
| `--host`       | `KLEAN_HOST`      | `127.0.0.1` (this computer only) |
| `--db`         | `KLEAN_DB`        | per-user data folder, below |
| `--frontend`   | `KLEAN_FRONTEND`  | the built-in UI; a folder path serves that instead, `none` = API only |
| `--no-browser` | `KLEAN_NO_BROWSER=1` | off                      |
| `--headless`   |                         | off (no browser, no terminal window) |
|                | `KLEAN_CORS`      | `*`                         |

Example: `./klean --port 9000 --db ~/chats.db`

### Where chats are stored

The database lives in a per-user folder rather than next to the executable, so the binary can be
moved, renamed or replaced without losing anything:

| OS      | Default database |
|---------|------------------|
| Linux   | `~/.local/share/klean/klean.db` (or `$XDG_DATA_HOME/klean/`) |
| macOS   | `~/Library/Application Support/Klean/klean.db` |
| Windows | `%LOCALAPPDATA%\Klean\klean.db` |

`--db` or `KLEAN_DB` puts it anywhere else.

### Working on the UI

The executable serves the copy of `frontend/` it was built with. To see edits without rebuilding,
serve the folder directly: `go run ./backend --frontend frontend`, then reload the page.

## Running the frontend separately

The backend serves the frontend by default. To host the UI elsewhere (another port, a dev server),
start the backend with `--frontend none` and set the backend address before `js/api.js` loads:

```html
<script>window.KLEAN_API = "http://localhost:8787";</script>
<script src="js/api.js"></script>
```

To work on the UI with no backend at all, load `js/api.mock.js` instead of `js/api.js`.

## How context is handled

Every request sends the chat's system prompt plus as many recent messages as fit in
75% of the chat's context window (the rest is left for the reply). Tokens are estimated at
about four characters each. Older messages stay in the database and in the UI; they're just not sent.
Set the window per chat from the ring in the top bar.

## API

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
go test ./...
```

The tests run against a small fake provider that speaks the streaming Responses API. You can also
point the UI at it (`go run ./backend/cmd/fakeprovider 9100` → base URL `http://127.0.0.1:9100/v1`).
