// frontend/console/inventory/src/infrastructure/http/inventoryRepositoryHTTP.utils.ts

export function s(v: unknown): string {
  return String(v ?? "").trim();
}

export function n(v: unknown): number {
  const x = Number(v ?? 0);
  return Number.isFinite(x) ? x : 0;
}

export function toOptionalString(v: any): string | undefined {
  const x = s(v);
  return x ? x : undefined;
}

export function toRgbNumberOrNull(v: any): number | null | undefined {
  if (v === undefined) return undefined;
  if (v === null) return null;

  if (typeof v === "number" && Number.isFinite(v)) return v;

  const str = s(v);
  if (!str) return null;

  const normalized = str.replace(/^#/, "").replace(/^0x/i, "");
  if (/^[0-9a-fA-F]{6}$/.test(normalized)) {
    const nn = Number.parseInt(normalized, 16);
    return Number.isFinite(nn) ? nn : null;
  }

  const d = Number.parseInt(str, 10);
  return Number.isFinite(d) ? d : null;
}

// ✅ productIdTag を "QRコード" のような表示用文字列に正規化する
export function toProductIdTagString(v: any): string | null | undefined {
  if (v === undefined) return undefined;
  if (v === null) return null;

  // 既に文字列
  if (typeof v === "string") {
    const str = s(v);
    if (!str) return null;

    // JSON文字列っぽい場合は parse してから抽出を試す
    const looksJson = str.startsWith("{") && str.endsWith("}");
    if (looksJson) {
      try {
        const obj = JSON.parse(str);
        const fromObj = toProductIdTagString(obj);
        return fromObj ?? str;
      } catch {
        return str;
      }
    }
    return str;
  }

  // 数値/boolean 等
  if (typeof v === "number" || typeof v === "boolean") {
    return String(v);
  }

  // オブジェクト: { Type: "QRコード" } など
  if (typeof v === "object") {
    const o: any = v;

    const candidates = [
      o?.label,
      o?.Label,
      o?.type,
      o?.Type,
      o?.value,
      o?.Value,
      o?.name,
      o?.Name,
    ];

    for (const c of candidates) {
      const str = s(c);
      if (str) return str;
    }

    // どうしても取れない場合は null（"[object Object]" を出さない）
    return null;
  }

  return null;
}
