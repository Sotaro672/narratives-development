// frontend/console/mintRequest/src/infrastructure/repository/http/mintRequests.ts

import { API_BASE } from "../../http/consoleApiBase";
import { getIdTokenOrThrow } from "../../http/firebaseAuth";
import { buildHeaders } from "../../http/httpClient";
import {
  logHttpError,
  logHttpRequest,
  logHttpResponse,
  safeTokenHint,
} from "../../http/httpLogger";

import type {
  InspectionBatchDTO,
  MintDTO,
  MintListRowDTO,
} from "../../api/mintRequestApi";

import type {
  MintRequestRowRaw,
  MintRequestsPayloadRaw,
} from "../../dto/mintRequestRaw.dto";

import { normalizeMintDTO } from "../../normalizers/mint";
import { normalizeMintListRow } from "../../normalizers/mintListRow";
import {
  extractRowKeyAsProductionId,
  normalizeMintRequestsRows,
} from "../../normalizers/mintRequestsRows";

// ===============================
// types
// ===============================

// ✅ "dto" view は今回の不具合原因になりうるため、フロント側からは使用しない前提で削除remember
type MintRequestsView = "management" | "list";
type MintRequestsViewOrNull = MintRequestsView | null;

type FetchMintRequestsResult = {
  rows: MintRequestRowRaw[];
  usedView: MintRequestsViewOrNull;
  usedUrl: string;
};

// ===============================
// helpers
// ===============================

function uniqTrimmedStrings(xs: string[]): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const x of xs ?? []) {
    const s = String(x ?? "").trim();
    if (!s) continue;
    if (seen.has(s)) continue;
    seen.add(s);
    out.push(s);
  }
  return out;
}

function buildMintRequestsUrl(ids: string[], view: MintRequestsViewOrNull): string {
  const base = `${API_BASE}/mint/requests?productionIds=${encodeURIComponent(
    ids.join(","),
  )}`;
  return view ? `${base}&view=${encodeURIComponent(view)}` : base;
}

function isLikelyMissingRouteStatus(status: number): boolean {
  // view 未対応などの「ルート/パラメータ非対応」を “候補切替” のトリガにする
  return status === 404 || status === 405;
}

function isServiceUnavailableStatus(status: number): boolean {
  return status === 502 || status === 503 || status === 504;
}

async function readTextSafe(res: Response): Promise<string> {
  try {
    return await res.text();
  } catch {
    return "";
  }
}

/**
 * Raw fetch for a single view.
 * - 404/405 は「その view/ルートが無い」扱いにして上位でフォールバックさせる
 * - それ以外の non-2xx はエラー（ログ付き）
 */
async function fetchMintRequestsRowsRawOnce(
  ids: string[],
  view: MintRequestsViewOrNull,
): Promise<FetchMintRequestsResult> {
  const safeIds = uniqTrimmedStrings(ids ?? []);
  if (safeIds.length === 0) {
    return { rows: [], usedView: view, usedUrl: "" };
  }

  const idToken = await getIdTokenOrThrow();
  const url = buildMintRequestsUrl(safeIds, view);

  logHttpRequest("fetchMintRequestsRowsRawOnce", {
    method: "GET",
    url,
    headers: {
      Authorization: `Bearer ${safeTokenHint(idToken)}`,
      "Content-Type": "application/json",
    },
    productionIds: safeIds,
    view,
  });

  let res: Response;
  try {
    res = await fetch(url, { method: "GET", headers: buildHeaders(idToken) });
  } catch (e: any) {
    // fetch 自体が落ちる (CORS / network / DNS etc.)
    logHttpError("fetchMintRequestsRowsRawOnce(fetch)", {
      method: "GET",
      url,
      view,
      productionIds: safeIds,
      error: String(e?.message ?? e),
    });
    throw new Error(
      `Failed to fetch mint requests (network): ${String(e?.message ?? e)}`,
    );
  }

  logHttpResponse("fetchMintRequestsRowsRawOnce", {
    method: "GET",
    url,
    status: res.status,
    statusText: res.statusText,
  });

  // view/route missing -> caller should fallback to next view
  if (isLikelyMissingRouteStatus(res.status)) {
    const body = await readTextSafe(res);
    logHttpError("fetchMintRequestsRowsRawOnce(view missing -> fallback)", {
      method: "GET",
      url,
      status: res.status,
      statusText: res.statusText,
      view,
      bodyPreview: body ? body.slice(0, 300) : "",
    });

    const err: any = new Error(
      `Mint requests view not supported: ${view ?? "null"} (${res.status})`,
    );
    err.__mint_requests_view_missing__ = true;
    err.status = res.status;
    err.view = view;
    err.url = url;
    throw err;
  }

  // 200-299 OK
  if (res.ok) {
    const json = (await res.json()) as MintRequestsPayloadRaw | null | undefined;
    return {
      rows: normalizeMintRequestsRows(json),
      usedView: view,
      usedUrl: url,
    };
  }

  // non-ok (not route-missing)
  const body = await readTextSafe(res);
  logHttpError("fetchMintRequestsRowsRawOnce(non-ok)", {
    method: "GET",
    url,
    status: res.status,
    statusText: res.statusText,
    view,
    bodyPreview: body ? body.slice(0, 800) : "",
  });

  const hint = isServiceUnavailableStatus(res.status)
    ? " (service unavailable)"
    : "";

  throw new Error(
    `Failed to fetch mint requests${hint}: ${res.status} ${res.statusText}${
      body ? ` body=${body.slice(0, 400)}` : ""
    }`,
  );
}

function isViewMissingError(e: any): boolean {
  return Boolean(
    e &&
      (e.__mint_requests_view_missing__ === true ||
        e.status === 404 ||
        e.status === 405),
  );
}

/**
 * Fetch with view fallbacks.
 * - list route: list -> management -> null
 * - management route: management -> null
 */
async function fetchMintRequestsRowsRawWithFallback(
  ids: string[],
  views: MintRequestsViewOrNull[],
): Promise<FetchMintRequestsResult> {
  const safeIds = uniqTrimmedStrings(ids ?? []);
  if (safeIds.length === 0) {
    return { rows: [], usedView: null, usedUrl: "" };
  }

  const candidates: MintRequestsViewOrNull[] = (views ?? []).filter(
    (v, i, arr) => {
      // dedupe incl. null
      return arr.indexOf(v) === i;
    },
  );

  let lastErr: any = null;

  for (const view of candidates) {
    try {
      return await fetchMintRequestsRowsRawOnce(safeIds, view);
    } catch (e: any) {
      lastErr = e;

      // view missing -> try next
      if (isViewMissingError(e)) continue;

      // network / 5xx etc. -> propagate (do not hide)
      throw e;
    }
  }

  // only reaches when all candidates were "view missing"
  throw new Error(
    `Mint requests endpoint does not support requested views. tried=${candidates
      .map((v) => (v === null ? "null" : v))
      .join(",")} lastError=${String(lastErr?.message ?? lastErr ?? "")}`,
  );
}

// ===============================
// Public exports (一覧/参照系)
// ===============================

/**
 * inspectionId (= productionId) で 1 件の MintDTO を取得。
 * ✅ 優先: view=management -> null
 * - management は createdByName/requestedByName 等の “表示名” を持つ可能性が高い
 */
export async function fetchMintByInspectionIdHTTP(
  inspectionId: string,
): Promise<MintDTO | null> {
  const iid = String(inspectionId ?? "").trim();
  if (!iid) throw new Error("inspectionId が空です");

  // ✅ view fallback（management 優先）
  const { rows } = await fetchMintRequestsRowsRawWithFallback([iid], [
    "management",
    null,
  ]);

  const row =
    (rows ?? []).find((r) => extractRowKeyAsProductionId(r) === iid) ??
    rows?.[0] ??
    null;

  if (!row) return null;

  const mintRaw = (row as any)?.mint ?? (row as any)?.Mint ?? null;

  // ✅ row 側の display fields を mint 側に引き継ぐ
  const merged = {
    ...(mintRaw ?? row ?? {}),
    createdByName:
      (row as any)?.requestedByName ??
      (row as any)?.createdByName ??
      (mintRaw as any)?.createdByName ??
      null,
    requestedByName: (row as any)?.requestedByName ?? null,
  };

  return normalizeMintDTO(merged);
}

/**
 * inspectionIds (= productionIds) で MintDTO を map で取得。
 * ✅ 優先: view=management -> null
 */
export async function fetchMintsByInspectionIdsHTTP(
  inspectionIds: string[],
): Promise<Record<string, MintDTO>> {
  const ids = uniqTrimmedStrings(inspectionIds ?? []);
  if (ids.length === 0) return {};

  const { rows } = await fetchMintRequestsRowsRawWithFallback(ids, [
    "management",
    null,
  ]);

  const out: Record<string, MintDTO> = {};
  for (const r of rows ?? []) {
    const key = extractRowKeyAsProductionId(r);
    if (!key) continue;

    const mintRaw = (r as any)?.mint ?? (r as any)?.Mint ?? null;

    const merged = {
      ...(mintRaw ?? r ?? {}),
      createdByName:
        (r as any)?.requestedByName ??
        (r as any)?.createdByName ??
        (mintRaw as any)?.createdByName ??
        null,
      requestedByName: (r as any)?.requestedByName ?? null,
    };

    out[key] = normalizeMintDTO(merged);
  }

  return out;
}

/**
 * inspectionIds (= productionIds) で “一覧用” MintListRowDTO を map で取得。
 * 優先: view=list -> management -> null
 */
export async function fetchMintListRowsByInspectionIdsHTTP(
  inspectionIds: string[],
): Promise<Record<string, MintListRowDTO>> {
  const ids = uniqTrimmedStrings(inspectionIds ?? []);
  if (ids.length === 0) return {};

  const { rows } = await fetchMintRequestsRowsRawWithFallback(ids, [
    "list",
    "management",
    null,
  ]);

  const out: Record<string, MintListRowDTO> = {};
  for (const r of rows ?? []) {
    const key = extractRowKeyAsProductionId(r);
    if (!key) continue;

    const base =
      (r as any)?.mint ?? (r as any)?.Mint ?? null
        ? (r as any)?.mint ?? (r as any)?.Mint
        : (r as any);

    const merged = {
      ...(base ?? {}),
      inspectionId: key,
      productionId: key,
      tokenName: (r as any)?.tokenName ?? (base as any)?.tokenName ?? null,
      createdByName: (r as any)?.createdByName ?? (base as any)?.createdByName ?? null,
      mintedAt: (r as any)?.mintedAt ?? (base as any)?.mintedAt ?? null,
      minted:
        typeof (r as any)?.minted === "boolean"
          ? (r as any).minted
          : (base as any)?.minted,
    };

    out[key] = normalizeMintListRow(merged);
  }

  return out;
}

// ===============================
// POST: mint request
// ===============================

export async function postMintRequestHTTP(
  productionId: string,
  tokenBlueprintId: string,
  scheduledBurnDate?: string,
): Promise<InspectionBatchDTO | null> {
  const trimmed = String(productionId ?? "").trim();
  if (!trimmed) throw new Error("productionId が空です");

  const idToken = await getIdTokenOrThrow();

  const url = `${API_BASE}/mint/inspections/${encodeURIComponent(
    trimmed,
  )}/request`;

  const payload: { tokenBlueprintId: string; scheduledBurnDate?: string } = {
    tokenBlueprintId: String(tokenBlueprintId ?? "").trim(),
  };

  if (scheduledBurnDate && String(scheduledBurnDate).trim()) {
    payload.scheduledBurnDate = String(scheduledBurnDate).trim();
  }

  logHttpRequest("postMintRequestHTTP", {
    method: "POST",
    url,
    headers: {
      Authorization: `Bearer ${safeTokenHint(idToken)}`,
      "Content-Type": "application/json",
    },
    productionId: trimmed,
    payload,
  });

  const res = await fetch(url, {
    method: "POST",
    headers: buildHeaders(idToken),
    body: JSON.stringify(payload),
  });

  logHttpResponse("postMintRequestHTTP", {
    method: "POST",
    url,
    status: res.status,
    statusText: res.statusText,
  });

  if (res.status === 404) return null;

  if (!res.ok) {
    const body = await readTextSafe(res);
    logHttpError("postMintRequestHTTP", {
      method: "POST",
      url,
      status: res.status,
      statusText: res.statusText,
      payload,
      bodyPreview: body ? body.slice(0, 1200) : "",
    });
    throw new Error(
      `Failed to post mint request: ${res.status} ${res.statusText}${
        body ? ` body=${body.slice(0, 400)}` : ""
      }`,
    );
  }

  const text = await readTextSafe(res);
  if (!text.trim()) return null;

  try {
    const json = JSON.parse(text) as InspectionBatchDTO | null | undefined;
    return json ?? null;
  } catch (e: any) {
    logHttpError("postMintRequestHTTP(parse)", {
      url,
      error: String(e?.message ?? e),
      bodyPreview: text.slice(0, 800),
    });
    return null;
  }
}
