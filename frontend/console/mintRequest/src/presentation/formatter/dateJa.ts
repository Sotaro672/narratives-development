// frontend/console/mintRequest/src/presentation/formatter/dateJa.ts

/**
 * 空/不正な日付でも落とさず、日本語ロケールで表示するためのユーティリティ。
 * - 入力は ISO / RFC3339 / "YYYY-MM-DD" / "YYYY/MM/DD" 等を想定（parse できない場合は生文字返し）
 */

export function asNonEmptyString(v: any): string {
  return typeof v === "string" && v.trim() ? v.trim() : "";
}

export function safeDateTimeLabelJa(
  v: string | null | undefined,
  fallback: string,
): string {
  const s = asNonEmptyString(v);
  if (!s) return fallback;

  const t = Date.parse(s);
  if (Number.isNaN(t)) return s; // 解析不可なら生文字

  return new Date(t).toLocaleString("ja-JP");
}

export function safeDateLabelJa(
  v: string | null | undefined,
  fallback: string,
): string {
  const s = asNonEmptyString(v);
  if (!s) return fallback;

  const t = Date.parse(s);
  if (Number.isNaN(t)) {
    // "YYYY-MM-DD" 等はそのまま出したいケースがあるので生文字
    return s;
  }

  return new Date(t).toLocaleDateString("ja-JP");
}
