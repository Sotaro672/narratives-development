// frontend/console/mintRequest/src/infrastructure/repository/http/mintRequests.ts

import { API_BASE } from "../../../../../shell/src/shared/http/apiBase";
import { getAuthJsonHeadersOrThrow } from "../../../../../shell/src/shared/http/authHeaders";

import type { InspectionBatchDTO } from "../../../domain/entity/inspections";
import type {
  MintDTO,
  MintListRowDTO,
} from "../../api/mintRequestApi";

import type { MintRequestManagementRowDTO } from "../../../application/dto/mintRequestManagementRow";

// ===============================
// types
// ===============================

type MintRequestsView = "management" | "list";

type FetchMintRequestsResult = {
  rows: MintRequestManagementRowDTO[];
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

function getProductionId(row: any): string | null {
  const productionId = String(row?.productionId ?? "").trim();
  return productionId || null;
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
    throw new Error(`Failed to fetch mint requests (network): ${String(e?.message ?? e)}`);
  }

  if (res.ok) {
    const json = (await res.json()) as MintRequestManagementRowDTO[] | null | undefined;

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
// Public exports
// ===============================

/**
 * productionId で 1 件の MintDTO を取得。
 * ✅ productionId を正とし、id / inspectionId fallback は使わない。
 */
export async function fetchMintByInspectionIdHTTP(
  productionId: string,
): Promise<MintDTO | null> {
  const pid = String(productionId ?? "").trim();

  if (!pid) {
    throw new Error("productionId が空です");
  }

  const { rows } = await fetchMintRequestsRowsRawOnce([pid], "management");

  const row =
    (rows ?? []).find((r) => getProductionId(r) === pid) ??
    rows?.[0] ??
    null;

  if (!row) return null;

  const mintRaw = (row as any)?.mint ?? null;

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
 * productionIds で MintDTO を map 取得。
 * ✅ key は productionId のみ。
 */
export async function fetchMintsByInspectionIdsHTTP(
  productionIds: string[],
): Promise<Record<string, MintDTO>> {
  const ids = uniqTrimmedStrings(productionIds ?? []);

  if (ids.length === 0) return {};

  const { rows } = await fetchMintRequestsRowsRawOnce(ids, "management");

  const out: Record<string, MintDTO> = {};

  for (const row of rows ?? []) {
    const key = getProductionId(row);
    if (!key) continue;

    const mintRaw = (row as any)?.mint ?? null;

    const merged = {
      ...(mintRaw ?? row ?? {}),
      createdByName:
        (row as any)?.requestedByName ??
        (row as any)?.createdByName ??
        (mintRaw as any)?.createdByName ??
        null,
      requestedByName: (row as any)?.requestedByName ?? null,
    };

    out[key] = merged as MintDTO;
  }

  return out;
}

/**
 * productionIds で “一覧用” MintListRowDTO を map 取得。
 * ✅ key は productionId のみ。
 */
export async function fetchMintListRowsByInspectionIdsHTTP(
  productionIds: string[],
): Promise<Record<string, MintListRowDTO>> {
  const ids = uniqTrimmedStrings(productionIds ?? []);

  if (ids.length === 0) return {};

  const { rows } = await fetchMintRequestsRowsRawOnce(ids, "list");

  const out: Record<string, MintListRowDTO> = {};

  for (const row of rows ?? []) {
    const key = getProductionId(row);
    if (!key) continue;

    out[key] = row as unknown as MintListRowDTO;
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
  const pid = String(productionId ?? "").trim();

  if (!pid) {
    throw new Error("productionId が空です");
  }

  const authHeaders = await getAuthJsonHeadersOrThrow();

  const url = `${API_BASE}/mint/inspections/${encodeURIComponent(pid)}/request`;

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