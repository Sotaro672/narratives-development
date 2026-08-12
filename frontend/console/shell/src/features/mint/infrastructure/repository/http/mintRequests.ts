// frontend/console/shell/src/features/mintRequest/infrastructure/repository/http/mintRequests.ts

import { API_BASE } from "../../../../../shared/http/apiBase";
import { getAuthJsonHeadersOrThrow } from "../../../../../shared/http/authHeaders";

import type { MintDTO } from "../../dto/mint.dto";
import type {
  MintFundingEstimate,
  MintTaskProgress,
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
    productionIds: productionIds.join(","),
    view,
  });

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
): string | null {
  const productionId = String(
    (row as Record<string, unknown>).productionId ?? "",
  ).trim();

  return productionId || null;
}

function toFiniteNumber(value: unknown, fallback = 0): number {
  const numberValue = Number(value);
  return Number.isFinite(numberValue) ? numberValue : fallback;
}

function toNonNegativeInteger(value: unknown): number {
  return Math.max(0, Math.trunc(toFiniteNumber(value, 0)));
}

function clampPercentage(value: unknown): number {
  const percentage = toFiniteNumber(value, 0);

  if (percentage <= 0) {
    return 0;
  }

  if (percentage >= 100) {
    return 100;
  }

  return Math.trunc(percentage);
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

function normalizeMintProgress(raw: unknown): MintTaskProgress | null {
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) {
    return null;
  }

  const value = raw as Record<string, unknown>;

  const total = toNonNegativeInteger(value.total);
  const pending = toNonNegativeInteger(value.pending);
  const minting = toNonNegativeInteger(value.minting);
  const minted = toNonNegativeInteger(value.minted);
  const failedRetryable = toNonNegativeInteger(value.failedRetryable);
  const failedFatal = toNonNegativeInteger(value.failedFatal);

  const calculatedPercentage =
    total > 0
      ? Math.trunc((Math.min(minted, total) / total) * 100)
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

function asRecord(value: unknown): Record<string, unknown> | null {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return null;
  }

  return value as Record<string, unknown>;
}

function requiredStringValue(value: unknown): string | null {
  if (typeof value !== "string") {
    return null;
  }

  const normalized = value.trim();
  return normalized || null;
}

function nullableStringValue(value: unknown): string | null {
  if (value === null || value === undefined) {
    return null;
  }

  if (typeof value !== "string") {
    return null;
  }

  const normalized = value.trim();
  return normalized || null;
}

function requiredFiniteNumber(value: unknown): number | null {
  if (typeof value !== "number" || !Number.isFinite(value)) {
    return null;
  }

  return value;
}

function requiredBoolean(value: unknown): boolean | null {
  return typeof value === "boolean" ? value : null;
}

function normalizeMintFundingEstimate(
  raw: unknown,
): MintFundingEstimate | null {
  const value = asRecord(raw);
  if (!value) {
    return null;
  }

  const reserve = asRecord(value.reserve);
  const feePayer = asRecord(value.feePayer);
  const resources = asRecord(value.resources);
  const estimate = asRecord(value.estimate);

  if (!reserve || !feePayer || !resources || !estimate) {
    return null;
  }

  const cluster = requiredStringValue(value.cluster);
  const mintQuantity = requiredFiniteNumber(value.mintQuantity);

  const reserveAddress = requiredStringValue(reserve.address);
  const reserveBalanceLamports = requiredStringValue(
    reserve.balanceLamports,
  );
  const reserveBalanceSol = requiredFiniteNumber(reserve.balanceSol);
  const reserveMinimumLamports = requiredStringValue(
    reserve.minimumLamports,
  );
  const reserveMinimumSol = requiredFiniteNumber(reserve.minimumSol);

  const feePayerAddress = requiredStringValue(feePayer.address);
  const feePayerBalanceLamports = requiredStringValue(
    feePayer.balanceLamports,
  );
  const feePayerBalanceSol = requiredFiniteNumber(
    feePayer.balanceSol,
  );
  const feePayerTargetLamports = requiredStringValue(
    feePayer.targetLamports,
  );
  const feePayerTargetSol = requiredFiniteNumber(
    feePayer.targetSol,
  );

  const sharedMerkleTreeExists = requiredBoolean(
    resources.sharedMerkleTreeExists,
  );
  const coreCollectionExists = requiredBoolean(
    resources.coreCollectionExists,
  );

  if (
    !cluster ||
    mintQuantity === null ||
    !Number.isSafeInteger(mintQuantity) ||
    mintQuantity <= 0 ||
    !reserveAddress ||
    !reserveBalanceLamports ||
    reserveBalanceSol === null ||
    !reserveMinimumLamports ||
    reserveMinimumSol === null ||
    !feePayerAddress ||
    !feePayerBalanceLamports ||
    feePayerBalanceSol === null ||
    !feePayerTargetLamports ||
    feePayerTargetSol === null ||
    sharedMerkleTreeExists === null ||
    coreCollectionExists === null
  ) {
    return null;
  }

  const mintTransactionFeePerItemLamports = requiredStringValue(
    estimate.mintTransactionFeePerItemLamports,
  );
  const mintTransactionFeePerItemSol = requiredFiniteNumber(
    estimate.mintTransactionFeePerItemSol,
  );
  const mintTransactionFeeTotalLamports = requiredStringValue(
    estimate.mintTransactionFeeTotalLamports,
  );
  const mintTransactionFeeTotalSol = requiredFiniteNumber(
    estimate.mintTransactionFeeTotalSol,
  );

  const merkleTreeCreationTransactionFeeLamports =
    requiredStringValue(
      estimate.merkleTreeCreationTransactionFeeLamports,
    );
  const merkleTreeCreationTransactionFeeSol = requiredFiniteNumber(
    estimate.merkleTreeCreationTransactionFeeSol,
  );
  const merkleTreeCreationRentLamports = requiredStringValue(
    estimate.merkleTreeCreationRentLamports,
  );
  const merkleTreeCreationRentSol = requiredFiniteNumber(
    estimate.merkleTreeCreationRentSol,
  );
  const merkleTreeCreationCostLamports = requiredStringValue(
    estimate.merkleTreeCreationCostLamports,
  );
  const merkleTreeCreationCostSol = requiredFiniteNumber(
    estimate.merkleTreeCreationCostSol,
  );

  const coreCollectionCreationTransactionFeeLamports =
    requiredStringValue(
      estimate.coreCollectionCreationTransactionFeeLamports,
    );
  const coreCollectionCreationTransactionFeeSol =
    requiredFiniteNumber(
      estimate.coreCollectionCreationTransactionFeeSol,
    );
  const coreCollectionCreationRentLamports = requiredStringValue(
    estimate.coreCollectionCreationRentLamports,
  );
  const coreCollectionCreationRentSol = requiredFiniteNumber(
    estimate.coreCollectionCreationRentSol,
  );
  const coreCollectionCreationCostLamports = requiredStringValue(
    estimate.coreCollectionCreationCostLamports,
  );
  const coreCollectionCreationCostSol = requiredFiniteNumber(
    estimate.coreCollectionCreationCostSol,
  );

  const provisioningCostLamports = requiredStringValue(
    estimate.provisioningCostLamports,
  );
  const provisioningCostSol = requiredFiniteNumber(
    estimate.provisioningCostSol,
  );

  const estimatedNetworkCostLamports = requiredStringValue(
    estimate.estimatedNetworkCostLamports,
  );
  const estimatedNetworkCostSol = requiredFiniteNumber(
    estimate.estimatedNetworkCostSol,
  );

  const requiredFeePayerBalanceLamports = requiredStringValue(
    estimate.requiredFeePayerBalanceLamports,
  );
  const requiredFeePayerBalanceSol = requiredFiniteNumber(
    estimate.requiredFeePayerBalanceSol,
  );

  const estimatedReserveTopUpLamports = requiredStringValue(
    estimate.estimatedReserveTopUpLamports,
  );
  const estimatedReserveTopUpSol = requiredFiniteNumber(
    estimate.estimatedReserveTopUpSol,
  );

  const reserveTransferFeeBufferLamports = requiredStringValue(
    estimate.reserveTransferFeeBufferLamports,
  );
  const reserveTransferFeeBufferSol = requiredFiniteNumber(
    estimate.reserveTransferFeeBufferSol,
  );

  const requiredReserveForTopUpLamports = requiredStringValue(
    estimate.requiredReserveForTopUpLamports,
  );
  const requiredReserveForTopUpSol = requiredFiniteNumber(
    estimate.requiredReserveForTopUpSol,
  );
  const sufficient = requiredBoolean(estimate.sufficient);

  if (
    !mintTransactionFeePerItemLamports ||
    mintTransactionFeePerItemSol === null ||
    !mintTransactionFeeTotalLamports ||
    mintTransactionFeeTotalSol === null ||
    !merkleTreeCreationTransactionFeeLamports ||
    merkleTreeCreationTransactionFeeSol === null ||
    !merkleTreeCreationRentLamports ||
    merkleTreeCreationRentSol === null ||
    !merkleTreeCreationCostLamports ||
    merkleTreeCreationCostSol === null ||
    !coreCollectionCreationTransactionFeeLamports ||
    coreCollectionCreationTransactionFeeSol === null ||
    !coreCollectionCreationRentLamports ||
    coreCollectionCreationRentSol === null ||
    !coreCollectionCreationCostLamports ||
    coreCollectionCreationCostSol === null ||
    !provisioningCostLamports ||
    provisioningCostSol === null ||
    !estimatedNetworkCostLamports ||
    estimatedNetworkCostSol === null ||
    !requiredFeePayerBalanceLamports ||
    requiredFeePayerBalanceSol === null ||
    !estimatedReserveTopUpLamports ||
    estimatedReserveTopUpSol === null ||
    !reserveTransferFeeBufferLamports ||
    reserveTransferFeeBufferSol === null ||
    !requiredReserveForTopUpLamports ||
    requiredReserveForTopUpSol === null ||
    sufficient === null
  ) {
    return null;
  }

  const sharedMerkleTreeAddress = nullableStringValue(
    resources.sharedMerkleTreeAddress,
  );
  const coreCollectionAddress = nullableStringValue(
    resources.coreCollectionAddress,
  );

  if (sharedMerkleTreeExists && !sharedMerkleTreeAddress) {
    return null;
  }

  if (coreCollectionExists && !coreCollectionAddress) {
    return null;
  }

  return {
    cluster,
    mintQuantity,
    reserve: {
      address: reserveAddress,
      balanceLamports: reserveBalanceLamports,
      balanceSol: reserveBalanceSol,
      minimumLamports: reserveMinimumLamports,
      minimumSol: reserveMinimumSol,
    },
    feePayer: {
      address: feePayerAddress,
      balanceLamports: feePayerBalanceLamports,
      balanceSol: feePayerBalanceSol,
      targetLamports: feePayerTargetLamports,
      targetSol: feePayerTargetSol,
    },
    resources: {
      sharedMerkleTreeExists,
      sharedMerkleTreeAddress,
      coreCollectionExists,
      coreCollectionAddress,
    },
    estimate: {
      mintTransactionFeePerItemLamports,
      mintTransactionFeePerItemSol,
      mintTransactionFeeTotalLamports,
      mintTransactionFeeTotalSol,
      merkleTreeCreationTransactionFeeLamports,
      merkleTreeCreationTransactionFeeSol,
      merkleTreeCreationRentLamports,
      merkleTreeCreationRentSol,
      merkleTreeCreationCostLamports,
      merkleTreeCreationCostSol,
      coreCollectionCreationTransactionFeeLamports,
      coreCollectionCreationTransactionFeeSol,
      coreCollectionCreationRentLamports,
      coreCollectionCreationRentSol,
      coreCollectionCreationCostLamports,
      coreCollectionCreationCostSol,
      provisioningCostLamports,
      provisioningCostSol,
      estimatedNetworkCostLamports,
      estimatedNetworkCostSol,
      requiredFeePayerBalanceLamports,
      requiredFeePayerBalanceSol,
      estimatedReserveTopUpLamports,
      estimatedReserveTopUpSol,
      reserveTransferFeeBufferLamports,
      reserveTransferFeeBufferSol,
      requiredReserveForTopUpLamports,
      requiredReserveForTopUpSol,
      sufficient,
    },
  };
}

/**
 * 現行BackendのGET /mint/requests responseを、
 * 既存のMintInfo取得経路と互換性のあるMintDTOへ変換する。
 *
 * Backendの正:
 * - productionId
 * - mintStatus
 * - tokenBlueprintId
 * - createdBy / createdByName
 * - requestedBy / requestedByName
 * - mintedAt
 *
 * frontendではproductionIdをMintのidとして扱う。
 */
function mergeMintDTOFromRow(
  row: MintRequestManagementRowDTO,
): MintDTO | null {
  const rawRow = row as Record<string, any>;

  const mintRaw =
    rawRow.mint &&
    typeof rawRow.mint === "object" &&
    !Array.isArray(rawRow.mint)
      ? rawRow.mint
      : null;

  const status = normalizeMintStatus(
    rawRow.mintStatus ??
      mintRaw?.status ??
      rawRow.status,
  );

  if (!status) {
    return null;
  }

  const productionId = String(
    rawRow.productionId ?? mintRaw?.id ?? "",
  ).trim();

  if (!productionId) {
    return null;
  }

  const mintProgress =
    normalizeMintProgress(rawRow.mintProgress) ??
    normalizeMintProgress(mintRaw?.mintProgress);

  const merged = {
    ...(mintRaw ?? rawRow),

    id: productionId,
    status,

    brandId:
      mintRaw?.brandId ??
      rawRow.brandId ??
      "",

    tokenBlueprintId:
      mintRaw?.tokenBlueprintId ??
      rawRow.tokenBlueprintId ??
      "",

    products:
      mintRaw?.products ??
      rawRow.products ??
      [],

    createdAt:
      mintRaw?.createdAt ??
      rawRow.createdAt ??
      "",

    createdBy:
      mintRaw?.createdBy ??
      rawRow.createdBy ??
      "",

    createdByName:
      mintRaw?.createdByName ??
      rawRow.createdByName ??
      null,

    requestedBy:
      mintRaw?.requestedBy ??
      rawRow.requestedBy ??
      null,

    requestedByName:
      mintRaw?.requestedByName ??
      rawRow.requestedByName ??
      null,

    mintedAt:
      mintRaw?.mintedAt ??
      rawRow.mintedAt ??
      null,

    scheduledBurnDate:
      mintRaw?.scheduledBurnDate ??
      rawRow.scheduledBurnDate ??
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
 * GET /mint/requestsの唯一の取得処理。
 *
 * - productionIdsは空文字と重複を除去する
 * - Backend responseは配列を正とする
 * - items / rows / dataなどの旧レスポンス形状は吸収しない
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
 * Mint detail右カラムではこのresponseを正として使用する。
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
// GET: MintDTO
// ===============================

/**
 * productionIdで1件のMintDTOを取得する。
 *
 * 現行BackendではGET /mint/requestsのmintStatusを正とする。
 * productionIdをMintのidとして扱う。
 *
 * Mint detailをmanagement row直接参照へ移行するまでの
 * 既存Application層との互換用。
 */
export async function fetchMintByProductionIdHTTP(
  productionId: string,
): Promise<MintDTO | null> {
  const row = await fetchMintRequestRowByProductionIdHTTP(
    productionId,
  );

  if (!row) {
    return null;
  }

  return mergeMintDTOFromRow(row);
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

  const estimate = normalizeMintFundingEstimate(payload);

  if (!estimate) {
    throw new Error(
      "Failed to fetch mint funding estimate: response shape is invalid",
    );
  }

  return estimate;
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