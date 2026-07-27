// frontend/console/shell/src/features/mintRequest/infrastructure/repository/http/mintRequests.ts

import { API_BASE } from "../../../../../shared/http/apiBase";
import { getAuthJsonHeadersOrThrow } from "../../../../../shared/http/authHeaders";

import type { MintDTO, MintListRowDTO } from "../../dto/mint.dto";

import type { MintRequestManagementRowDTO } from "../../../application/dto/mintRequestManagementRow";
import type { MintStatus } from "../../../domain/mints";

// ===============================
// types
// ===============================

export type MintRequestsView = "management" | "list";

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
  status: Extract<MintStatus, "QUEUED">;
  message: string;
};

// ===============================
// helpers
// ===============================

function uniqStrings(values: string[]): string[] {
  const seen = new Set<string>();
  const result: string[] = [];

  for (const value of values ?? []) {
    const normalized = String(value ?? "").trim();

    if (!normalized) continue;
    if (seen.has(normalized)) continue;

    seen.add(normalized);
    result.push(normalized);
  }

  return result;
}

function buildMintRequestsUrl(
  productionIds: string[],
  view: MintRequestsView,
): string {
  const query = new URLSearchParams({
    productionIds: productionIds.join(","),
    view,
  });

  return `${API_BASE}/mint/requests?${query.toString()}`;
}

function isServiceUnavailableStatus(status: number): boolean {
  return status === 502 || status === 503 || status === 504;
}

async function readTextSafe(response: Response): Promise<string> {
  try {
    return await response.text();
  } catch {
    return "";
  }
}

function getProductionId(
  row: MintRequestManagementRowDTO,
): string | null {
  const productionId = String(
    (row as Record<string, unknown>)?.productionId ?? "",
  ).trim();

  return productionId || null;
}

function toFiniteNumber(
  value: unknown,
  fallback = 0,
): number {
  const numberValue = Number(value);

  return Number.isFinite(numberValue)
    ? numberValue
    : fallback;
}

function toNonNegativeInteger(value: unknown): number {
  return Math.max(
    0,
    Math.trunc(toFiniteNumber(value, 0)),
  );
}

function clampPercentage(value: unknown): number {
  const percentage = toFiniteNumber(value, 0);

  if (percentage <= 0) return 0;
  if (percentage >= 100) return 100;

  return Math.trunc(percentage);
}

function normalizeMintStatus(
  value: unknown,
): MintStatus | null {
  const status = String(value ?? "").toUpperCase();

  switch (status) {
    case "CREATED":
    case "QUEUED":
    case "MINTING":
    case "PARTIALLY_MINTED":
    case "MINTED":
    case "FAILED_RETRYABLE":
    case "FAILED_FATAL":
      return status;

    default:
      return null;
  }
}

function normalizeMintProgress(
  raw: unknown,
): MintTaskProgressDTO | null {
  if (!raw || typeof raw !== "object") {
    return null;
  }

  const value = raw as Record<string, unknown>;

  const total = toNonNegativeInteger(value.total);
  const pending = toNonNegativeInteger(value.pending);
  const minting = toNonNegativeInteger(value.minting);
  const minted = toNonNegativeInteger(value.minted);
  const failedRetryable = toNonNegativeInteger(
    value.failedRetryable,
  );
  const failedFatal = toNonNegativeInteger(
    value.failedFatal,
  );

  const calculatedPercentage =
    total > 0
      ? Math.trunc(
          (Math.min(minted, total) / total) * 100,
        )
      : 0;

  return {
    total,
    pending,
    minting,
    minted,
    failedRetryable,
    failedFatal,
    percentage:
      value.percentage === undefined
        ? clampPercentage(calculatedPercentage)
        : clampPercentage(value.percentage),
  };
}

function normalizeMintQueuedResponse(
  raw: unknown,
  fallbackProductionId: string,
): MintQueuedResponse | null {
  if (!raw || typeof raw !== "object") {
    return null;
  }

  const value = raw as Record<string, unknown>;

  const mintRequestId = String(
    value.mintRequestId ?? "",
  ).trim();

  const productionId = String(
    value.productionId ?? fallbackProductionId,
  ).trim();

  const status = normalizeMintStatus(value.status);
  const message = String(value.message ?? "");

  if (
    !mintRequestId ||
    !productionId ||
    status !== "QUEUED"
  ) {
    return null;
  }

  return {
    mintRequestId,
    productionId,
    status,
    message,
  };
}

function mergeMintDTOFromRow(
  row: MintRequestManagementRowDTO,
): MintDTO | null {
  const rawRow = row as Record<string, any>;

  const mintRaw =
    rawRow.mint &&
    typeof rawRow.mint === "object"
      ? rawRow.mint
      : null;

  const status = normalizeMintStatus(
    mintRaw?.status ?? rawRow.status,
  );

  if (!status) {
    return null;
  }

  const mintProgress =
    normalizeMintProgress(rawRow.mintProgress) ??
    normalizeMintProgress(mintRaw?.mintProgress);

  const merged = {
    ...(mintRaw ?? rawRow),

    status,

    createdByName:
      rawRow.requestedByName ??
      rawRow.createdByName ??
      mintRaw?.createdByName ??
      null,

    requestedByName:
      rawRow.requestedByName ??
      mintRaw?.requestedByName ??
      null,

    onChainTxSignature:
      mintRaw?.onChainTxSignature ??
      rawRow.onChainTxSignature ??
      mintRaw?.txSignature ??
      rawRow.txSignature ??
      null,

    mintProgress,
  };

  return merged as MintDTO;
}

// ===============================
// GET: /mint/requests
// ===============================

/**
 * GET /mint/requests の唯一の取得処理。
 *
 * - productionIds は空文字と重複を除去する
 * - Backend response は配列を正とする
 * - items / rows / data などの旧レスポンス形状は吸収しない
 * - HTTPエラー、空レスポンス、不正JSON、不正なレスポンス型を区別する
 */
export async function fetchMintRequestRowsHTTP(
  productionIds: string[],
  view: MintRequestsView = "management",
): Promise<MintRequestManagementRowDTO[]> {
  const ids = uniqStrings(productionIds);

  if (ids.length === 0) {
    return [];
  }

  const authHeaders =
    await getAuthJsonHeadersOrThrow();

  const url = buildMintRequestsUrl(ids, view);

  let response: Response;

  try {
    response = await fetch(url, {
      method: "GET",
      headers: authHeaders,
    });
  } catch (error: unknown) {
    const message =
      error instanceof Error
        ? error.message
        : String(error);

    throw new Error(
      `Failed to fetch mint requests (network): ${message}`,
    );
  }

  const text = await readTextSafe(response);

  if (!response.ok) {
    const hint = isServiceUnavailableStatus(
      response.status,
    )
      ? " (service unavailable)"
      : "";

    throw new Error(
      `Failed to fetch mint requests${hint}: ` +
        `${response.status} ${response.statusText}` +
        (text
          ? ` body=${text.slice(0, 400)}`
          : ""),
    );
  }

  if (!text.trim()) {
    throw new Error(
      "Failed to fetch mint requests: response is empty",
    );
  }

  let payload: unknown;

  try {
    payload = JSON.parse(text);
  } catch {
    throw new Error(
      "Failed to fetch mint requests: response is not valid JSON",
    );
  }

  if (!Array.isArray(payload)) {
    throw new Error(
      "Failed to fetch mint requests: response must be an array",
    );
  }

  return payload as MintRequestManagementRowDTO[];
}

// ===============================
// GET: MintDTO
// ===============================

/**
 * productionId で1件のMintDTOを取得する。
 *
 * productionIdを正とし、
 * id / inspectionId fallbackは使用しない。
 */
export async function fetchMintByProductionIdHTTP(
  productionId: string,
): Promise<MintDTO | null> {
  const normalizedProductionId = String(
    productionId ?? "",
  ).trim();

  if (!normalizedProductionId) {
    throw new Error("productionId が空です");
  }

  const rows = await fetchMintRequestRowsHTTP(
    [normalizedProductionId],
    "management",
  );

  const row =
    rows.find(
      (item) =>
        getProductionId(item) ===
        normalizedProductionId,
    ) ??
    rows[0] ??
    null;

  if (!row) {
    return null;
  }

  return mergeMintDTOFromRow(row);
}

/**
 * productionIdsでMintDTOをまとめて取得する。
 *
 * 戻り値のkeyはproductionIdのみとする。
 */
export async function fetchMintsByProductionIdsHTTP(
  productionIds: string[],
): Promise<Record<string, MintDTO>> {
  const ids = uniqStrings(productionIds);

  if (ids.length === 0) {
    return {};
  }

  const rows = await fetchMintRequestRowsHTTP(
    ids,
    "management",
  );

  const result: Record<string, MintDTO> = {};

  for (const row of rows) {
    const productionId = getProductionId(row);

    if (!productionId) continue;

    const mint = mergeMintDTOFromRow(row);

    if (!mint) continue;

    result[productionId] = mint;
  }

  return result;
}

/**
 * productionIdsで一覧表示用MintListRowDTOを
 * まとめて取得する。
 *
 * 戻り値のkeyはproductionIdのみとする。
 */
export async function fetchMintListRowsByProductionIdsHTTP(
  productionIds: string[],
): Promise<Record<string, MintListRowDTO>> {
  const ids = uniqStrings(productionIds);

  if (ids.length === 0) {
    return {};
  }

  const rows = await fetchMintRequestRowsHTTP(
    ids,
    "list",
  );

  const result: Record<string, MintListRowDTO> = {};

  for (const row of rows) {
    const productionId = getProductionId(row);

    if (!productionId) continue;

    const rawRow = row as Record<string, any>;

    const status = normalizeMintStatus(
      rawRow.status ?? rawRow.mint?.status,
    );

    result[productionId] = {
      ...rawRow,
      productionId,
      status,
    } as MintListRowDTO;
  }

  return result;
}

// ===============================
// POST: mint request
// ===============================

export async function postMintRequestHTTP(
  productionId: string,
  tokenBlueprintId: string,
  scheduledBurnDate?: string,
): Promise<MintQueuedResponse | null> {
  const normalizedProductionId = String(
    productionId ?? "",
  ).trim();

  if (!normalizedProductionId) {
    throw new Error("productionId が空です");
  }

  const normalizedTokenBlueprintId = String(
    tokenBlueprintId ?? "",
  ).trim();

  if (!normalizedTokenBlueprintId) {
    throw new Error("tokenBlueprintId が空です");
  }

  const authHeaders =
    await getAuthJsonHeadersOrThrow();

  const url =
    `${API_BASE}/mint/inspections/` +
    `${encodeURIComponent(normalizedProductionId)}` +
    "/request";

  const payload: {
    tokenBlueprintId: string;
    scheduledBurnDate?: string;
  } = {
    tokenBlueprintId:
      normalizedTokenBlueprintId,
  };

  const normalizedScheduledBurnDate = String(
    scheduledBurnDate ?? "",
  ).trim();

  if (normalizedScheduledBurnDate) {
    payload.scheduledBurnDate =
      normalizedScheduledBurnDate;
  }

  let response: Response;

  try {
    response = await fetch(url, {
      method: "POST",
      headers: authHeaders,
      body: JSON.stringify(payload),
    });
  } catch (error: unknown) {
    const message =
      error instanceof Error
        ? error.message
        : String(error);

    throw new Error(
      `Failed to post mint request (network): ${message}`,
    );
  }

  if (response.status === 404) {
    return null;
  }

  const text = await readTextSafe(response);

  if (!response.ok) {
    throw new Error(
      `Failed to post mint request: ` +
        `${response.status} ${response.statusText}` +
        (text
          ? ` body=${text.slice(0, 400)}`
          : ""),
    );
  }

  if (!text.trim()) {
    return null;
  }

  let payloadResponse: unknown;

  try {
    payloadResponse = JSON.parse(text);
  } catch {
    throw new Error(
      "Failed to post mint request: response is not valid JSON",
    );
  }

  return normalizeMintQueuedResponse(
    payloadResponse,
    normalizedProductionId,
  );
}