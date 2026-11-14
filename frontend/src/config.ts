/* Every tunable number, key and fixed list the UI uses, in one place.
   Plain data only: vite.config.ts imports this file too. */

/* ---------- backend ---------- */

/** Where the Vite dev server forwards /api (the backend's default --port). */
export const DEV_BACKEND_URL = 'http://localhost:8787';

/** Backend origin. Empty means "same origin as the page", which is how the executable serves it.
    Set VITE_KLEAN_API at build time to host the UI elsewhere. */
export const API_ORIGIN: string = (import.meta.env?.VITE_KLEAN_API ?? '').replace(/\/+$/, '');

/** Every backend route lives under this prefix. */
export const API_PREFIX = '/api';

/** Endpoint paths, relative to API_PREFIX. */
export const ENDPOINTS = {
  providers: '/providers',
  provider: (id: string) => `/providers/${id}`,
  providerModels: (id: string) => `/providers/${id}/models`,
  chats: '/chats',
  chat: (id: string) => `/chats/${id}`,
  chatRestore: (id: string) => `/chats/${id}/restore`,
  chatMessages: (id: string) => `/chats/${id}/messages`,
  artifact: (id: string) => `/artifacts/${id}`,
  artifactPdf: (id: string) => `/artifacts/${id}/pdf`,
  image: (id: string) => `/images/${id}`,
  skills: '/skills',
  skill: (id: string) => `/skills/${id}`,
  settings: '/settings',
  presets: '/presets',
  preset: (id: string) => `/presets/${id}`,
} as const;

/** HTTP status the backend sends when an answered question is no longer open. */
export const STATUS_CONFLICT = 409;
/** HTTP status of a reply with no body. */
export const STATUS_NO_CONTENT = 204;
/** HTTP status for a request the backend understood but won't accept. */
export const STATUS_INVALID = 422;
/** HTTP status for "already gone". */
export const STATUS_NOT_FOUND = 404;
/** HTTP status when the backend can't make a report's PDF: no Chromium-based browser on this machine. */
export const STATUS_NOT_IMPLEMENTED = 501;
/** HTTP status when the backend's browser failed to make the PDF, or took too long. */
export const STATUS_BAD_GATEWAY = 502;

/* ---------- browser storage ---------- */

export const STORAGE_KEYS = {
  theme: 'sb:theme',
  scheme: 'sb:scheme',
  tools: 'sb:tools',
  artWidth: 'sb:artWidth',
} as const;

/* ---------- appearance ---------- */

export type ThemePref = 'system' | 'light' | 'dark';
export const THEME_PREFS: readonly ThemePref[] = ['system', 'light', 'dark'];
export const DEFAULT_THEME: ThemePref = 'system';

/** Colour schemes: id (matches [data-scheme] in styles/tokens.css), name, one-line description. */
export const SCHEMES = [
  { id: 'klean', name: 'Klean', note: 'Cool grey with a blue accent' },
  { id: 'graphite', name: 'Graphite', note: 'Neutral greys, ink for an accent' },
  { id: 'paper', name: 'Paper', note: 'Warm paper and amber' },
  { id: 'moss', name: 'Moss', note: 'Soft sage with a forest green accent' },
  { id: 'contrast', name: 'Contrast', note: 'Black and white, strong borders' },
] as const;
export type SchemeId = (typeof SCHEMES)[number]['id'];
export const DEFAULT_SCHEME: SchemeId = 'klean';

/** Breakpoints; keep in step with the @media rules in the CSS. */
export const ART_OVERLAY_QUERY = '(max-width:1100px)'; // artifact panel slides over the thread
export const MOBILE_QUERY = '(max-width:820px)'; // sidebar becomes a drawer
export const DARK_QUERY = '(prefers-color-scheme: dark)';

/* ---------- chat defaults ---------- */

/** Matches the backend's store.DefaultContextWindow. */
export const DEFAULT_CONTEXT_WINDOW = 1048576;
export const DEFAULT_TEMPERATURE = 0.7;
export const TEMPERATURE_RANGE = { min: 0, max: 2, step: 0.1 } as const;
export const CONTEXT_WINDOW_OPTIONS = [
  { value: 32768, label: '32k tokens' },
  { value: 65536, label: '64k tokens' },
  { value: 131072, label: '128k tokens' },
  { value: 262144, label: '256k tokens' },
  { value: 524288, label: '512k tokens' },
  { value: 1048576, label: '1M tokens' },
] as const;
/** What a chat starts with until the user has picked a configuration of their own. */
export const DEFAULT_CHAT_SETTINGS = { systemPrompt: '', temperature: DEFAULT_TEMPERATURE, contextWindow: DEFAULT_CONTEXT_WINDOW };

/* ---------- context estimate ---------- */

/** Rough token estimate: about this many characters per token. */
export const CHARS_PER_TOKEN = 4;
/** Per-message overhead (role, separators) added to each message's estimate. */
export const MESSAGE_OVERHEAD_TOKENS = 4;
/** Share of the context window history may use; the rest is left for the reply. */
export const HISTORY_BUDGET_SHARE = 0.75;
/** Above this share of the window the ring turns red. */
export const CONTEXT_WARN_SHARE = 0.9;
/** Token counts below this show one decimal ("1.2k"); from here on, none ("33k"). */
export const KFMT_DECIMALS_BELOW = 10000;
/** Circumference of the context ring (r = 7): 2 * PI * 7. */
export const CONTEXT_RING_LENGTH = 43.98;

/* ---------- timings (ms) ---------- */

export const THREAD_FADE_MS = 120; // fade-out before a different chat paints
export const COPY_FEEDBACK_MS = 1400; // "Copied" tick on copy buttons
export const TOAST_MS = 2600;
export const TOAST_WITH_ACTION_MS = 5000; // toasts with Undo stay longer
export const TOAST_EXIT_MS = 200; // matches .toast.out transition
export const SAVE_DEBOUNCE_MS = 300; // chat settings → backend
export const CHAT_PANEL_FOCUS_MS = 280; // after the panel slides in
export const PALETTE_FOCUS_MS = 30;
export const ART_FOCUS_MS = 60; // focus the artifact switcher after opening
export const ART_CLEAR_MS = 340; // empty the artifact view once the panel has slid out
export const DOWNLOAD_REVOKE_MS = 1000;

/* ---------- layout ---------- */

export const STICK_TO_BOTTOM_PX = 80; // closer than this to the bottom keeps following the stream
export const COMPOSER_MAX_HEIGHT_PX = 240;
export const POPOVER_EDGE_PX = 8; // min distance from the window edge
export const POPOVER_GAP_PX = 6; // between anchor and popover
export const MENU_GAP_PX = 4;
export const MENU_EDGE_RESERVE_PX = 180; // menu is kept this far from the right edge
/** Artifact panel width in px when dragged: its limits, the arrow-key step, and the largest share of the
    window it may take. Until it is dragged it uses --art from styles/tokens.css; keep `initial` in step with that. */
export const ART_WIDTH = { min: 380, max: 1400, step: 24, maxShare: 0.6, initial: { share: 0.4, max: 680 } } as const;

/* ---------- limits ---------- */

export const MAX_ASK_OPTIONS = 4;
export const MAX_HIGHLIGHT_CHARS = 300000; // bigger artifacts show as plain text
export const MAX_FILENAME_CHARS = 80;
/** Matches the backend's limit on a preset's name. */
export const PRESET_NAME_MAX_CHARS = 80;

/* ---------- images in messages (the backend enforces the same limits) ---------- */

export const IMAGE_TYPES: readonly string[] = ['image/png', 'image/jpeg', 'image/webp', 'image/gif'];
export const MAX_IMAGES_PER_MESSAGE = 8;
export const MAX_IMAGE_BYTES = 5 * 1024 * 1024;
/** Bigger images are shrunk in the browser before upload: the longest side in px, and the JPEG quality. */
export const IMAGE_DOWNSCALE = { maxSide: 2048, quality: 0.85 } as const;

/* ---------- skills (the backend enforces the same limits) ---------- */

export const SKILL_NAME_MAX_CHARS = 64;
export const SKILL_DESCRIPTION_MAX_CHARS = 1024;
export const SKILL_INSTRUCTIONS_MAX_CHARS = 100000;
/** Above this share of the description limit, the editor shows the count. */
export const SKILL_COUNT_FROM_SHARE = 0.8;
export const SKILL_FILE_MAX_BYTES = 1024 * 1024;
export const SKILL_FILE_ACCEPT = '.md,.markdown,text/markdown';

/** How the backend's advice for a model without tool support reads. When it comes back although the
    composer's tools are off, the tool in the request was use_skill, and the reply error says so. */
export const TOOLS_UNSUPPORTED_MARK = "doesn't appear to support tools";

/* ---------- message ids ---------- */

/** Id of the user message shown before the backend confirms it. */
export const PENDING_MESSAGE_ID = 'tmp';
/** Prefix for the id of a reply that was stopped before the backend finished it. */
export const LOCAL_MESSAGE_PREFIX = 'local-';

/* ---------- tools and search ---------- */

export type ToolGroup = 'artifacts' | 'ask' | 'web';
export const TOOL_GROUPS: readonly { key: ToolGroup; name: string; sub: string }[] = [
  { key: 'artifacts', name: 'Artifacts', sub: 'Pages, documents and code that open beside the chat' },
  { key: 'ask', name: 'Ask questions', sub: 'Let the model check with you instead of guessing' },
  { key: 'web', name: 'Web search', sub: 'Search the web and read pages' },
];

export const SEARCH_PROVIDERS = [
  { id: 'duckduckgo', name: 'DuckDuckGo', sub: 'Works straight away, no key needed' },
  { id: 'brave', name: 'Brave Search', sub: 'Needs an API key from brave.com/search/api' },
  { id: 'tavily', name: 'Tavily', sub: 'Needs an API key from tavily.com' },
] as const;
/** The search provider that needs no key, and the default. */
export const KEYLESS_SEARCH_PROVIDER = 'duckduckgo';

/* ---------- welcome screen ---------- */

export const STARTERS = [
  { title: 'Explain a concept', prompt: 'How does SQLite handle concurrent writes?' },
  { title: 'Write some code', prompt: 'Write a Python function that retries a request with backoff.' },
  { title: 'Think it through', prompt: 'Help me compare Postgres and SQLite for a small side project.' },
  { title: 'Tighten some text', prompt: 'Make this paragraph shorter and clearer: ' },
] as const;

/* ---------- artifacts ---------- */

export const ARTIFACT_KIND_LABELS: Record<string, string> = {
  html: 'HTML page',
  svg: 'SVG image',
  markdown: 'Document',
  code: 'Code',
  report: 'PDF report',
};

/** A report is a document to read and download as a PDF; its HTML is never shown. */
export const REPORT_KIND = 'report';
/** The tools whose result is a new artifact. */
export const ARTIFACT_CREATE_TOOLS: readonly string[] = ['create_artifact', 'create_report'];

/** highlight.js language for kinds that are not "code". */
export const ARTIFACT_KIND_LANGUAGE: Record<string, string> = { html: 'xml', svg: 'xml', markdown: 'markdown' };

/** Download extension for kinds that are not "code". */
export const ARTIFACT_KIND_EXTENSION: Record<string, string> = { html: 'html', svg: 'svg', markdown: 'md', report: 'pdf' };

/** Download extension by code language. */
export const LANGUAGE_EXTENSION: Record<string, string> = {
  python: 'py', py: 'py', javascript: 'js', js: 'js', typescript: 'ts', ts: 'ts', jsx: 'jsx', tsx: 'tsx',
  go: 'go', golang: 'go', rust: 'rs', c: 'c', cpp: 'cpp', 'c++': 'cpp', csharp: 'cs', 'c#': 'cs',
  java: 'java', kotlin: 'kt', swift: 'swift', ruby: 'rb', php: 'php', bash: 'sh', shell: 'sh', sh: 'sh',
  zsh: 'sh', powershell: 'ps1', sql: 'sql', json: 'json', yaml: 'yml', yml: 'yml', toml: 'toml',
  xml: 'xml', html: 'html', css: 'css', scss: 'scss', markdown: 'md', lua: 'lua', r: 'r', perl: 'pl',
  dart: 'dart', scala: 'scala', haskell: 'hs', elixir: 'ex', zig: 'zig',
};
export const FALLBACK_EXTENSION = 'txt';

/** Page wrapped around an SVG artifact in its preview frame. */
export const SVG_PREVIEW_HEAD =
  '<!doctype html><meta charset="utf-8"><style>html,body{height:100%;margin:0}body{display:grid;place-items:center;background:#fff;padding:24px;box-sizing:border-box}svg{max-width:100%;max-height:100%;height:auto}</style>';

/* ---------- reports ---------- */

export const CSS_PX_PER_MM = 96 / 25.4;
/** The page a report is laid out on, previewed on and printed to: A4 portrait. */
export const REPORT_PAGE = { widthMm: 210, heightMm: 297, marginMm: 14 } as const;
/** The preview's sheets: never drawn larger than `maxFit` times life size, `gutterPx` from the panel's edges. */
export const REPORT_PREVIEW = { maxFit: 1.25, gutterPx: 20 } as const;
export const PDF_TYPE = 'application/pdf';
export const REPORT_PDF = {
  /** Canvas pixels per CSS pixel: 2 keeps text sharp when the PDF is zoomed or printed. */
  scale: 2,
  image: { type: 'jpeg', quality: 0.98 },
  /** html2pdf.js's page-break modes: CSS break-before/after/inside, and its .html2pdf__page-break class. */
  pagebreakModes: ['css', 'legacy'],
  /** Never split across two pages, when they fit on one. */
  avoid: 'tr, p, li, h1, h2, h3, h4, h5, h6, img, svg, figure, blockquote, pre',
  /** Browsers refuse taller canvases; a longer report is rendered at a lower scale to fit. */
  maxCanvasPx: 32000,
  timeoutMs: 90000,
  /** How long to wait for the backend's PDF before making it in the browser instead: a little past its own limit (config.PDFRenderTimeout). */
  serverTimeoutMs: 35000,
} as const;
/** What a report's frame may load: nothing from the network, and no script. */
export const REPORT_CSP = "default-src 'none'; style-src 'unsafe-inline'; img-src data:; font-src data:";

/* ---------- sidebar date groups ---------- */

export const DAY_MS = 864e5;
export const WEEK_DAYS = 7; // "Previous 7 days"
export const MONTH_DAYS = 30; // "Previous 30 days"

/* ---------- mock backend (npm run dev:mock) ---------- */

export const MOCK_LATENCY = {
  quick: 40, // list chats
  read: 60, // most reads
  write: 120, // settings, answering
  models: [450, 400], // base + random extra
  firstToken: [400, 300],
  word: [14, 38],
  toolWrite: 350, // default time "writing" tool arguments
  toolRun: 650,
  title: 1500, // the summary title, after the reply is done
} as const;
