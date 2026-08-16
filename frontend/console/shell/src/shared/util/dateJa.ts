// frontend/console/shell/src/shared/util/dateJa.ts

/**
 * 日時表示を "yyyy/MM/dd HH:mm" に統一する。
 *
 * 対応入力:
 * - ISO / RFC3339
 * - YYYY-MM-DD
 * - YYYY/MM/DD
 * - YYYY-MM-DD HH:mm
 * - YYYY/MM/DD HH:mm
 * - YYYY-MM-DD HH:mm:ss
 * - YYYY/MM/DD HH:mm:ss
 *
 * 秒は表示しない。
 * parse できない値は既存互換のため生文字列を返す。
 */
export function safeDateTimeLabelJa(
  value: string | null | undefined,
  fallback: string,
): string {
  const source =
    typeof value === "string"
      ? value.trim()
      : "";

  if (!source) {
    return fallback;
  }

  const direct = source.match(
    /^(\d{4})[-/](\d{1,2})[-/](\d{1,2})(?:[ T](\d{1,2}):(\d{1,2})(?::\d{1,2}(?:\.\d+)?)?)?$/,
  );

  if (direct) {
    const year = Number(direct[1]);
    const month = Number(direct[2]);
    const day = Number(direct[3]);
    const hour = Number(direct[4] ?? "0");
    const minute = Number(direct[5] ?? "0");

    if (
      Number.isFinite(year) &&
      Number.isFinite(month) &&
      Number.isFinite(day) &&
      Number.isFinite(hour) &&
      Number.isFinite(minute)
    ) {
      const pad2 = (number: number): string =>
        String(number).padStart(2, "0");

      return `${year}/${pad2(month)}/${pad2(day)} ${pad2(hour)}:${pad2(minute)}`;
    }
  }

  const timestamp = Date.parse(source);

  if (Number.isNaN(timestamp)) {
    return source;
  }

  const date = new Date(timestamp);
  const pad2 = (number: number): string =>
    String(number).padStart(2, "0");

  return `${date.getFullYear()}/${pad2(date.getMonth() + 1)}/${pad2(date.getDate())} ${pad2(date.getHours())}:${pad2(date.getMinutes())}`;
}