// frontend/console/shell/src/features/mintRequest/infrastructure/repository/http/mintRequests.ts

import { API_BASE } from "../../../../../shared/http/apiBase";
import { getAuthJsonHeadersOrThrow } from "../../../../../shared/http/authHeaders";

import type {
  MintFundingEstimate,
} from "../../../application/port/MintRequestRepository";
import type { MintRequestManagementRowDTO } from "../../dto/mintRequestManagementRow";
import type { MintStatus } from "../../../../../shared/types/mints";

// ===============================
// types
// ===============================

export type MintRequestsView = "management" | "list";

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
    if (!normalized || seen.has(normalized)) {
      continue;
    }

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
    view,
  });

  if (productionIds.length > 0) {
    query.set(
      "productionIds",
      productionIds.join(","),
    );
  }

  return `${API_BASE}/mint/requests?${query.toString()}`;
}

function buildMintFundingEstimateUrl(
  productionId: string,
  tokenBlueprintId: string,
): string {
  const query = new URLSearchParams({
    productionId,
    tokenBlueprintId,
  });

  return `${API_BASE}/mint/funding-estimate?${query.toString()}`;
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
): string {
  return row.productionId;
}

function normalizeMintStatus(value: unknown): MintStatus | null {
  const status = String(value ?? "").trim().toUpperCase();

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

function normalizeMintQueuedResponse(
  raw: unknown,
  fallbackProductionId: string,
): MintQueuedResponse | null {
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) {
    return null;
  }

  const value = raw as Record<string, unknown>;

  const mintRequestId = String(value.mintRequestId ?? "").trim();
  const productionId = String(
    value.productionId ?? fallbackProductionId,
  ).trim();
  const status = normalizeMintStatus(value.status);
  const message = String(value.message ?? "");

  if (!mintRequestId || !productionId || status !== "QUEUED") {
    return null;
  }

  return {
    mintRequestId,
    productionId,
    status,
    message,
  };
}

// ===============================
// GET: /mint/requests
// ===============================

/**
 * GET /mint/requestsの唯一の取得処理。
 *
 * - productionIdsは空文字と重複を除去する
 * - productionIdsが空の場合はBackend側で現在companyの全件を取得する
 * - Backend responseは配列を正とする
 * - items / rows / dataなどの旧レスポンス形状は吸収しない
 * - Backend BFFのfield名と型をそのまま正とする
 * - HTTPエラー、空レスポンス、不正JSON、不正なレスポンス型を区別する
 */
export async function fetchMintRequestRowsHTTP(
  productionIds: string[],
  view: MintRequestsView = "management",
): Promise<MintRequestManagementRowDTO[]> {
  const ids = uniqStrings(productionIds);

  const authHeaders = await getAuthJsonHeadersOrThrow();
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
    const hint = isServiceUnavailableStatus(response.status)
      ? " (service unavailable)"
      : "";

    throw new Error(
      `Failed to fetch mint requests${hint}: ` +
        `${response.status} ${response.statusText}` +
        (text ? ` body=${text.slice(0, 400)}` : ""),
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
// GET: management row
// ===============================

/**
 * productionIdでGET /mint/requestsのmanagement rowを1件取得する。
 *
 * Mint detailではこのresponseを正として使用する。
 */
export async function fetchMintRequestRowByProductionIdHTTP(
  productionId: string,
): Promise<MintRequestManagementRowDTO | null> {
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

  return (
    rows.find(
      (item) =>
        getProductionId(item) ===
        normalizedProductionId,
    ) ??
    null
  );
}

// ===============================
// GET: /mint/funding-estimate
// ===============================

/**
 * productionIdとtokenBlueprintIdから
 * Bubblegum V2 Mintに必要なSOL見積を取得する。
 *
 * metadataUriはFrontendから渡さない。
 * mintQuantity、Brand Wallet、TokenBlueprint情報はBackend側で解決する。
 *
 * Backend BFFのresponseを正とし、
 * Frontendではfieldの再構築や型変換を行わない。
 */
export async function fetchMintFundingEstimateHTTP(
  productionId: string,
  tokenBlueprintId: string,
): Promise<MintFundingEstimate> {
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

  const authHeaders = await getAuthJsonHeadersOrThrow();
  const url = buildMintFundingEstimateUrl(
    normalizedProductionId,
    normalizedTokenBlueprintId,
  );

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
      `Failed to fetch mint funding estimate (network): ${message}`,
    );
  }

  const text = await readTextSafe(response);

  if (!response.ok) {
    const hint = isServiceUnavailableStatus(response.status)
      ? " (service unavailable)"
      : "";

    throw new Error(
      `Failed to fetch mint funding estimate${hint}: ` +
        `${response.status} ${response.statusText}` +
        (text ? ` body=${text.slice(0, 400)}` : ""),
    );
  }

  if (!text.trim()) {
    throw new Error(
      "Failed to fetch mint funding estimate: response is empty",
    );
  }

  let payload: unknown;

  try {
    payload = JSON.parse(text);
  } catch {
    throw new Error(
      "Failed to fetch mint funding estimate: response is not valid JSON",
    );
  }

  return payload as MintFundingEstimate;
}

// ===============================
// POST: mint request
// ===============================

/**
 * ミント申請を送信する。
 *
 * Backendは申請を受け付けた後、
 * 非同期でミント処理を開始する。
 *
 * 正常受付時は202 Acceptedと
 * QUEUEDレスポンスを返す。
 *
 * scheduledBurnDateはFrontendから送信しない。
 */
export async function postMintRequestHTTP(
  productionId: string,
  tokenBlueprintId: string,
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

  const authHeaders = await getAuthJsonHeadersOrThrow();

  const url =
    `${API_BASE}/mint/inspections/` +
    `${encodeURIComponent(normalizedProductionId)}` +
    "/request";

  const requestPayload = {
    tokenBlueprintId: normalizedTokenBlueprintId,
  };

  let response: Response;

  try {
    response = await fetch(url, {
      method: "POST",
      headers: authHeaders,
      body: JSON.stringify(requestPayload),
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
      "Failed to post mint request: " +
        `${response.status} ${response.statusText}` +
        (text ? ` body=${text.slice(0, 400)}` : ""),
    );
  }

  if (!text.trim()) {
    return null;
  }

  let responsePayload: unknown;

  try {
    responsePayload = JSON.parse(text);
  } catch {
    throw new Error(
      "Failed to post mint request: response is not valid JSON",
    );
  }

  return normalizeMintQueuedResponse(
    responsePayload,
    normalizedProductionId,
  );
}