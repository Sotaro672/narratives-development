// frontend/console/mintRequest/src/infrastructure/normalizers/string.ts

export function asTrimmedString(v: any): string {
  return typeof v === "string" ? v.trim() : String(v ?? "").trim();
}

export function asMaybeString(v: any): string | null {
  const s = asTrimmedString(v);
  return s ? s : null;
}
