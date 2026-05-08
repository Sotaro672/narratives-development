//frontend\console\list\src\infrastructure\http\list\dates.ts
import { s } from "./string";

export function parseDateMs(v: unknown): number {
  const t = s(v);
  if (!t) return 0;
  const ms = Date.parse(t);
  if (!Number.isFinite(ms)) return 0;
  return ms;
}
