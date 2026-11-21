# API

The UI only talks to the backend through these routes, under `/api`. Errors answer with `{detail}`.

| Method | Path | |
|---|---|---|
| GET | `/api/health` | liveness |
| GET/POST | `/api/providers` | list / add (`name`, `baseUrl`, `apiKey?`) |
| PATCH/DELETE | `/api/providers/{id}` | edit / remove (`apiKey: ""` clears the key) |
| GET | `/api/providers/{id}/models` | the provider's models |
| GET/POST | `/api/chats` | list / create |
| GET/PATCH/DELETE | `/api/chats/{id}` | read with `messages` and `artifacts` / update settings, title or favourite / delete |
| POST | `/api/chats/{id}/restore` | undo a delete (kept for 24 hours) |
| POST | `/api/chats/{id}/messages` | send; streams Server-Sent Events (below) |
| GET | `/api/chats/{id}/stream` | attach to the running reply: everything sent so far, then the rest. 204 when none is running |
| POST | `/api/chats/{id}/stop` | stop the running reply; answers once the partial reply is saved |
| GET | `/api/images/{id}` | an attached image's bytes |
| GET | `/api/artifacts` | the library: every chat's artifacts, without their content |
| GET | `/api/artifacts/{id}/versions` | every version of an artifact |
| DELETE | `/api/artifacts/{id}` | remove an artifact |
| POST | `/api/artifacts/{id}/pdf` | a report as a PDF, rendered with an installed Chromium browser |
| GET/POST | `/api/presets` | list / save (`name`, `providerId`, `model`, `systemPrompt`, `temperature`, `contextWindow`) |
| PATCH/DELETE | `/api/presets/{id}` | change any of those fields / delete |
| GET/POST | `/api/skills` | list / add (`name`, `description`, `instructions`, `enabled`) |
| PATCH/DELETE | `/api/skills/{id}` | change any of those fields / delete |
| GET/PATCH | `/api/settings` | `{searchProvider, hasSearchKey, chatDefaults}`; PATCH takes `searchProvider` (`duckduckgo`, `brave`, `tavily`), `searchKey` (`""` clears it; never returned) and `chatDefaults` (`null` clears it) |

## Sending a message

`POST /api/chats/{id}/messages` takes `{content, providerId, model}`, plus any of:

- `images`: `[{mediaType, data}]`, the bytes in base64. Up to 8 PNG, JPEG, WebP or GIF images of 5 MB each.
- `files`: `[{name, text}]`. Up to 8 text files of 512 KB each.
- `editMessageId` to replace a message and everything after it, or `regenerate: true` to replace the last reply. Both
  also delete the artifacts the removed replies created. An edited message keeps its images and files.
- `tools`: the enabled tool groups, a subset of `["artifacts", "ask", "web"]`. Left out, all are enabled; `[]` sends no
  tools.
- `answer: {callId, text}` to answer a pending `ask_user` question. No user message is created, and the model continues
  in a new assistant message. Sending an ordinary message instead closes the question as skipped.

A chat that already has a reply running answers 409. Closing the connection doesn't stop the reply: it keeps running
and is saved, and `/stream` attaches to it again. `/stop` stops it.

Events, in the order they can occur:

| Event | Data | |
|---|---|---|
| `user` | the saved user message | first, unless regenerating or answering |
| `answered` | `{messageId, part}` | first when the request had `answer`: the question's part, now done |
| `token` | `{delta}` | text; may come before, between and after tool calls |
| `tool_start` | `{id, name}` | the model began a tool call |
| `tool_run` | `{id, name, args}` | arguments complete, the tool is running |
| `artifact` | `{artifact}` | the full artifact after a create, edit or update, before its `tool_end` |
| `tool_end` | `{part}` | the finished tool part |
| `done` | `{message, title}` | the saved reply. If its last part is a pending `ask_user`, it is waiting for you |
| `error` | `{message}` | instead of `done` |
| `stopped` | `{message}` | instead of `done` when stopped; `null` when nothing had arrived |
| `title` | `{title}` | after `done`, when a summary title was written for the chat |

Assistant messages carry `parts`, the reply in the order it was produced: `{type: "text", text}` and
`{type: "tool", id, name, args, status: "pending"|"done"|"error", result, error?}`. `content` is the text parts joined.
`result` is `{artifactId, title, kind, language, version}` for the artifact tools, `{answer}` for `ask_user`,
`{name}` for `use_skill`, `{query, results: [{title, url, snippet}]}` for `web_search` and `{url, title, chars}` for
`fetch_page`. It is `null` while pending and on error.
