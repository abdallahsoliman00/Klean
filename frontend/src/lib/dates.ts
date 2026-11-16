import { DAY_MS, MONTH_DAYS, WEEK_DAYS } from '../config';

/** The sidebar group a chat falls in, by how many calendar days ago it was updated. */
export function groupOf(ts: number): string {
  const d = new Date(ts);
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  const diff = (today.getTime() - new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime()) / DAY_MS;
  if (diff <= 0) return 'Today';
  if (diff === 1) return 'Yesterday';
  if (diff < WEEK_DAYS) return 'Previous 7 days';
  if (diff < MONTH_DAYS) return 'Previous 30 days';
  return 'Older';
}

const UNITS = [['year', 365 * DAY_MS], ['month', MONTH_DAYS * DAY_MS], ['day', DAY_MS], ['hour', DAY_MS / 24], ['minute', DAY_MS / 1440]] as const;
const relative = new Intl.RelativeTimeFormat(undefined, { numeric: 'auto' });

/** "5 minutes ago", "yesterday", "last month". */
export function timeAgo(ts: number): string {
  const diff = ts - Date.now();
  const [unit, size] = UNITS.find(([, ms]) => Math.abs(diff) >= ms) ?? ['minute', 0];
  return size ? relative.format(Math.round(diff / size), unit) : 'just now';
}
