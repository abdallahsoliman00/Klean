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
│   │   ├── tools/            the tools the model can call; web search and page fetching
│   │   ├── launch/           opens the browser, and a terminal window when double-clicked
│   │   └── fakeprovider/     a fake OpenAI-compatible server for tests
│   └── cmd/fakeprovider/     the same fake server as a program, for UI work
└── frontend/                 React + Vite + TypeScript UI, built into dist/ and embedded into the executable
    ├── embed.go              embeds dist/ (index.html at its root)
    ├── index.html            page shell and the pre-paint theme script
    ├── vite.config.ts        build, dev server and /api proxy
    └── src/
        ├── config.ts         every timeout, size, limit, default, storage key and endpoint
        ├── api/              the only code that talks to the backend (+ mock/ for UI work alone)
        ├── store/            app state, one slice per concern (zustand)
        ├── components/       one folder per area, each with its own CSS
        ├── hooks/            small reusable behaviours (scrolling, autosize, shortcuts…)
        ├── lib/              plain helpers: markdown, highlighting, tokens, theme…
        └── styles/           design tokens, base styles, fonts (self-hosted, works offline)
```

## Run it

Needs Go 1.26+ (https://go.dev/dl/) and Node.js 24+ (https://nodejs.org/) to build.

**macOS / Linux**
```bash
./run.sh
```

**Windows**
```bat
run.bat
```

Both build the UI and `./klean` (dependencies are downloaded the first time), start it, and open
**http://localhost:8787** in your browser. Manual equivalent:
`(cd frontend && npm ci && npm run build) && go build -o klean ./backend && ./klean`

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
the history as input items (messages, and earlier tool calls with their outputs), and `store: false`
so the provider keeps no copy. A provider that only
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

## Frontend

The UI is React + TypeScript, built with Vite into `frontend/dist`, which `frontend/embed.go`
embeds into the executable. Build it before any `go build`, `go vet` or `go test`:

```bash
cd frontend
npm ci
npm run build       # typecheck (tsgo), then vite build into dist/
```

Other scripts:

| Script              | What it does |
|---------------------|--------------|
| `npm run dev`       | Vite dev server with hot reload; `/api` is proxied to the backend on http://localhost:8787 (start it with `go run ./backend --no-browser`) |
| `npm run dev:mock`  | the same, with an in-memory mock backend instead (`src/api/mock`), for UI work with no backend at all |
| `npm run typecheck` | `tsgo --noEmit` |
| `npm run lint`      | `oxlint` |

Tunable values (timeouts, sizes, limits, defaults, storage keys, endpoint paths) live in
`frontend/src/config.ts`. Markdown, sanitising and highlighting come from the `marked`, `dompurify`
and `highlight.js` packages; fonts are bundled from `src/assets/fonts`, so the app works offline.

To serve a built copy without rebuilding the executable, point the backend at it:
`go run ./backend --frontend frontend/dist`. To host the UI on another origin, start the backend
with `--frontend none` and build with `VITE_KLEAN_API=http://localhost:8787 npm run build`.

## Tools

The model can call tools while it answers. They are sent as Responses API function tools, in three
groups that the composer switches on and off:

| Group | Tool | What it does |
|---|---|---|
| `artifacts` | `create_artifact` | Puts a standalone page, SVG, document or source file (`kind`: `html`, `svg`, `markdown`, `code`) in the side panel |
| | `update_artifact` | Replaces an artifact's content; the version goes up by one |
| `ask` | `ask_user` | Asks you a question with up to four options and ends the turn until you answer |
| `web` | `web_search` | Top six results (title, URL, snippet) |
| | `fetch_page` | Downloads a page and returns its readable text (first 12,000 characters) |

When the model calls tools, Klean runs them, sends the results back and asks again, up to
8 requests per reply. Tool calls and their outputs are saved with the message and replayed in later
requests, so the model remembers what it searched for and what its artifacts contain.

**Web search** uses DuckDuckGo's HTML results by default, which needs no key but sometimes turns
automated requests away. For dependable search, pick Brave Search or Tavily in settings and save an
API key for it; the key stays in the database and is never sent to the browser.

**Page fetching runs on your computer**, with URLs chosen by the model, so it only fetches
`http(s)` URLs on the public internet: loopback, private-network and link-local addresses are refused,
also when a public name or a redirect leads to one. It does not go through a proxy.

**Models without tool support:** if the provider rejects the request because of the tools, the error
says so. Turn the tools off in the composer and send again; with all groups off, no tools are sent
and earlier tool calls are left out of the history.

## How context is handled

Every request sends the chat's system prompt plus as many recent messages as fit in
75% of the chat's context window (the rest is left for the reply). Tokens are estimated at
about four characters each. Older messages stay in the database and in the UI; they're just not sent.
Tool calls count too (arguments and outputs), as do the tool definitions; a call is never sent
without its output.
Set the window per chat from the ring in the top bar.

## API

| Method | Path | |
|---|---|---|
| GET/POST | `/api/providers` | list / add (`name`, `baseUrl`, `apiKey?`) |
| PATCH/DELETE | `/api/providers/{id}` | edit / remove (`apiKey: ""` clears the key) |
| GET | `/api/providers/{id}/models` | models from the provider |
| GET/POST | `/api/chats` | list / create |
| GET/PATCH/DELETE | `/api/chats/{id}` | read with `messages` and `artifacts` / update settings or title / delete |
| POST | `/api/chats/{id}/restore` | undo a delete (kept for 24 hours) |
| POST | `/api/chats/{id}/messages` | send; streams Server-Sent Events (below) |
| DELETE | `/api/artifacts/{id}` | remove an artifact |
| GET/PATCH | `/api/settings` | `{searchProvider, hasSearchKey}`; PATCH takes `searchProvider` (`duckduckgo`, `brave`, `tavily`) and `searchKey` (`""` clears it; never returned) |

`/messages` body: `{content, providerId, model}`, plus any of:

- `editMessageId` to replace a message and everything after it, or `regenerate: true` to replace
  the last reply. Both also delete the artifacts the removed replies created.
- `tools`: the enabled tool groups, a subset of `["artifacts", "ask", "web"]`. Left out, all are
  enabled; `[]` sends no tools.
- `answer: {callId, text}` to answer a pending `ask_user` question. No user message is created and
  the model continues in a new assistant message. Sending an ordinary message instead closes the
  question as skipped.

Events, in the order they can occur:

| Event | Data | |
|---|---|---|
| `user` | the saved user message | first, unless regenerating or answering |
| `answered` | `{messageId, part}` | first when the request had `answer`: the question's part, now done |
| `token` | `{delta}` | text; may come before, between and after tool calls |
| `tool_start` | `{id, name}` | the model began a tool call |
| `tool_run` | `{id, name, args}` | arguments complete, the tool is running |
| `artifact` | `{artifact}` | the full artifact after a create or update, before its `tool_end` |
| `tool_end` | `{part}` | the finished tool part |
| `done` | `{message, title}` | the saved reply. If its last part is a pending `ask_user`, it is waiting for you |
| `error` | `{message}` | instead of `done` |

Assistant messages carry `parts`, the reply in the order it was produced:
`{type: "text", text}` and
`{type: "tool", id, name, args, status: "pending"|"done"|"error", result, error?}`;
`content` is the text parts joined. `result` is `{artifactId, title, kind, language, version}` for the
artifact tools, `{answer}` for `ask_user`, `{query, results: [{title, url, snippet}]}` for
`web_search` and `{url, title, chars}` for `fetch_page`; it is `null` while pending and on error.

Closing the connection stops generation; the partial reply, including the tool calls that
finished, is saved and marked as stopped.

## Tests

```bash
(cd frontend && npm ci && npm run build)   # the Go packages embed the built UI
go test ./...
```

The tests run against a small fake provider that speaks the streaming Responses API. You can also
point the UI at it (`go run ./backend/cmd/fakeprovider 9100` → base URL `http://127.0.0.1:9100/v1`).

It answers every message with the same canned reply, except for these words in the message:

| Word | Effect |
|---|---|
| `/artifact`, `/code`, `/svg` | creates an HTML page, a Python file (`kind: code`), an SVG image |
| `/update` | updates the most recent artifact in the chat |
| `/ask` | asks a question with three options |
| `/search words…` | searches the web for the words after it |
| `/fetch URL` | fetches that URL (`https://example.com/` if none is given) |
| `/loop` | calls a tool on every request, to show the round limit |
| `NOTOOLS` | rejects the request with a 400 if it carries tools |
| `SLOW`, `FAIL`, `MIDFAIL` | streams slowly, answers 500, fails partway through |

The tool words only act when the matching tool group is enabled; after the tool's output comes
back, the fake replies with a line of text quoting it.
