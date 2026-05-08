// frontend/console/mintRequest/src/infrastructure/repository/http/mintRequestManagementQuery.ts
//
// ✅ productionIds 前提
// ✅ view フォールバック（list -> management -> dto -> null）
// ✅ 既存HTTPユーティリティ整合（consoleApiBase / httpLogger）
//
// NOTE:
// - CORS を避けるには、API_BASE が「同一オリジンの rewrite 経由」になっている必要があります。
//   このファイルでは API_BASE を consoleApiBase の値に統一します。

import { API_BASE } from "../../../../../shell/src/shared/http/apiBase";
import { getAuthHeadersOrThrow } from "../../../../../shell/src/shared/http/authHeaders";
import {
  logHttpError,
  logHttpRequest,
  logHttpResponse,
  safeTokenHint,
} from "../../http/httpLogger";

import type { MintRequestManagementRowDTO } from "../../../application/dto/mintRequestManagementRow";
import { fetchProductionIdsForCurrentCompanyHTTP } from "./productions";

// ============================================================
// Types
// ============================================================

export type MintRequestManagementView = "list" | "management" | "dto" | null;

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

function buildPath(productionIds: string[], view: MintRequestManagementView): string {
  const ids = uniqNonEmptyStrings(productionIds ?? []);
  const base = `/mint/requests?productionIds=${encodeURIComponent(ids.join(","))}`;
  if (!view) return base;
  return `${base}&view=${encodeURIComponent(view)}`;
}

function normalizeItemsShape(
  payload: any,
): { items: MintRequestManagementRowDTO[]; shape: MintRequestManagementQueryResult["shape"] } | null {
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

function getAuthValueOrThrow(authHeaders: Record<string, string>): string {
  const authValue = String((authHeaders as any)?.Authorization ?? "").trim();
  if (!authValue) {
    throw new Error("Authorization header is missing (not logged in or token unavailable)");
  }
  return authValue;
}

function extractIdTokenForLog(authValue: string): string {
  const m = String(authValue ?? "").match(/^Bearer\s+(.+)$/i);
  return String(m?.[1] ?? "").trim();
}

async function requestJSON<T>(path: string): Promise<T> {
  const authHeaders = await getAuthHeadersOrThrow();
  const authValue = getAuthValueOrThrow(authHeaders);
  const idToken = extractIdTokenForLog(authValue);

  // API_BASE は consoleApiBase に統一（同一オリジンrewriteで CORS 回避する前提）
  const url = normalizeUrl(API_BASE, path);

  logHttpRequest("fetchMintRequestManagementRowsQueryHTTP", {
    method: "GET",
    url,
    path,
    headers: {
      Authorization: idToken ? `Bearer ${safeTokenHint(idToken)}` : safeTokenHint(authValue),
      "Content-Type": "application/json",
    },
  });

  const res = await fetch(url, { method: "GET", headers: authHeaders });

  logHttpResponse("fetchMintRequestManagementRowsQueryHTTP", {
    method: "GET",
    url,
    status: res.status,
    statusText: res.statusText,
  });

  const txt = await res.text().catch(() => "");

  // view フォールバックのため、404 は「未実装扱い」で上位が継続できるように throw する
  if (!res.ok) {
    logHttpError("fetchMintRequestManagementRowsQueryHTTP", {
      method: "GET",
      url,
      status: res.status,
      statusText: res.statusText,
      bodyPreview: txt ? txt.slice(0, 800) : "",
    });

    // 認可系はフォールバックしても意味が薄いので即エラー（原因特定しやすくする）
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
    logHttpError("fetchMintRequestManagementRowsQueryHTTP(parse)", {
      url,
      bodyPreview: txt.slice(0, 800),
    });
    throw new Error("MintRequestQueryService response is not JSON");
  }
}

// ============================================================
// Public API
// ============================================================

const DEFAULT_VIEWS: MintRequestManagementView[] = ["list", "management", "dto", null];

/**
 * MintRequestManagement 一覧取得（Query Service 直叩き）
 *
 * ✅ productionIds 前提:
 * - productionIds を引数で渡さない場合、/productions 系から company の productionIds を解決してから叩く
 *
 * ✅ view フォールバック:
 * - view=list -> management -> dto -> (viewなし) の順で試す
 *
 * ✅ shape 吸収:
 * - [] / {items:[]} / {rows:[]} / {data:[]} を吸収
 */
export async function fetchMintRequestManagementRowsQueryHTTP(
  args?: {
    productionIds?: string[];
    views?: MintRequestManagementView[];
  },
): Promise<MintRequestManagementQueryResult> {
  const views =
    (args?.views ?? DEFAULT_VIEWS).filter(
      (v) => v === "list" || v === "management" || v === "dto" || v === null,
    ) ?? DEFAULT_VIEWS;

  const productionIds = uniqNonEmptyStrings(args?.productionIds ?? []) ?? [];

  const effectiveProductionIds =
    productionIds.length > 0
      ? productionIds
      : uniqNonEmptyStrings(await fetchProductionIdsForCurrentCompanyHTTP());

  // productionIds 前提：0件なら backend を叩かずに空で返す
  if (effectiveProductionIds.length === 0) {
    const usedPath = buildPath([], views[0] ?? "list");
    return {
      items: [],
      usedPath,
      shape: "array",
      usedView: views[0] ?? "list",
      productionIdsCount: 0,
    };
  }

  let lastErr: any = null;

  for (const view of views) {
    const path = buildPath(effectiveProductionIds, view);

    try {
      const dto = await requestJSON<any>(path);

      const normalized = normalizeItemsShape(dto);
      if (!normalized) {
        // shape が想定外なら次の view へ（backend 実装差異に強くする）
        lastErr = new Error("unexpected response shape");
        continue;
      }

      return {
        items: normalized.items ?? [],
        usedPath: path,
        shape: normalized.shape,
        usedView: view,
        productionIdsCount: effectiveProductionIds.length,
      };
    } catch (e: any) {
      lastErr = e;
      // 次の view を試す（404/500/その他）
      continue;
    }
  }

  throw new Error(
    `MintRequestQueryService failed for all views. lastError=${
      lastErr?.message ?? String(lastErr ?? "")
    }`,
  );
}

/**
 * 互換 export（命名揺れ吸収）
 * - 既存 usecase 側が fetchMintRequestManagementRowsHTTP を期待していても落ちないようにする
 */
export const fetchMintRequestManagementRowsHTTP = fetchMintRequestManagementRowsQueryHTTP;
