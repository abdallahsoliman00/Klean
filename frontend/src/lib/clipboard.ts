/** Copies text; false when the browser blocked it. */
export async function copyText(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch {
    return false;
  }
}

export const COPY_BLOCKED = 'Copy was blocked by the browser';
