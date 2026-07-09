// frontend/console/mintRequest/src/infrastructure/repository/http/mintRequests.ts

import { API_BASE } from "../../../../../shell/src/shared/http/apiBase";
import { getAuthJsonHeadersOrThrow } from "../../../../../shell/src/shared/http/authHeaders";

import type { MintDTO, MintListRowDTO } from "../../api/mintRequestApi";

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

export type MintTaskProgressDTO = {
  total: number;
  pending: number;
  minting: number;
  minted: number;
  failedRetryable: number;
  failedFatal: number;
  percentage: number;
};

export type MintQueuedResponse = {
  mintRequestId: string;
  productionId: string;
  status: "QUEUED";
  message: string;
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

function buildMintRequestsUrl(
  productionIds: string[],
  view: MintRequestsView,
): string {
  const base = `${API_BASE}/mint/requests?productionIds=${encodeURIComponent(
    productionIds.join(","),
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

function toFiniteNumber(value: unknown, fallback = 0): number {
  const n = Number(value);
  return Number.isFinite(n) ? n : fallback;
}

function clampPercentage(value: unknown): number {
  const n = toFiniteNumber(value, 0);
  if (n <= 0) return 0;
  if (n >= 100) return 100;
  return Math.trunc(n);
}

function normalizeMintProgress(raw: unknown): MintTaskProgressDTO | null {
  if (!raw || typeof raw !== "object") return null;

  const obj = raw as Record<string, unknown>;

  const total = Math.max(0, Math.trunc(toFiniteNumber(obj.total, 0)));
  const pending = Math.max(0, Math.trunc(toFiniteNumber(obj.pending, 0)));
  const minting = Math.max(0, Math.trunc(toFiniteNumber(obj.minting, 0)));
  const minted = Math.max(0, Math.trunc(toFiniteNumber(obj.minted, 0)));
  const failedRetryable = Math.max(
    0,
    Math.trunc(toFiniteNumber(obj.failedRetryable, 0)),
  );
  const failedFatal = Math.max(
    0,
    Math.trunc(toFiniteNumber(obj.failedFatal, 0)),
  );

  const calculatedPercentage =
    total > 0 ? Math.trunc((Math.min(minted, total) / total) * 100) : 0;

  return {
    total,
    pending,
    minting,
    minted,
    failedRetryable,
    failedFatal,
    percentage:
      obj.percentage === undefined
        ? clampPercentage(calculatedPercentage)
        : clampPercentage(obj.percentage),
  };
}

function normalizeBoolean(value: unknown): boolean {
  if (typeof value === "boolean") return value;

  const s = String(value ?? "").trim().toLowerCase();
  return s === "true" || s === "1" || s === "yes";
}

function normalizeMintQueuedResponse(
  raw: unknown,
  fallbackProductionId: string,
): MintQueuedResponse | null {
  if (!raw || typeof raw !== "object") return null;

  const obj = raw as Record<string, unknown>;

  const mintRequestId = String(obj.mintRequestId ?? "").trim();
  const productionId = String(obj.productionId ?? fallbackProductionId).trim();
  const status = String(obj.status ?? "").trim();
  const message = String(obj.message ?? "").trim();

  if (!mintRequestId || !productionId || status !== "QUEUED") {
    return null;
  }

  return {
    mintRequestId,
    productionId,
    status: "QUEUED",
    message,
  };
}

function mergeMintDTOFromRow(row: MintRequestManagementRowDTO): MintDTO {
  const anyRow = row as any;
  const mintRaw = anyRow?.mint ?? null;

  const mintProgress =
    normalizeMintProgress(anyRow?.mintProgress) ??
    normalizeMintProgress(mintRaw?.mintProgress);

  const merged = {
    ...(mintRaw ?? anyRow ?? {}),

    createdByName:
      anyRow?.requestedByName ??
      anyRow?.createdByName ??
      mintRaw?.createdByName ??
      null,

    requestedByName: anyRow?.requestedByName ?? null,

    onChainTxSignature:
      mintRaw?.onChainTxSignature ??
      anyRow?.onChainTxSignature ??
      mintRaw?.txSignature ??
      anyRow?.txSignature ??
      null,

    minted: normalizeBoolean(mintRaw?.minted ?? anyRow?.minted),

    mintProgress,
  };

  return merged as MintDTO;
}

/**
 * Raw fetch for a single view.
 * - fallback は行わない
 * - 404/405 もそのままエラーにする
 * - Backend response は配列を正とする
 */
async function fetchMintRequestsRowsRawOnce(
  productionIds: string[],
  view: MintRequestsView,
): Promise<FetchMintRequestsResult> {
  const safeProductionIds = uniqTrimmedStrings(productionIds ?? []);

  if (safeProductionIds.length === 0) {
    return { rows: [], usedView: view, usedUrl: "" };
  }

  const authHeaders = await getAuthJsonHeadersOrThrow();

  const url = buildMintRequestsUrl(safeProductionIds, view);

  let res: Response;

  try {
    res = await fetch(url, { method: "GET", headers: authHeaders });
  } catch (e: any) {
    throw new Error(
      `Failed to fetch mint requests (network): ${String(e?.message ?? e)}`,
    );
  }

  if (res.ok) {
    const json =
      (await res.json()) as MintRequestManagementRowDTO[] | null | undefined;

    return {
      rows: Array.isArray(json) ? json : [],
      usedView: view,
      usedUrl: url,
    };
  }

  const body = await readTextSafe(res);

  const hint = isServiceUnavailableStatus(res.status)
    ? " (service unavailable)"
    : "";

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
export async function fetchMintByProductionIdHTTP(
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

  return mergeMintDTOFromRow(row);
}

/**
 * productionIds で MintDTO を map 取得。
 * ✅ key は productionId のみ。
 */
export async function fetchMintsByProductionIdsHTTP(
  productionIds: string[],
): Promise<Record<string, MintDTO>> {
  const ids = uniqTrimmedStrings(productionIds ?? []);

  if (ids.length === 0) return {};

  const { rows } = await fetchMintRequestsRowsRawOnce(ids, "management");

  const out: Record<string, MintDTO> = {};

  for (const row of rows ?? []) {
    const key = getProductionId(row);
    if (!key) continue;

    out[key] = mergeMintDTOFromRow(row);
  }

  return out;
}

/**
 * productionIds で “一覧用” MintListRowDTO を map 取得。
 * ✅ key は productionId のみ。
 */
export async function fetchMintListRowsByProductionIdsHTTP(
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
): Promise<MintQueuedResponse | null> {
  const pid = String(productionId ?? "").trim();

  if (!pid) {
    throw new Error("productionId が空です");
  }

  const tbID = String(tokenBlueprintId ?? "").trim();
  if (!tbID) {
    throw new Error("tokenBlueprintId が空です");
  }

  const authHeaders = await getAuthJsonHeadersOrThrow();

  const url = `${API_BASE}/mint/inspections/${encodeURIComponent(pid)}/request`;

  const payload: { tokenBlueprintId: string; scheduledBurnDate?: string } = {
    tokenBlueprintId: tbID,
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

  const text = await readTextSafe(res);

  if (!res.ok) {
    throw new Error(
      `Failed to post mint request: ${res.status} ${res.statusText}${
        text ? ` body=${text.slice(0, 400)}` : ""
      }`,
    );
  }

  if (!text.trim()) return null;

  try {
    const json = JSON.parse(text) as unknown;
    return normalizeMintQueuedResponse(json, pid);
  } catch {
    return null;
  }
}