/* localStorage that never throws (private windows and blocked storage just forget). */

export function readString(key: string): string | null {
  try {
    return localStorage.getItem(key);
  } catch {
    return null;
  }
}

export function writeString(key: string, value: string) {
  try {
    localStorage.setItem(key, value);
  } catch {
    /* storage unavailable */
  }
}

export function readJson<T>(key: string): T | null {
  try {
    return JSON.parse(readString(key) ?? 'null') as T | null;
  } catch {
    return null;
  }
}

export const writeJson = (key: string, value: unknown) => writeString(key, JSON.stringify(value));
