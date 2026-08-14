// frontend/console/shell/src/features/model/infrastructure/repository/modelRepositoryHTTP.ts

import { API_BASE } from "../../../shared/http/apiBase";
import { getAuthJsonHeadersOrThrow } from "../../../shared/http/authHeaders";

export type Volume = {
  value: number;
  unit: string;
};

/* =========================================================
 * Request types
 * =======================================================*/

export type CreateApparelModelVariationRequest = {
  kind: "apparel";
  modelNumber: string;
  size: string;
  color: string;
  rgb: number;
  measurements?: Record<string, number | null | undefined>;
};

export type CreateAlcoholModelVariationRequest = {
  kind: "alcohol";
  modelNumber: string;
  volume: Volume;
};

export type CreateModelVariationRequest =
  | CreateApparelModelVariationRequest
  | CreateAlcoholModelVariationRequest;

/* =========================================================
 * Internal request bodies
 * =======================================================*/

type CreateApparelModelVariationBody = {
  kind: "apparel";
  modelNumber: string;
  size: string;
  color: string;
  rgb: number;
  measurements?: Record<string, number>;
};

type CreateAlcoholModelVariationBody = {
  kind: "alcohol";
  modelNumber: string;
  volume: Volume;
};

type CreateModelVariationBody =
  | CreateApparelModelVariationBody
  | CreateAlcoholModelVariationBody;

type ReplaceModelVariationsBody = {
  variations: CreateModelVariationBody[];
};

/* =========================================================
 * Request helpers
 * =======================================================*/

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === "object" && !Array.isArray(value);
}

function isAlcoholCreatePayload(
  payload: CreateModelVariationRequest,
): payload is CreateAlcoholModelVariationRequest {
  return payload.kind === "alcohol";
}

function requireProductBlueprintId(productBlueprintId: string): string {
  const normalizedProductBlueprintId = productBlueprintId.trim();

  if (!normalizedProductBlueprintId) {
    throw new Error("modelRepositoryHTTP: productBlueprintIdが空です");
  }

  return normalizedProductBlueprintId;
}

function requireNonEmptyString(value: string, fieldName: string): string {
  if (typeof value !== "string" || value.length === 0) {
    throw new Error(`modelRepositoryHTTP: ${fieldName}が空です`);
  }

  return value;
}

function requireInteger(value: number, fieldName: string): number {
  if (typeof value !== "number" || !Number.isFinite(value) || !Number.isInteger(value)) {
    throw new Error(`modelRepositoryHTTP: ${fieldName}は整数である必要があります`);
  }

  return value;
}

function normalizeRGB(value: number): number {
  const rgb = requireInteger(value, "rgb");

  if (rgb < 0 || rgb > 0xffffff) {
    throw new Error("modelRepositoryHTTP: rgbは0から16777215の範囲である必要があります");
  }

  return rgb;
}

function normalizeVolume(volume: Volume): Volume {
  const value = requireInteger(volume.value, "volume.value");

  if (value <= 0) {
    throw new Error("modelRepositoryHTTP: volume.valueは1以上である必要があります");
  }

  const unit = requireNonEmptyString(volume.unit, "volume.unit");

  if (unit !== "ml" && unit !== "L") {
    throw new Error(`modelRepositoryHTTP: 未対応のvolume.unitです: ${unit}`);
  }

  return { value, unit };
}

function normalizeMeasurements(
  value?: Record<string, number | null | undefined>,
): Record<string, number> | undefined {
  if (value === undefined) return undefined;

  const measurements: Record<string, number> = {};

  for (const [key, rawValue] of Object.entries(value)) {
    if (!key) {
      throw new Error("modelRepositoryHTTP: measurementsの項目名が空です");
    }

    if (rawValue === null || rawValue === undefined) continue;

    const measurementValue = requireInteger(rawValue, `measurements.${key}`);

    if (measurementValue < 0) {
      throw new Error(`modelRepositoryHTTP: measurements.${key}は0以上である必要があります`);
    }

    measurements[key] = measurementValue;
  }

  return Object.keys(measurements).length > 0 ? measurements : undefined;
}

function toCreateRequestBody(
  payload: CreateModelVariationRequest,
): CreateModelVariationBody {
  const modelNumber = requireNonEmptyString(payload.modelNumber, "modelNumber");

  if (isAlcoholCreatePayload(payload)) {
    return {
      kind: "alcohol",
      modelNumber,
      volume: normalizeVolume(payload.volume),
    };
  }

  return {
    kind: "apparel",
    modelNumber,
    size: requireNonEmptyString(payload.size, "size"),
    color: requireNonEmptyString(payload.color, "color"),
    rgb: normalizeRGB(payload.rgb),
    measurements: normalizeMeasurements(payload.measurements),
  };
}

/* =========================================================
 * HTTP error helpers
 * =======================================================*/

function parseErrorDetail(text: string): string {
  if (!text) return "";

  try {
    const detail: unknown = JSON.parse(text);

    if (typeof detail === "string") return detail;

    if (isRecord(detail)) {
      if (typeof detail.error === "string") return detail.error;
      if (typeof detail.message === "string") return detail.message;
    }

    return JSON.stringify(detail);
  } catch {
    return text;
  }
}

function createHTTPError(
  operation: string,
  response: Response,
  text: string,
): Error {
  const detail = parseErrorDetail(text);
  const statusText = response.statusText ?? "";

  return new Error(
    `${operation} (${response.status}) ${statusText} ${detail}`.trim(),
  );
}

/* =========================================================
 * PUT /models/{productBlueprintId}/variations
 *
 * ProductBlueprint配下のModel Variationを一括置換する。
 * Backend側で既存削除と新規作成を単一transactionで処理する。
 *
 * 書き込み後の完成形はProductBlueprint Detail BFFを正とするため、
 * このAPIでは成功レスポンスをFrontendのModelへ再構築しない。
 * =======================================================*/

export async function createModelVariations(
  productBlueprintId: string,
  variations: CreateModelVariationRequest[],
): Promise<void> {
  const normalizedProductBlueprintId =
    requireProductBlueprintId(productBlueprintId);

  const url =
    `${API_BASE}/models/${encodeURIComponent(normalizedProductBlueprintId)}/variations`;

  const body: ReplaceModelVariationsBody = {
    variations: variations.map(toCreateRequestBody),
  };

  const response = await fetch(url, {
    method: "PUT",
    headers: await getAuthJsonHeadersOrThrow(),
    body: JSON.stringify(body),
  });

  if (response.ok) return;

  const text = await response.text().catch(() => "");

  throw createHTTPError(
    "モデルバリエーションの一括置換に失敗しました",
    response,
    text,
  );
}