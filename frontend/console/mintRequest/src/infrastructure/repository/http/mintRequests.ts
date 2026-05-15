// frontend/console/mintRequest/src/infrastructure/repository/http/mintRequests.ts

import { API_BASE } from "../../../../../shell/src/shared/http/apiBase";
import { getAuthJsonHeadersOrThrow } from "../../../../../shell/src/shared/http/authHeaders";

import type {
  InspectionBatchDTO,
  MintDTO,
  MintListRowDTO,
} from "../../api/mintRequestApi";

import type { MintRequestRowRaw } from "../../dto/mintRequestRaw.dto";

// ===============================
// types
// ===============================

// ✅ "dto" view は今回の不具合原因になりうるため、フロント側からは使用しない前提
type MintRequestsView = "management" | "list";

type FetchMintRequestsResult = {
  rows: MintRequestRowRaw[];
  usedView: MintRequestsView;
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

function buildMintRequestsUrl(ids: string[], view: MintRequestsView): string {
  const base = `${API_BASE}/mint/requests?productionIds=${encodeURIComponent(
    ids.join(","),
  )}`;

  return `${base}&view=${encodeURIComponent(view)}`;
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

function getRowProductionId(row: any): string | null {
  return row?.productionId ?? row?.id ?? null;
}

/**
 * Raw fetch for a single view.
 * - fallback は行わない
 * - 404/405 もそのままエラーにする
 * - Backend response は配列を正とする
 */
async function fetchMintRequestsRowsRawOnce(
  ids: string[],
  view: MintRequestsView,
): Promise<FetchMintRequestsResult> {
  const safeIds = uniqTrimmedStrings(ids ?? []);

  if (safeIds.length === 0) {
    return { rows: [], usedView: view, usedUrl: "" };
  }

  const authHeaders = await getAuthJsonHeadersOrThrow();

  const url = buildMintRequestsUrl(safeIds, view);

  let res: Response;

  try {
    res = await fetch(url, { method: "GET", headers: authHeaders });
  } catch (e: any) {
    // fetch 自体が落ちる (CORS / network / DNS etc.)
    throw new Error(`Failed to fetch mint requests (network): ${String(e?.message ?? e)}`);
  }

  if (res.ok) {
    const json = (await res.json()) as MintRequestRowRaw[] | null | undefined;

    return {
      rows: Array.isArray(json) ? json : [],
      usedView: view,
      usedUrl: url,
    };
  }

  const body = await readTextSafe(res);

  const hint = isServiceUnavailableStatus(res.status) ? " (service unavailable)" : "";

  throw new Error(
    `Failed to fetch mint requests${hint}: ${res.status} ${res.statusText}${
      body ? ` body=${body.slice(0, 400)}` : ""
    }`,
  );
}

// ===============================
// Public exports (一覧/参照系)
// ===============================

/**
 * inspectionId (= productionId) で 1 件の MintDTO を取得。
 * ✅ view=management のみ使用する
 */
export async function fetchMintByInspectionIdHTTP(
  inspectionId: string,
): Promise<MintDTO | null> {
  const iid = String(inspectionId ?? "").trim();

  if (!iid) {
    throw new Error("inspectionId が空です");
  }

  const { rows } = await fetchMintRequestsRowsRawOnce([iid], "management");

  const row =
    (rows ?? []).find((r) => getRowProductionId(r) === iid) ??
    rows?.[0] ??
    null;

  if (!row) return null;

  const mintRaw = (row as any)?.mint ?? null;

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

  return merged as MintDTO;
}

/**
 * inspectionIds (= productionIds) で MintDTO を map で取得。
 * ✅ view=management のみ使用する
 */
export async function fetchMintsByInspectionIdsHTTP(
  inspectionIds: string[],
): Promise<Record<string, MintDTO>> {
  const ids = uniqTrimmedStrings(inspectionIds ?? []);

  if (ids.length === 0) return {};

  const { rows } = await fetchMintRequestsRowsRawOnce(ids, "management");

  const out: Record<string, MintDTO> = {};

  for (const r of rows ?? []) {
    const key = getRowProductionId(r);
    if (!key) continue;

    const mintRaw = (r as any)?.mint ?? null;

    const merged = {
      ...(mintRaw ?? r ?? {}),
      createdByName:
        (r as any)?.requestedByName ??
        (r as any)?.createdByName ??
        (mintRaw as any)?.createdByName ??
        null,
      requestedByName: (r as any)?.requestedByName ?? null,
    };

    out[key] = merged as MintDTO;
  }

  return out;
}

/**
 * inspectionIds (= productionIds) で “一覧用” MintListRowDTO を map で取得。
 * ✅ view=list response を正とし、normalizer は使用しない
 */
export async function fetchMintListRowsByInspectionIdsHTTP(
  inspectionIds: string[],
): Promise<Record<string, MintListRowDTO>> {
  const ids = uniqTrimmedStrings(inspectionIds ?? []);

  if (ids.length === 0) return {};

  const { rows } = await fetchMintRequestsRowsRawOnce(ids, "list");

  const out: Record<string, MintListRowDTO> = {};

  for (const r of rows ?? []) {
    const key = getRowProductionId(r);
    if (!key) continue;

    out[key] = r as unknown as MintListRowDTO;
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

  if (!trimmed) {
    throw new Error("productionId が空です");
  }

  const authHeaders = await getAuthJsonHeadersOrThrow();

  const url = `${API_BASE}/mint/inspections/${encodeURIComponent(trimmed)}/request`;

  const payload: { tokenBlueprintId: string; scheduledBurnDate?: string } = {
    tokenBlueprintId: String(tokenBlueprintId ?? "").trim(),
  };

  if (scheduledBurnDate && String(scheduledBurnDate).trim()) {
    payload.scheduledBurnDate = String(scheduledBurnDate).trim();
  }

  const res = await fetch(url, {
    method: "POST",
    headers: authHeaders,
    body: JSON.stringify(payload),
  });

  if (res.status === 404) return null;

  if (!res.ok) {
    const body = await readTextSafe(res);

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
  } catch {
    return null;
  }
}