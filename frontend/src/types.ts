/* Shapes the backend sends and accepts (see backend/internal/store). */
import type { ToolGroup } from './config';

export interface Provider {
  id: string;
  name: string;
  baseUrl: string;
  hasKey: boolean;
}

export interface ProviderInput {
  name: string;
  baseUrl: string;
  apiKey?: string;
}

export interface Model {
  id: string;
  ownedBy: string;
}

/** Arguments a tool was called with. Each tool uses a few of these. */
export interface ToolArgs {
  query?: string;
  url?: string;
  id?: string;
  title?: string;
  kind?: string;
  language?: string;
  content?: string;
  edits?: { old_text: string; new_text: string }[];
  question?: string;
  multiple?: boolean;
  options?: { label: string; description?: string }[];
  name?: string;
}

export interface SearchResult {
  title?: string;
  url: string;
  snippet?: string;
}

/** What a tool returned. Each tool fills a few of these. */
export interface ToolResult {
  query?: string;
  results?: SearchResult[];
  url?: string;
  title?: string;
  chars?: number;
  artifactId?: string;
  kind?: string;
  language?: string;
  version?: number;
  answer?: string;
  name?: string;
}

export interface TextPart {
  type: 'text';
  text: string;
}

/** A tool call. No status means it is still running (only seen while streaming);
    "pending" is an ask_user question waiting for an answer. */
export interface ToolPart {
  type: 'tool';
  id: string;
  name: string;
  args: ToolArgs | null;
  result: ToolResult | null;
  status?: 'done' | 'error' | 'pending';
  error?: string;
}

export type Part = TextPart | ToolPart;

/** An image as it is uploaded with a message: base64 without the "data:" prefix. */
export interface ImageInput {
  mediaType: string;
  data: string;
}

/** A text file attached to a message, with its whole text. */
export interface FileInput {
  name: string;
  text: string;
}

/** An image a message carries. Its bytes are at GET /images/:id. */
export interface MessageImage {
  id: string;
  mediaType: string;
  /** Client-only: the local copy, shown until the server's has loaded. */
  url?: string;
}

export interface Message {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  images?: MessageImage[] | null;
  files?: FileInput[] | null;
  parts?: Part[] | null;
  model?: string | null;
  stopped?: boolean;
  createdAt?: number;
  /** Client-only React key that survives the pending id being replaced. */
  key?: string;
}

export interface ChatSettings {
  systemPrompt: string;
  temperature: number;
  contextWindow: number;
}

/** Everything a chat is configured with: the model and its settings. A preset saves exactly this. */
export interface ChatConfig extends ChatSettings {
  providerId: string | null;
  model: string | null;
}

/** A saved, named configuration. */
export interface Preset extends ChatConfig {
  id: string;
  name: string;
}

export type PresetInput = Omit<Preset, 'id'>;

/** Reusable instructions the assistant loads when a task calls for them (the shape of a SKILL.md file). */
export interface Skill {
  id: string;
  name: string;
  description: string;
  instructions: string;
  enabled: boolean;
}

export type SkillInput = Omit<Skill, 'id' | 'enabled'> & { enabled?: boolean };

export interface ChatSummary extends Partial<ChatSettings> {
  id: string;
  title: string;
  providerId: string | null;
  model: string | null;
  /** Starred: listed under Favourites at the top of the sidebar. */
  favorite: boolean;
  /** A reply is being written at the backend, whether or not this page is following it. */
  running: boolean;
  updatedAt: number;
}

export interface Artifact {
  id: string;
  chatId?: string;
  title: string;
  kind: string;
  language: string;
  content: string;
  version: number;
  createdAt: number;
  updatedAt: number;
}

/** One saved version of an artifact (GET /artifacts/:id/versions, oldest first). Numbers need not start at 1. */
export interface ArtifactVersion {
  version: number;
  title: string;
  content: string;
  createdAt: number;
}

/** A row of the artifacts library (GET /artifacts): an artifact without its content, and the chat it lives in. */
export interface ArtifactSummary extends Omit<Artifact, 'content' | 'chatId'> {
  chatId: string;
  chatTitle: string;
}

export interface Chat extends ChatSummary {
  systemPrompt: string;
  temperature: number;
  contextWindow: number;
  messages: Message[];
  artifacts: Artifact[];
}

export interface ChatCreate extends Partial<ChatSettings> {
  providerId: string | null;
  model: string | null;
}

export type ChatPatch = Partial<ChatSettings & { title: string; providerId: string | null; model: string | null; favorite: boolean }>;

export interface SearchSettings {
  searchProvider: string;
  hasSearchKey: boolean;
}

export interface SearchSettingsPatch {
  searchProvider?: string;
  searchKey?: string;
}

/** GET /settings: the search settings, plus the configuration new chats start with (null until one is saved). */
export interface AppSettings extends SearchSettings {
  chatDefaults: ChatConfig | null;
}

export interface SettingsPatch extends SearchSettingsPatch {
  /** Replaces the saved configuration whole; null clears it. */
  chatDefaults?: ChatConfig | null;
}

/** Body of POST /chats/:id/messages. */
export interface SendBody {
  content: string;
  images?: ImageInput[];
  files?: FileInput[];
  providerId: string | null;
  model: string | null;
  editMessageId?: string;
  regenerate?: boolean;
  tools: ToolGroup[];
  answer?: { callId: string; text: string };
}

export interface SendCallbacks {
  /** Aborting drops this connection only: the reply carries on at the backend (chats.stop ends it). */
  signal?: AbortSignal;
  onUser?: (m: Message) => void;
  onAnswered?: (a: { messageId: string; part: ToolPart }) => void;
  onToken?: (delta: string) => void;
  onToolStart?: (t: { id: string; name: string }) => void;
  onToolRun?: (t: { id: string; name: string; args: ToolArgs }) => void;
  onToolEnd?: (part: ToolPart) => void;
  onArtifact?: (a: Artifact) => void;
  /** A summary title, which may arrive after the reply has finished. */
  onTitle?: (title: string) => void;
}

export interface SendResult {
  message: Message;
  title: string | null;
}

/** Whether a newer Klean release exists (GET /api/update). */
export interface UpdateInfo {
  current: string;
  latest: string;
  available: boolean;
  page: string;
}

/** Everything the UI can ask of the backend. Implemented over HTTP (api/http) and in memory (api/mock). */
export interface Api {
  providers: {
    list(): Promise<Provider[]>;
    create(data: ProviderInput): Promise<Provider>;
    update(id: string, data: ProviderInput): Promise<Provider>;
    remove(id: string): Promise<void>;
    models(id: string): Promise<Model[]>;
  };
  chats: {
    list(): Promise<ChatSummary[]>;
    get(id: string): Promise<Chat | null>;
    create(data: ChatCreate): Promise<Chat>;
    update(id: string, patch: ChatPatch): Promise<unknown>;
    remove(id: string): Promise<void>;
    restore(id: string): Promise<void>;
    /** Starts a reply and follows it to its end: finished, or stopped (the message then has `stopped`, and is empty when nothing was written). */
    send(chatId: string, body: SendBody, cb?: SendCallbacks): Promise<SendResult>;
    /** Follows the reply a chat is already writing, from its start. Null when none is being written. */
    attach(chatId: string, cb?: SendCallbacks): Promise<SendResult | null>;
    /** Stops the reply being written; what there is of it is saved. */
    stop(chatId: string): Promise<void>;
  };
  artifacts: {
    /** Every artifact in every chat, newest first. */
    list(): Promise<ArtifactSummary[]>;
    versions(id: string): Promise<ArtifactVersion[]>;
    remove(id: string): Promise<void>;
    /** A report as a PDF with real text, rendered by a browser on the backend's machine. Rejects with
        status 501 when there is none; the caller then makes the PDF itself (lib/reportDownload). */
    pdf(id: string): Promise<Blob>;
  };
  presets: {
    list(): Promise<Preset[]>;
    create(data: PresetInput): Promise<Preset>;
    update(id: string, patch: Partial<PresetInput>): Promise<Preset>;
    remove(id: string): Promise<void>;
  };
  settings: {
    get(): Promise<AppSettings>;
    update(patch: SettingsPatch): Promise<AppSettings>;
  };
  skills: {
    list(): Promise<Skill[]>;
    create(data: SkillInput): Promise<Skill>;
    update(id: string, patch: Partial<SkillInput>): Promise<Skill>;
    remove(id: string): Promise<void>;
  };
  update: {
    check(): Promise<UpdateInfo>;
    /** Downloads the latest release and puts it in place of the running executable; it runs from the next start. */
    apply(): Promise<{ version: string }>;
  };
  /** The running backend's version: "0.0.5", or "dev" for a build without one. */
  version(): Promise<string>;
  /** Where an image's bytes are, for an <img>. */
  imageUrl(id: string): string;
}
