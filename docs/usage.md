# Using Klean

## Running it

Run the executable. You can also build and run it from source with `./scripts/run.sh` (`scripts\run.bat` on Windows).

- **The browser opens** at the app's address. Use `--no-browser` to skip that.
- **Double-clicked in a file manager**, Klean opens its own terminal window showing the address and where the
  database is. **Closing that window stops Klean.** On Linux it looks for your desktop's terminal: `$TERMINAL`,
  `xdg-terminal-exec`, Konsole, Ptyxis, GNOME Terminal and others.
- **Started from a terminal**, it stays in that terminal, and Ctrl+C stops it.
- **Already running?** Starting it again just opens the browser at the running copy.
- If an error stops it in its own window, the window waits for Enter so you can read why.
- For a service or autostart entry, use `--headless`: no window and no browser.

## Providers

Open **Providers & settings** (bottom left), then **Add provider**. The base URL is the part before `/models` and
`/responses`.

Any provider that provides an OpenAI compatible endpoint works. Here are some of the most common ones:

| Provider   | Base URL                        |
|------------|---------------------------------|
| OpenRouter | `https://openrouter.ai/api/v1`  |
| OpenAI     | `https://api.openai.com/v1`     |
| Ollama     | `http://localhost:11434/v1`     |
| LM Studio  | `http://localhost:1234/v1`      |
| llama.cpp  | `http://localhost:8080/v1`      |

Models are loaded from `{base}/models`. Use **Reload** in the model picker (Ctrl+K) after pulling new ones. API keys are
kept in the database and never sent to the browser. The `OPENAI_*` environment variables are ignored: each provider
only gets the key saved for it.

Replies use the OpenAI **Responses API** (`POST {base}/responses`, streamed) with `store: false`, so the provider keeps
no copy. A provider that only has Chat Completions is used through `/chat/completions` instead.

## Options

| Flag           | Environment variable | Default |
|----------------|----------------------|---------|
| `--port`       | `KLEAN_PORT`         | `8787` |
| `--host`       | `KLEAN_HOST`         | `127.0.0.1` (this computer only) |
| `--db`         | `KLEAN_DB`           | the per-user data folder, below |
| `--frontend`   | `KLEAN_FRONTEND`     | the built-in UI; a folder path serves that instead, `none` serves the API only |
| `--no-browser` | `KLEAN_NO_BROWSER=1` | off |
| `--headless`   |                      | off (no browser, no terminal window) |
|                | `KLEAN_CORS`         | `*` |

Example: `./klean --port 9000 --db ~/chats.db`

## Where chats are stored

The database lives in a per-user folder rather than next to the executable, so the executable can be moved, renamed or
replaced without losing anything:

| OS      | Default database |
|---------|------------------|
| Linux   | `~/.local/share/klean/klean.db` (or `$XDG_DATA_HOME/klean/`) |
| macOS   | `~/Library/Application Support/Klean/klean.db` |
| Windows | `%LOCALAPPDATA%\Klean\klean.db` |


## Tools

The model can call tools while it answers. The composer switches three groups on and off:

| Group | Tool | What it does |
|---|---|---|
| `artifacts` | `create_artifact` | Puts a page, SVG, document or source file in the side panel |
| | `create_report` | Writes a printable report, shown as A4 pages and downloaded as a PDF |
| | `edit_artifact` | Changes part of an artifact by replacing exact passages, each one unique in the file |
| | `update_artifact` | Rewrites an artifact whole |
| `ask` | `ask_user` | Asks you a question with up to four options and waits for your answer |
| `web` | `web_search` | The top six results (title, URL, snippet) |
| | `fetch_page` | Downloads a page and returns its readable text |

`use_skill` is offered whenever you have enabled skills (Settings → Skills): instructions you saved for a kind of task.

- Every change to an artifact is a new version. Older ones stay in the panel's tabs, and the library (below **New chat**)
  lists the artifacts of every chat.
- **Web search** uses DuckDuckGo by default, which needs no key but sometimes turns automated requests away. For
  dependable search, pick Brave Search or Tavily in settings and save an API key for it.
- **Page fetching runs on your computer**, so it only fetches `http(s)` URLs on the public internet. Loopback,
  private-network and link-local addresses are refused, even when a public name or a redirect leads to one.
- **Report PDFs** are rendered with Chrome, Chromium, Edge or Brave when one is installed, so their text can be
  selected. Without one, the PDF is made in the page, and each page is an image.
- **Models without tool support:** if the provider rejects the tools, the error says so. Turn the tools off in the
  composer and send again.

## Context

Every request sends the chat's system prompt plus as many recent messages as fit in 75% of the chat's context window.
The rest of the window is left for the reply. Tokens are estimated at about four characters each, and tool calls,
attached files and the tool definitions count too. Older messages stay in the database and in the UI; they just aren't
sent. Set the window per chat from the ring in the top bar.

**Chat settings** (the sliders button in the top bar) holds **presets**: a saved model, system prompt, temperature and
context window. Whatever configuration you used last is what the next new chat starts with.
