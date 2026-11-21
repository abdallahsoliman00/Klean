# Development

You need Go 1.26+ (https://go.dev/dl/) and Node.js 24+ (https://nodejs.org/). Each script comes as `.sh`, `.bat` and
`.ps1` in `scripts/`.

| Script | What it does |
|---|---|
| `run` | Builds the UI and `./klean`, then starts it |
| `dev` | The Go backend on `:8787` plus the Vite dev server with hot reload on <http://localhost:5173>, using a throwaway database in `.dev/`. `--mock` runs the UI alone against an in-memory mock backend; anything after `--` goes to Vite |
| `check` | Everything CI runs: `gofmt`, `oxlint`, `tsgo --noEmit`, the Vite build, `go vet` and `go test`. `--fix` reformats Go and fixes lint findings first. Run it before committing |
| `build` | One self-contained executable per platform in `dist/` (Linux, macOS and Windows; amd64 and arm64) |

Pushing a `v*` tag runs `.github/workflows/release.yml`, which tests, builds every platform and publishes the archives
on GitHub Releases.

## Layout

```
klean/
├── go.mod                    one Go module for the whole repo (so the UI can be embedded)
├── backend/                  Go: API, SQLite, calls to providers
│   ├── main.go               flags and startup
│   ├── internal/
│   │   ├── config/           every constant the backend uses
│   │   ├── server/           HTTP routes, sending messages, running tools (+ API tests)
│   │   ├── store/            SQLite schema, queries, and where the database lives
│   │   ├── llm/              the openai-go SDK: models, Responses and Chat Completions
│   │   ├── prompt/           which messages fit the context window, and titles
│   │   ├── tools/            tool definitions, web search, page fetching, artifact edits
│   │   ├── pdf/              report PDFs through an installed Chromium browser
│   │   ├── launch/           opens the browser, and a terminal window when double-clicked
│   │   └── fakeprovider/     a fake OpenAI-compatible server for tests
│   └── cmd/fakeprovider/     the same fake server as a program, for UI work
└── frontend/                 React + Vite + TypeScript, built into dist/ and embedded into the executable
    ├── embed.go              embeds dist/
    ├── tests/                plain Node tests (npm test)
    └── src/
        ├── config.ts         every timeout, size, limit, default, storage key and endpoint
        ├── api/              the only code that talks to the backend (+ mock/ for UI work alone)
        ├── store/            app state, one slice per concern (zustand)
        ├── components/       one folder per area, each with its own CSS
        ├── hooks/            small reusable behaviours
        ├── lib/              plain helpers: markdown, highlighting, tokens, theme…
        └── styles/           design tokens, base styles, fonts (self-hosted, works offline)
```

The backend's conventions are in [backend/AGENTS.md](../backend/AGENTS.md), the frontend's in
[frontend/AGENTS.md](../frontend/AGENTS.md).

## Frontend

The UI is built into `frontend/dist`, which the executable embeds, so build it before any `go build`, `go vet` or
`go test`: `cd frontend && npm ci && npm run build`.

| npm script | What it does |
|---|---|
| `npm run dev` | Vite with hot reload; `/api` goes to the backend on :8787 (`go run ./backend --no-browser`) |
| `npm run dev:mock` | The same, against the in-memory mock backend in `src/api/mock` |
| `npm run typecheck` | `tsgo --noEmit` |
| `npm run lint` | `oxlint` |
| `npm test` | The plain Node tests in `frontend/tests` |

To serve a built copy without rebuilding the executable: `go run ./backend --frontend frontend/dist`. To host the UI on
another origin, start the backend with `--frontend none` and build with
`VITE_KLEAN_API=http://localhost:8787 npm run build`.

Reports are untrusted HTML: they are sanitised with DOMPurify and only shown in a frame sandboxed without
`allow-scripts`, with a Content-Security-Policy that loads nothing from the network (`src/lib/report.ts`).

### The mock backend

In `npm run dev:mock`, these words in a message show each tool's UI: `/artifact`, `/code`, `/svg`, `/md`, `/report`,
`/update`, `/ask`, `/askmany`, `/search`, `/fetch`, `/fail`, `/skill <name>`. `NOIMAGES` makes the model refuse attached
images and `/notools` makes it refuse tools. In a chat with `/serverpdf`, **Download PDF** returns a PDF as the backend
would; with `/pdffail` the render fails and falls back to the in-page PDF.

## Tests

`./scripts/check.sh` runs everything. The Go tests alone: `go test ./...`, after building the UI.

The tests run against a fake provider that speaks the streaming Responses API. You can point the UI at it too:
`go run ./backend/cmd/fakeprovider 9100`, then use the base URL `http://127.0.0.1:9100/v1`. It answers every message
with the same canned reply, except for these words:

| Word | Effect |
|---|---|
| `/artifact`, `/code`, `/svg`, `/report` | Creates an HTML page, a Python file, an SVG image or a report |
| `/update` | Rewrites the most recent artifact in the chat |
| `/edit old => new` | Edits the most recent artifact, replacing `old` with `new` |
| `/ask` | Asks a question with three options |
| `/skill name` | Loads that skill |
| `/search words…` | Searches the web for the words after it |
| `/fetch URL` | Fetches that URL (`https://example.com/` if none is given) |
| `/loop` | Calls a tool on every request, to show the round limit |
| `NOTOOLS` | Rejects the request with a 400 if it carries tools |
| `SLOW`, `FAIL`, `MIDFAIL` | Streams slowly, answers 500, or fails partway through |

The tool words only act when the matching tool group is enabled.
