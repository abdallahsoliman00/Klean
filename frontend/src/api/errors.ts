/** An error from the backend. It carries the HTTP status, so callers can tell "already gone" from a real failure. */
export class ApiError extends Error {
  readonly status?: number;
  constructor(message: string, status?: number) {
    super(message);
    this.status = status;
  }
}

export const OFFLINE_MESSAGE = "Can't reach the Klean backend. Check that it's running.";

/** Reads {"detail": ...} from a failed response, the shape the backend sends. */
export async function errorFromResponse(res: Response): Promise<ApiError> {
  return new ApiError(await errorText(res), res.status);
}

async function errorText(res: Response): Promise<string> {
  try {
    const j = (await res.json()) as { detail?: unknown };
    if (typeof j.detail === 'string') return j.detail;
    if (Array.isArray(j.detail)) return j.detail.map((d: { msg?: string }) => d.msg).join('; ');
  } catch {
    /* not JSON */
  }
  return `Request failed (${res.status}).`;
}

export const isAbort = (e: unknown) => (e as { name?: string } | null)?.name === 'AbortError';
