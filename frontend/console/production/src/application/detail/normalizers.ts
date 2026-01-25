//frontend\console\production\src\application\detail\normalizers.ts
// ---------- helpers ----------
export const asString = (v: any): string => (typeof v === "string" ? v : "");

export const asNonEmptyString = (v: any): string =>
  typeof v === "string" && v.trim() ? v.trim() : "";

/**
 * API から来る日時を Date に正規化する
 * - string (ISO) / number (ms) / Date / Firestore Timestamp っぽいもの を許容
 * - 変換できなければ null
 */
export const toDate = (v: any): Date | null => {
  if (!v) return null;

  // already Date
  if (v instanceof Date) {
    return Number.isNaN(v.getTime()) ? null : v;
  }

  // Firestore Timestamp shape (best-effort)
  // { seconds: number, nanoseconds: number } or { _seconds, _nanoseconds }
  const seconds =
    typeof v?.seconds === "number"
      ? v.seconds
      : typeof v?._seconds === "number"
        ? v._seconds
        : null;
  const nanos =
    typeof v?.nanoseconds === "number"
      ? v.nanoseconds
      : typeof v?._nanoseconds === "number"
        ? v._nanoseconds
        : 0;

  if (typeof seconds === "number") {
    const ms = seconds * 1000 + Math.floor((nanos ?? 0) / 1e6);
    const d = new Date(ms);
    return Number.isNaN(d.getTime()) ? null : d;
  }

  // number (ms)
  if (typeof v === "number") {
    const d = new Date(v);
    return Number.isNaN(d.getTime()) ? null : d;
  }

  // string
  if (typeof v === "string") {
    const d = new Date(v);
    return Number.isNaN(d.getTime()) ? null : d;
  }

  return null;
};
