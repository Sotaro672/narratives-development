// frontend/console/mintRequest/src/infrastructure/repository/http/mintRequestManagementQuery.ts
import { API_BASE } from "../../../../../shell/src/shared/http/apiBase";
import { getAuthHeadersOrThrow } from "../../../../../shell/src/shared/http/authHeaders";

import type { MintRequestManagementRowDTO } from "../../../application/dto/mintRequestManagementRow";
import { fetchProductionIdsForCurrentCompanyHTTP } from "./productions";

// ============================================================
// Types
// ============================================================

export type MintRequestManagementView = "list";

export type MintRequestManagementQueryResult = {
  items: MintRequestManagementRowDTO[];
  usedPath: string; // 実際に成功した path（querystring 含む）
  shape: "array" | "items" | "rows" | "data";
  usedView: MintRequestManagementView;
  productionIdsCount: number;
};

// ============================================================
// helpers
// ============================================================

function uniqNonEmptyStrings(xs: string[]): string[] {
  const out: string[] = [];
  const seen = new Set<string>();

  for (const x of xs ?? []) {
    const s = String(x ?? "").trim();
    if (!s) continue;
    if (seen.has(s)) continue;

    seen.add(s);
    out.push(s);
  }

  return out;
}

function buildPath(productionIds: string[]): string {
  const ids = uniqNonEmptyStrings(productionIds ?? []);
  return `/mint/requests?productionIds=${encodeURIComponent(
    ids.join(","),
  )}&view=list`;
}

function normalizeItemsShape(
  payload: any,
): {
  items: MintRequestManagementRowDTO[];
  shape: MintRequestManagementQueryResult["shape"];
} | null {
  // 1) array
  if (Array.isArray(payload)) {
    return { items: payload as MintRequestManagementRowDTO[], shape: "array" };
  }

  // 2) { items: [] }
  if (payload && Array.isArray((payload as any).items)) {
    return {
      items: (payload as any).items as MintRequestManagementRowDTO[],
      shape: "items",
    };
  }

  // 3) { rows: [] }
  if (payload && Array.isArray((payload as any).rows)) {
    return {
      items: (payload as any).rows as MintRequestManagementRowDTO[],
      shape: "rows",
    };
  }

  // 4) { data: [] }
  if (payload && Array.isArray((payload as any).data)) {
    return {
      items: (payload as any).data as MintRequestManagementRowDTO[],
      shape: "data",
    };
  }

  return null;
}

function normalizeUrl(base: string, path: string): string {
  const b = String(base ?? "").replace(/\/+$/g, "");
  const p = String(path ?? "");

  if (!b) return p.startsWith("/") ? p : `/${p}`;

  return `${b}${p.startsWith("/") ? p : `/${p}`}`;
}

async function requestJSON<T>(path: string): Promise<T> {
  const authHeaders = await getAuthHeadersOrThrow();

  // API_BASE は consoleApiBase に統一（同一オリジンrewriteで CORS 回避する前提）
  const url = normalizeUrl(API_BASE, path);

  const res = await fetch(url, { method: "GET", headers: authHeaders });

  const txt = await res.text().catch(() => "");

  if (!res.ok) {
    if (res.status === 401 || res.status === 403) {
      throw new Error(
        `MintRequestQueryService auth error: ${res.status} ${res.statusText}${
          txt ? `\n${txt}` : ""
        }`,
      );
    }

    throw new Error(
      `MintRequestQueryService error: ${res.status} ${res.statusText}${
        txt ? `\n${txt}` : ""
      }`,
    );
  }

  // 空レスポンスは許容しない（呼び出し側が shape 判定できないため）
  if (!txt.trim()) {
    throw new Error("MintRequestQueryService response is empty");
  }

  try {
    return JSON.parse(txt) as T;
  } catch (_e) {
    throw new Error("MintRequestQueryService response is not JSON");
  }
}

// ============================================================
// Public API
// ============================================================

/**
 * MintRequestManagement 一覧取得（Query Service 直叩き）
 *
 * ✅ productionIds 前提:
 * - productionIds を引数で渡さない場合、/productions 系から company の productionIds を解決してから叩く
 *
 * ✅ fallback なし:
 * - view=list のみ使用する
 *
 * ✅ shape 吸収:
 * - [] / {items:[]} / {rows:[]} / {data:[]} を吸収
 */
export async function fetchMintRequestManagementRowsQueryHTTP(
  args?: {
    productionIds?: string[];
  },
): Promise<MintRequestManagementQueryResult> {
  const productionIds = uniqNonEmptyStrings(args?.productionIds ?? []) ?? [];

  const effectiveProductionIds =
    productionIds.length > 0
      ? productionIds
      : uniqNonEmptyStrings(await fetchProductionIdsForCurrentCompanyHTTP());

  const usedPath = buildPath(effectiveProductionIds);

  // productionIds 前提：0件なら backend を叩かずに空で返す
  if (effectiveProductionIds.length === 0) {
    return {
      items: [],
      usedPath,
      shape: "array",
      usedView: "list",
      productionIdsCount: 0,
    };
  }

  const dto = await requestJSON<any>(usedPath);

  const normalized = normalizeItemsShape(dto);

  if (!normalized) {
    throw new Error("MintRequestQueryService response has unexpected shape");
  }

  return {
    items: normalized.items ?? [],
    usedPath,
    shape: normalized.shape,
    usedView: "list",
    productionIdsCount: effectiveProductionIds.length,
  };
}