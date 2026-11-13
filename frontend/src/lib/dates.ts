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
