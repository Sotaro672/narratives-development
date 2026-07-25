// frontend/console/model/src/infrastructure/repository/http/modelRepositoryHTTP.ts

import { API_BASE } from "../../../../shell/src/shared/http/apiBase";
import {
  getAuthHeadersOrThrow,
  getAuthJsonHeadersOrThrow,
} from "../../../../shell/src/shared/http/authHeaders";

/* =========================================================
 * Request types
 * =======================================================*/

export type ModelVariationKind = "apparel" | "alcohol";

export type Volume = {
  value: number;
  unit: string;
};

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
 * Response types
 * =======================================================*/

export type ApparelModelVariationResponse = {
  id: string;
  productBlueprintId: string;
  kind: "apparel";
  modelNumber: string;
  size: string;
  color: {
    name: string;
    rgb: number;
  };
  measurements?: Record<string, number>;
  createdAt?: string;
  updatedAt?: string;
};

export type AlcoholModelVariationResponse = {
  id: string;
  productBlueprintId: string;
  kind: "alcohol";
  modelNumber: string;
  volume: Volume;
  createdAt?: string;
  updatedAt?: string;
};

export type ModelVariationResponse =
  | ApparelModelVariationResponse
  | AlcoholModelVariationResponse;

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
 * Generic helpers
 * =======================================================*/

function isRecord(value: unknown): value is Record<string, unknown> {
  return (
    value !== null &&
    typeof value === "object" &&
    !Array.isArray(value)
  );
}

function isAlcoholCreatePayload(
  payload: CreateModelVariationRequest,
): payload is CreateAlcoholModelVariationRequest {
  return payload.kind === "alcohol";
}

function requireProductBlueprintId(
  productBlueprintId: string,
): string {
  const normalizedProductBlueprintId = productBlueprintId.trim();

  if (!normalizedProductBlueprintId) {
    throw new Error(
      "modelRepositoryHTTP: productBlueprintIdが空です",
    );
  }

  return normalizedProductBlueprintId;
}

function requireModelId(modelId: string): string {
  const normalizedModelId = modelId.trim();

  if (!normalizedModelId) {
    throw new Error(
      "modelRepositoryHTTP: modelIdが空です",
    );
  }

  return normalizedModelId;
}

function requireNonEmptyString(
  value: string,
  fieldName: string,
): string {
  if (typeof value !== "string" || value.length === 0) {
    throw new Error(
      `modelRepositoryHTTP: ${fieldName}が空です`,
    );
  }

  return value;
}

function requireInteger(
  value: number,
  fieldName: string,
): number {
  if (
    typeof value !== "number" ||
    !Number.isFinite(value) ||
    !Number.isInteger(value)
  ) {
    throw new Error(
      `modelRepositoryHTTP: ${fieldName}は整数である必要があります`,
    );
  }

  return value;
}

function normalizeRGB(value: number): number {
  const rgb = requireInteger(value, "rgb");

  if (rgb < 0 || rgb > 0xffffff) {
    throw new Error(
      "modelRepositoryHTTP: rgbは0から16777215の範囲である必要があります",
    );
  }

  return rgb;
}

function normalizeVolume(volume: Volume): Volume {
  const value = requireInteger(
    volume.value,
    "volume.value",
  );

  if (value <= 0) {
    throw new Error(
      "modelRepositoryHTTP: volume.valueは1以上である必要があります",
    );
  }

  const unit = requireNonEmptyString(
    volume.unit,
    "volume.unit",
  );

  if (unit !== "ml" && unit !== "L") {
    throw new Error(
      `modelRepositoryHTTP: 未対応のvolume.unitです: ${unit}`,
    );
  }

  return {
    value,
    unit,
  };
}

function normalizeMeasurements(
  value?: Record<string, number | null | undefined>,
): Record<string, number> | undefined {
  if (value === undefined) {
    return undefined;
  }

  const measurements: Record<string, number> = {};

  for (const [key, rawValue] of Object.entries(value)) {
    if (!key) {
      throw new Error(
        "modelRepositoryHTTP: measurementsの項目名が空です",
      );
    }

    if (rawValue === null || rawValue === undefined) {
      continue;
    }

    const measurementValue = requireInteger(
      rawValue,
      `measurements.${key}`,
    );

    if (measurementValue < 0) {
      throw new Error(
        `modelRepositoryHTTP: measurements.${key}は0以上である必要があります`,
      );
    }

    measurements[key] = measurementValue;
  }

  if (Object.keys(measurements).length === 0) {
    return undefined;
  }

  return measurements;
}

function toCreateRequestBody(
  payload: CreateModelVariationRequest,
): CreateModelVariationBody {
  const modelNumber = requireNonEmptyString(
    payload.modelNumber,
    "modelNumber",
  );

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
    size: requireNonEmptyString(
      payload.size,
      "size",
    ),
    color: requireNonEmptyString(
      payload.color,
      "color",
    ),
    rgb: normalizeRGB(payload.rgb),
    measurements: normalizeMeasurements(
      payload.measurements,
    ),
  };
}

/* =========================================================
 * Response envelope helpers
 * =======================================================*/

function unwrapSingleResponse(value: unknown): unknown {
  if (!isRecord(value)) {
    return value;
  }

  if (isRecord(value.modelVariation)) {
    return value.modelVariation;
  }

  if (isRecord(value.data)) {
    if (isRecord(value.data.modelVariation)) {
      return value.data.modelVariation;
    }

    if (
      typeof value.data.id === "string" ||
      typeof value.data.kind === "string"
    ) {
      return value.data;
    }
  }

  return value;
}

function unwrapListResponse(value: unknown): unknown[] {
  if (Array.isArray(value)) {
    return value;
  }

  if (!isRecord(value)) {
    throw new Error(
      "modelRepositoryHTTP: model variation list responseが配列ではありません",
    );
  }

  const directCandidates: unknown[] = [
    value.variations,
    value.modelVariations,
    value.items,
  ];

  for (const candidate of directCandidates) {
    if (Array.isArray(candidate)) {
      return candidate;
    }
  }

  if (Array.isArray(value.data)) {
    return value.data;
  }

  if (isRecord(value.data)) {
    const dataCandidates: unknown[] = [
      value.data.variations,
      value.data.modelVariations,
      value.data.items,
    ];

    for (const candidate of dataCandidates) {
      if (Array.isArray(candidate)) {
        return candidate;
      }
    }
  }

  throw new Error(
    "modelRepositoryHTTP: model variation list responseに配列がありません",
  );
}

function optionalResponseString(
  value: unknown,
): string | undefined {
  return typeof value === "string"
    ? value
    : undefined;
}

function parseResponseMeasurements(
  value: unknown,
): Record<string, number> | undefined {
  if (value === undefined || value === null) {
    return undefined;
  }

  if (!isRecord(value)) {
    throw new Error(
      "modelRepositoryHTTP: measurements responseがobjectではありません",
    );
  }

  const measurements: Record<string, number> = {};

  for (const [key, rawValue] of Object.entries(value)) {
    if (!key) {
      throw new Error(
        "modelRepositoryHTTP: measurements responseの項目名が空です",
      );
    }

    if (
      typeof rawValue !== "number" ||
      !Number.isFinite(rawValue) ||
      !Number.isInteger(rawValue) ||
      rawValue < 0
    ) {
      throw new Error(
        `modelRepositoryHTTP: measurements.${key} responseが不正です`,
      );
    }

    measurements[key] = rawValue;
  }

  if (Object.keys(measurements).length === 0) {
    return undefined;
  }

  return measurements;
}

function parseResponseColor(
  json: Record<string, unknown>,
): {
  name: string;
  rgb: number;
} {
  if (isRecord(json.color)) {
    const name = json.color.name;
    const rgb = json.color.rgb;

    if (
      typeof name !== "string" ||
      !name ||
      typeof rgb !== "number" ||
      !Number.isFinite(rgb) ||
      !Number.isInteger(rgb) ||
      rgb < 0 ||
      rgb > 0xffffff
    ) {
      throw new Error(
        "modelRepositoryHTTP: apparel color responseが不正です",
      );
    }

    return {
      name,
      rgb,
    };
  }

  if (
    typeof json.color === "string" &&
    json.color &&
    typeof json.rgb === "number" &&
    Number.isFinite(json.rgb) &&
    Number.isInteger(json.rgb) &&
    json.rgb >= 0 &&
    json.rgb <= 0xffffff
  ) {
    return {
      name: json.color,
      rgb: json.rgb,
    };
  }

  throw new Error(
    "modelRepositoryHTTP: apparel color responseがありません",
  );
}

function parseResponseVolume(
  value: unknown,
): Volume {
  if (!isRecord(value)) {
    throw new Error(
      "modelRepositoryHTTP: alcohol volume responseがobjectではありません",
    );
  }

  if (
    typeof value.value !== "number" ||
    !Number.isFinite(value.value) ||
    !Number.isInteger(value.value) ||
    value.value <= 0
  ) {
    throw new Error(
      "modelRepositoryHTTP: alcohol volume.value responseが不正です",
    );
  }

  if (
    typeof value.unit !== "string" ||
    (value.unit !== "ml" && value.unit !== "L")
  ) {
    throw new Error(
      "modelRepositoryHTTP: alcohol volume.unit responseが不正です",
    );
  }

  return {
    value: value.value,
    unit: value.unit,
  };
}

function parseModelVariationResponse(
  value: unknown,
  fallbackProductBlueprintId?: string,
): ModelVariationResponse {
  const unwrapped = unwrapSingleResponse(value);

  if (!isRecord(unwrapped)) {
    throw new Error(
      "modelRepositoryHTTP: model variation responseがobjectではありません",
    );
  }

  const id = unwrapped.id;
  const kind = unwrapped.kind;
  const modelNumber = unwrapped.modelNumber;

  const productBlueprintId =
    typeof unwrapped.productBlueprintId === "string"
      ? unwrapped.productBlueprintId
      : fallbackProductBlueprintId;

  if (
    typeof id !== "string" ||
    !id ||
    typeof productBlueprintId !== "string" ||
    !productBlueprintId ||
    typeof kind !== "string" ||
    typeof modelNumber !== "string" ||
    !modelNumber
  ) {
    throw new Error(
      "modelRepositoryHTTP: model variation responseの共通項目が不正です",
    );
  }

  const base = {
    id,
    productBlueprintId,
    modelNumber,
    createdAt: optionalResponseString(
      unwrapped.createdAt,
    ),
    updatedAt: optionalResponseString(
      unwrapped.updatedAt,
    ),
  };

  if (kind === "alcohol") {
    return {
      ...base,
      kind: "alcohol",
      volume: parseResponseVolume(
        unwrapped.volume,
      ),
    };
  }

  if (kind === "apparel") {
    if (
      typeof unwrapped.size !== "string" ||
      !unwrapped.size
    ) {
      throw new Error(
        "modelRepositoryHTTP: apparel size responseが不正です",
      );
    }

    return {
      ...base,
      kind: "apparel",
      size: unwrapped.size,
      color: parseResponseColor(unwrapped),
      measurements: parseResponseMeasurements(
        unwrapped.measurements,
      ),
    };
  }

  throw new Error(
    `modelRepositoryHTTP: 未対応のkindです: ${kind}`,
  );
}

function parseModelVariationListResponse(
  value: unknown,
  fallbackProductBlueprintId?: string,
): ModelVariationResponse[] {
  return unwrapListResponse(value).map(
    (variation) =>
      parseModelVariationResponse(
        variation,
        fallbackProductBlueprintId,
      ),
  );
}

/* =========================================================
 * Response ID helpers
 * =======================================================*/

function responseIdCandidate(
  value: unknown,
): string {
  if (!isRecord(value)) {
    return "";
  }

  if (typeof value.id === "string" && value.id.trim()) {
    return value.id.trim();
  }

  if (isRecord(value.modelVariation)) {
    const modelVariationId =
      responseIdCandidate(value.modelVariation);

    if (modelVariationId) {
      return modelVariationId;
    }
  }

  if (isRecord(value.data)) {
    const dataId = responseIdCandidate(value.data);

    if (dataId) {
      return dataId;
    }
  }

  return "";
}

function responseLocationId(
  locationHeader: string | null,
): string {
  const location = String(locationHeader ?? "").trim();

  if (!location) {
    return "";
  }

  const match = location.match(
    /\/models\/([^/?#]+)(?:[/?#]|$)/,
  );

  if (!match?.[1]) {
    return "";
  }

  return decodeURIComponent(match[1]).trim();
}

function parseResponseIds(
  value: unknown,
): string[] | undefined {
  if (Array.isArray(value)) {
    if (
      value.every(
        (item) =>
          typeof item === "string" &&
          item.length > 0,
      )
    ) {
      return value;
    }

    return undefined;
  }

  if (!isRecord(value)) {
    return undefined;
  }

  if (Array.isArray(value.ids)) {
    if (
      value.ids.every(
        (item) =>
          typeof item === "string" &&
          item.length > 0,
      )
    ) {
      return value.ids;
    }

    return undefined;
  }

  if (isRecord(value.data)) {
    return parseResponseIds(value.data);
  }

  return undefined;
}

/* =========================================================
 * HTTP response helpers
 * =======================================================*/

function parseErrorDetail(text: string): string {
  if (!text) {
    return "";
  }

  try {
    const detail: unknown = JSON.parse(text);

    if (typeof detail === "string") {
      return detail;
    }

    if (isRecord(detail)) {
      if (typeof detail.error === "string") {
        return detail.error;
      }

      if (typeof detail.message === "string") {
        return detail.message;
      }
    }

    return JSON.stringify(detail);
  } catch {
    return text;
  }
}

function parseJSONResponseText(
  text: string,
): unknown {
  try {
    return JSON.parse(text);
  } catch {
    throw new Error(
      "modelRepositoryHTTP: Backend responseをJSONとして解析できませんでした",
    );
  }
}

function createHTTPError(
  operation: string,
  response: Response,
  text: string,
): Error {
  const detail = parseErrorDetail(text);

  return new Error(
    `${operation} (${response.status}) ${
      response.statusText ?? ""
    } ${detail}`,
  );
}

/* =========================================================
 * POST /models/{productBlueprintId}/variations
 * =======================================================*/

export async function createModelVariation(
  productBlueprintId: string,
  payload: CreateModelVariationRequest,
): Promise<ModelVariationResponse> {
  const normalizedProductBlueprintId =
    requireProductBlueprintId(productBlueprintId);

  const url = `${API_BASE}/models/${encodeURIComponent(
    normalizedProductBlueprintId,
  )}/variations`;

  const body = toCreateRequestBody(payload);

  const response = await fetch(url, {
    method: "POST",
    headers: {
      ...(await getAuthJsonHeadersOrThrow()),
      Accept: "application/json",
    },
    body: JSON.stringify(body),
  });

  const text = await response.text().catch(() => "");

  if (!response.ok) {
    throw createHTTPError(
      "モデルバリエーションの作成に失敗しました",
      response,
      text,
    );
  }

  const json: unknown = text
    ? parseJSONResponseText(text)
    : undefined;

  if (json !== undefined) {
    try {
      return parseModelVariationResponse(
        json,
        normalizedProductBlueprintId,
      );
    } catch (parseError) {
      const responseId =
        responseIdCandidate(json) ||
        responseLocationId(
          response.headers.get("Location"),
        );

      if (responseId) {
        return getModelVariationById(responseId);
      }

      throw parseError;
    }
  }

  const responseId = responseLocationId(
    response.headers.get("Location"),
  );

  if (!responseId) {
    throw new Error(
      "modelRepositoryHTTP: 作成後のmodel variation IDを取得できませんでした",
    );
  }

  return getModelVariationById(responseId);
}

/* =========================================================
 * PUT /models/{productBlueprintId}/variations
 *
 * 複数variationを単一requestで原子的に置換する。
 * Backendでは既存削除と新規作成を単一transactionで処理する。
 * =======================================================*/

export async function createModelVariations(
  productBlueprintId: string,
  variations: CreateModelVariationRequest[],
): Promise<string[]> {
  const normalizedProductBlueprintId =
    requireProductBlueprintId(productBlueprintId);

  const url = `${API_BASE}/models/${encodeURIComponent(
    normalizedProductBlueprintId,
  )}/variations`;

  const body: ReplaceModelVariationsBody = {
    variations: variations.map(
      (variation) =>
        toCreateRequestBody(variation),
    ),
  };

  const response = await fetch(url, {
    method: "PUT",
    headers: {
      ...(await getAuthJsonHeadersOrThrow()),
      Accept: "application/json",
    },
    body: JSON.stringify(body),
  });

  const text = await response.text().catch(() => "");

  if (!response.ok) {
    throw createHTTPError(
      "モデルバリエーションの一括置換に失敗しました",
      response,
      text,
    );
  }

  if (!text) {
    const replaced =
      await listModelVariationsByProductBlueprintId(
        normalizedProductBlueprintId,
      );

    if (replaced.length !== variations.length) {
      throw new Error(
        "modelRepositoryHTTP: 一括置換後の件数がrequest件数と一致しません",
      );
    }

    return replaced.map((variation) => variation.id);
  }

  const json = parseJSONResponseText(text);

  const responseIds = parseResponseIds(json);

  if (responseIds) {
    if (responseIds.length !== variations.length) {
      throw new Error(
        "modelRepositoryHTTP: 一括置換responseのID件数がrequest件数と一致しません",
      );
    }

    return responseIds;
  }

  const replaced =
    parseModelVariationListResponse(
      json,
      normalizedProductBlueprintId,
    );

  if (replaced.length !== variations.length) {
    throw new Error(
      "modelRepositoryHTTP: 一括置換responseの件数がrequest件数と一致しません",
    );
  }

  const ids = replaced.map(
    (variation) => variation.id,
  );

  if (ids.some((id) => !id)) {
    throw new Error(
      "modelRepositoryHTTP: 一括置換responseに空のmodel variation IDがあります",
    );
  }

  return ids;
}

/* =========================================================
 * GET /models/{id}
 * =======================================================*/

export async function getModelVariationById(
  modelId: string,
): Promise<ModelVariationResponse> {
  const normalizedModelId = requireModelId(modelId);

  const url = `${API_BASE}/models/${encodeURIComponent(
    normalizedModelId,
  )}`;

  const response = await fetch(url, {
    method: "GET",
    headers: {
      ...(await getAuthHeadersOrThrow()),
      Accept: "application/json",
    },
  });

  const text = await response.text().catch(() => "");

  if (!response.ok) {
    throw createHTTPError(
      "モデルバリエーション取得失敗",
      response,
      text,
    );
  }

  if (!text) {
    throw new Error(
      "modelRepositoryHTTP: model variation取得responseが空です",
    );
  }

  return parseModelVariationResponse(
    parseJSONResponseText(text),
  );
}

/* =========================================================
 * GET /models/by-blueprint/{productBlueprintId}/variations
 * =======================================================*/

export async function listModelVariationsByProductBlueprintId(
  productBlueprintId: string,
): Promise<ModelVariationResponse[]> {
  const normalizedProductBlueprintId =
    requireProductBlueprintId(productBlueprintId);

  const url = `${API_BASE}/models/by-blueprint/${encodeURIComponent(
    normalizedProductBlueprintId,
  )}/variations`;

  const response = await fetch(url, {
    method: "GET",
    headers: {
      ...(await getAuthHeadersOrThrow()),
      Accept: "application/json",
    },
  });

  const text = await response.text().catch(() => "");

  if (!response.ok) {
    throw createHTTPError(
      "モデルバリエーション一覧取得失敗",
      response,
      text,
    );
  }

  if (!text) {
    return [];
  }

  return parseModelVariationListResponse(
    parseJSONResponseText(text),
    normalizedProductBlueprintId,
  );
}