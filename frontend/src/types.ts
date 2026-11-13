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
  question?: string;
  multiple?: boolean;
  options?: { label: string; description?: string }[];
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

export interface Message {
  id: string;
  role: 'user' | 'assistant';
  content: string;
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

export interface ChatSummary extends Partial<ChatSettings> {
  id: string;
  title: string;
  providerId: string | null;
  model: string | null;
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

export type ChatPatch = Partial<ChatSettings & { title: string; providerId: string | null; model: string | null }>;

export interface SearchSettings {
  searchProvider: string;
  hasSearchKey: boolean;
}

export interface SearchSettingsPatch {
  searchProvider?: string;
  searchKey?: string;
}

/** Body of POST /chats/:id/messages. */
export interface SendBody {
  content: string;
  providerId: string | null;
  model: string | null;
  editMessageId?: string;
  regenerate?: boolean;
  tools: ToolGroup[];
  answer?: { callId: string; text: string };
}

export interface SendCallbacks {
  signal?: AbortSignal;
  onUser?: (m: Message) => void;
  onAnswered?: (a: { messageId: string; part: ToolPart }) => void;
  onToken?: (delta: string) => void;
  onToolStart?: (t: { id: string; name: string }) => void;
  onToolRun?: (t: { id: string; name: string; args: ToolArgs }) => void;
  onToolEnd?: (part: ToolPart) => void;
  onArtifact?: (a: Artifact) => void;
}

export interface SendResult {
  message: Message;
  title: string | null;
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
    send(chatId: string, body: SendBody, cb?: SendCallbacks): Promise<SendResult>;
  };
  artifacts: {
    remove(id: string): Promise<void>;
  };
  settings: {
    get(): Promise<SearchSettings>;
    update(patch: SearchSettingsPatch): Promise<SearchSettings>;
  };
}
