// frontend/console/productBlueprint/src/presentation/util/dateFormat.ts

import { safeDateTimeLabelJa } from "../../../../shell/src/shared/util/dateJa";
import { formatProductBlueprintDate } from "../../infrastructure/api/productBlueprintApi";

/**
 * 日時を yyyy/MM/dd HH:mm に統一して返すユーティリティ。
 *
 * - Firestore Timestamp 等の { toDate(): Date } に対応
 * - ISO / RFC3339 / "YYYY-MM-DD" / "YYYY/MM/DD" / "YYYY/MM/DD HH:mm(:ss)" 等を許容
 * - safeDateTimeLabelJa で "yyyy/MM/dd HH:mm:ss" / "yyyy/MM/dd" に正規化した後、HH:mm までに丸める
 * - パース不能なら formatProductBlueprintDate(=従来) を試し、それでもダメなら生文字を返す（互換維持）
 */
export function formatDateTimeYYYYMMDDHHmm(value: unknown): string {
  if (value == null) return "";

  // Firestore Timestamp など
  try {
    if (typeof (value as any)?.toDate === "function") {
      const d: Date = (value as any).toDate();
      if (!Number.isNaN(d.getTime())) {
        return toYYYYMMDDHHmmOrFallback(d.toISOString(), value);
      }
    }
  } catch {
    // ignore
  }

  const s = String(value ?? "").trim();
  if (!s) return "";

  return toYYYYMMDDHHmmOrFallback(s, value);
}

// ------------------------------
// internal helpers
// ------------------------------

function toYYYYMMDDHHmmOrFallback(input: string, raw: unknown): string {
  const label = safeDateTimeLabelJa(input, "");
  const hhmm = extractYYYYMMDDHHmm(label);
  if (hhmm) return hhmm;

  const legacy = String(formatProductBlueprintDate(raw as any) ?? "").trim();
  if (legacy) return legacy;

  // 最終フォールバック：safeDateTimeLabelJa の結果 or 生文字
  return label || input;
}

/**
 * safeDateTimeLabelJa の出力（"yyyy/MM/dd HH:mm:ss" or "yyyy/MM/dd"）を
 * "yyyy/MM/dd HH:mm" に丸めて返す。
 */
function extractYYYYMMDDHHmm(label: string): string | null {
  const s = String(label ?? "").trim();
  if (!s) return null;

  // "yyyy/MM/dd HH:mm:ss" or "yyyy/MM/dd HH:mm"
  const m = s.match(/^(\d{4}\/\d{2}\/\d{2}) (\d{2}:\d{2})(?::\d{2})?$/);
  if (m) return `${m[1]} ${m[2]}`;

  // "yyyy/MM/dd"
  const m2 = s.match(/^(\d{4}\/\d{2}\/\d{2})$/);
  if (m2) return `${m2[1]} 00:00`;

  return null;
}
