// frontend/console/shell/src/features/mintRequest/infrastructure/repository/http/mintRequests.ts

import { API_BASE } from "../../../../../shared/http/apiBase";
import { getAuthJsonHeadersOrThrow } from "../../../../../shared/http/authHeaders";

import type {
  MintFundingEstimate,
  MintQueuedResponse,
} from "../../../application/port/MintRequestRepository";
import type { MintRequestManagementRowDTO } from "../../dto/mintRequestManagementRow";

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

function buildMintRequestsUrl(productionIds: string[]): string {
  const query = new URLSearchParams();

  if (productionIds.length > 0) {
    query.set("productionIds", productionIds.join(","));
  }

  const queryString = query.toString();

  return queryString
    ? `${API_BASE}/mint/requests?${queryString}`
    : `${API_BASE}/mint/requests`;
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

// ===============================
// GET: /mint/requests
// ===============================

/**
 * GET /mint/requestsの唯一の取得処理。
 *
 * - productionIdsは空文字と重複を除去する
 * - productionIdsが空の場合はBackend側で現在companyの全件を取得する
 * - Backend responseの配列を正とする
 * - items / rows / dataなどの旧レスポンス形状は吸収しない
 * - Backend BFFのfield名と型をそのまま使用する
 */
export async function fetchMintRequestRowsHTTP(
  productionIds: string[],
): Promise<MintRequestManagementRowDTO[]> {
  const ids = uniqStrings(productionIds);
  const authHeaders = await getAuthJsonHeadersOrThrow();
  const url = buildMintRequestsUrl(ids);

  let response: Response;

  try {
    response = await fetch(url, {
      method: "GET",
      headers: authHeaders,
    });
  } catch (error: unknown) {
    const message = error instanceof Error ? error.message : String(error);

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
// GET: mint request row
// ===============================

/**
 * productionIdでGET /mint/requestsのrowを1件取得する。
 *
 * Mint detailではこのBackend responseを正として使用する。
 */
export async function fetchMintRequestRowByProductionIdHTTP(
  productionId: string,
): Promise<MintRequestManagementRowDTO | null> {
  const normalizedProductionId = String(productionId ?? "").trim();

  if (!normalizedProductionId) {
    throw new Error("productionId が空です");
  }

  const rows = await fetchMintRequestRowsHTTP([
    normalizedProductionId,
  ]);

  return (
    rows.find(
      (row) => row.productionId === normalizedProductionId,
    ) ?? null
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
  const normalizedProductionId = String(productionId ?? "").trim();

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
    const message = error instanceof Error ? error.message : String(error);

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
 * Backend BFFのMintQueuedResponseを返す。
 *
 * scheduledBurnDateはFrontendから送信しない。
 */
export async function postMintRequestHTTP(
  productionId: string,
  tokenBlueprintId: string,
): Promise<MintQueuedResponse | null> {
  const normalizedProductionId = String(productionId ?? "").trim();

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
    const message = error instanceof Error ? error.message : String(error);

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

  let payload: unknown;

  try {
    payload = JSON.parse(text);
  } catch {
    throw new Error(
      "Failed to post mint request: response is not valid JSON",
    );
  }

  return payload as MintQueuedResponse;
}