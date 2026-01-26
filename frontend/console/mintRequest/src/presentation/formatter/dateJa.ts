// frontend/console/mintRequest/src/presentation/formatter/dateJa.ts

/**
 * 空/不正な日付でも落とさず、表示を "yyyy/mm/dd hh:mm:ss" / "yyyy/mm/dd" に固定するユーティリティ。
 * - 入力は ISO / RFC3339 / "YYYY-MM-DD" / "YYYY/MM/DD" / "YYYY/MM/DD HH:mm(:ss)" 等を想定
 * - parse できない場合は生文字返し（既存互換）
 */

export function asNonEmptyString(v: any): string {
  return typeof v === "string" && v.trim() ? v.trim() : "";
}

const pad2 = (n: number): string => String(n).padStart(2, "0");

const formatYMD = (y: number, m: number, d: number): string =>
  `${y}/${pad2(m)}/${pad2(d)}`;

const formatYMDHMS = (
  y: number,
  m: number,
  d: number,
  hh: number,
  mm: number,
  ss: number,
): string => `${formatYMD(y, m, d)} ${pad2(hh)}:${pad2(mm)}:${pad2(ss)}`;

/**
 * 文字列が "YYYY-MM-DD" / "YYYY/MM/DD" の場合は timezone 影響を避けるため、文字列から直接整形する
 */
function tryParseYMDFromString(s: string): { y: number; m: number; d: number } | null {
  const m = s.match(/^(\d{4})[-\/](\d{1,2})[-\/](\d{1,2})$/);
  if (!m) return null;
  const y = Number(m[1]);
  const mo = Number(m[2]);
  const d = Number(m[3]);
  if (!Number.isFinite(y) || !Number.isFinite(mo) || !Number.isFinite(d)) return null;
  return { y, m: mo, d };
}

/**
 * 文字列が "YYYY-MM-DD HH:mm(:ss)" / "YYYY/MM/DD HH:mm(:ss)" 等なら直接整形する
 */
function tryParseYMDHMSFromString(s: string): {
  y: number;
  m: number;
  d: number;
  hh: number;
  mm: number;
  ss: number;
} | null {
  const m = s.match(
    /^(\d{4})[-\/](\d{1,2})[-\/](\d{1,2})[ T](\d{1,2}):(\d{1,2})(?::(\d{1,2}))?$/,
  );
  if (!m) return null;

  const y = Number(m[1]);
  const mo = Number(m[2]);
  const d = Number(m[3]);
  const hh = Number(m[4]);
  const mm = Number(m[5]);
  const ss = Number(m[6] ?? "0");

  if (
    !Number.isFinite(y) ||
    !Number.isFinite(mo) ||
    !Number.isFinite(d) ||
    !Number.isFinite(hh) ||
    !Number.isFinite(mm) ||
    !Number.isFinite(ss)
  ) {
    return null;
  }

  return { y, m: mo, d, hh, mm, ss };
}

export function safeDateTimeLabelJa(
  v: string | null | undefined,
  fallback: string,
): string {
  const s = asNonEmptyString(v);
  if (!s) return fallback;

  // 1) "YYYY/MM/DD HH:mm(:ss)" 等はそのまま整形（環境非依存）
  const directDT = tryParseYMDHMSFromString(s);
  if (directDT) {
    return formatYMDHMS(
      directDT.y,
      directDT.m,
      directDT.d,
      directDT.hh,
      directDT.mm,
      directDT.ss,
    );
  }

  // 2) "YYYY-MM-DD" / "YYYY/MM/DD" は日付として整形（timezone 影響回避）
  const directD = tryParseYMDFromString(s);
  if (directD) {
    return formatYMDHMS(directD.y, directD.m, directD.d, 0, 0, 0);
  }

  // 3) ISO / RFC3339 等は Date.parse -> ローカル時刻で整形（表示形式は固定）
  const t = Date.parse(s);
  if (Number.isNaN(t)) return s;

  const dt = new Date(t);
  return formatYMDHMS(
    dt.getFullYear(),
    dt.getMonth() + 1,
    dt.getDate(),
    dt.getHours(),
    dt.getMinutes(),
    dt.getSeconds(),
  );
}

export function safeDateLabelJa(
  v: string | null | undefined,
  fallback: string,
): string {
  const s = asNonEmptyString(v);
  if (!s) return fallback;

  // 1) "YYYY-MM-DD" / "YYYY/MM/DD" は timezone 影響回避で直接整形
  const directD = tryParseYMDFromString(s);
  if (directD) return formatYMD(directD.y, directD.m, directD.d);

  // 2) "YYYY/MM/DD HH:mm(:ss)" 等は日付部分だけ使う
  const directDT = tryParseYMDHMSFromString(s);
  if (directDT) return formatYMD(directDT.y, directDT.m, directDT.d);

  // 3) ISO / RFC3339 等
  const t = Date.parse(s);
  if (Number.isNaN(t)) return s;

  const dt = new Date(t);
  return formatYMD(dt.getFullYear(), dt.getMonth() + 1, dt.getDate());
}
