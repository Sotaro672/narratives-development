// frontend/console/shell/src/features/model/infrastructure/repository/modelRepositoryHTTP.ts

import { API_BASE } from "../../../../shared/http/apiBase";
import {
  getAuthHeadersOrThrow,
  getAuthJsonHeadersOrThrow,
} from "../../../../shared/http/authHeaders";

import {
  parseModelVariationListResponse,
  type ModelVariationResponse,
  type Volume,
} from "../codec/modelVariationCodec";

export type {
  ModelVariationKind,
  ModelVariationColor,
  ApparelModelVariationResponse,
  AlcoholModelVariationResponse,
  ModelVariationResponse,
  Volume,
} from "../codec/modelVariationCodec";

/* =========================================================
 * Request types
 * =======================================================*/

export type CreateApparelModelVariationRequest = {
  kind: "apparel";
  modelNumber: string;
  size: string;
  color: string;
  rgb: number;
  measurements?: Record<
    string,
    number | null | undefined
  >;
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
 * Generic helpers
 * =======================================================*/

function isRecord(
  value: unknown,
): value is Record<string, unknown> {
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
  const normalizedProductBlueprintId =
    productBlueprintId.trim();

  if (!normalizedProductBlueprintId) {
    throw new Error(
      "modelRepositoryHTTP: productBlueprintIdが空です",
    );
  }

  return normalizedProductBlueprintId;
}

function requireNonEmptyString(
  value: string,
  fieldName: string,
): string {
  if (
    typeof value !== "string" ||
    value.length === 0
  ) {
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

function normalizeRGB(
  value: number,
): number {
  const rgb = requireInteger(
    value,
    "rgb",
  );

  if (
    rgb < 0 ||
    rgb > 0xffffff
  ) {
    throw new Error(
      "modelRepositoryHTTP: rgbは0から16777215の範囲である必要があります",
    );
  }

  return rgb;
}

function normalizeVolume(
  volume: Volume,
): Volume {
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

  if (
    unit !== "ml" &&
    unit !== "L"
  ) {
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
  value?: Record<
    string,
    number | null | undefined
  >,
): Record<string, number> | undefined {
  if (value === undefined) {
    return undefined;
  }

  const measurements: Record<string, number> = {};

  for (
    const [key, rawValue]
    of Object.entries(value)
  ) {
    if (!key) {
      throw new Error(
        "modelRepositoryHTTP: measurementsの項目名が空です",
      );
    }

    if (
      rawValue === null ||
      rawValue === undefined
    ) {
      continue;
    }

    const measurementValue =
      requireInteger(
        rawValue,
        `measurements.${key}`,
      );

    if (measurementValue < 0) {
      throw new Error(
        `modelRepositoryHTTP: measurements.${key}は0以上である必要があります`,
      );
    }

    measurements[key] =
      measurementValue;
  }

  if (
    Object.keys(measurements).length === 0
  ) {
    return undefined;
  }

  return measurements;
}

function toCreateRequestBody(
  payload: CreateModelVariationRequest,
): CreateModelVariationBody {
  const modelNumber =
    requireNonEmptyString(
      payload.modelNumber,
      "modelNumber",
    );

  if (isAlcoholCreatePayload(payload)) {
    return {
      kind: "alcohol",
      modelNumber,
      volume: normalizeVolume(
        payload.volume,
      ),
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

    rgb: normalizeRGB(
      payload.rgb,
    ),

    measurements:
      normalizeMeasurements(
        payload.measurements,
      ),
  };
}

/* =========================================================
 * Response helpers
 * =======================================================*/

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
    return parseResponseIds(
      value.data,
    );
  }

  return undefined;
}

/* =========================================================
 * HTTP response helpers
 * =======================================================*/

function parseErrorDetail(
  text: string,
): string {
  if (!text) {
    return "";
  }

  try {
    const detail: unknown =
      JSON.parse(text);

    if (typeof detail === "string") {
      return detail;
    }

    if (isRecord(detail)) {
      if (
        typeof detail.error === "string"
      ) {
        return detail.error;
      }

      if (
        typeof detail.message === "string"
      ) {
        return detail.message;
      }
    }

    return JSON.stringify(
      detail,
    );
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
  const detail =
    parseErrorDetail(text);

  const statusText =
    response.statusText ?? "";

  return new Error(
    `${operation} (${response.status}) ${statusText} ${detail}`.trim(),
  );
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
    requireProductBlueprintId(
      productBlueprintId,
    );

  const url =
    `${API_BASE}/models/${encodeURIComponent(
      normalizedProductBlueprintId,
    )}/variations`;

  const body: ReplaceModelVariationsBody = {
    variations:
      variations.map(
        (variation) =>
          toCreateRequestBody(
            variation,
          ),
      ),
  };

  const response = await fetch(
    url,
    {
      method: "PUT",

      headers: {
        ...(
          await getAuthJsonHeadersOrThrow()
        ),

        Accept: "application/json",
      },

      body: JSON.stringify(
        body,
      ),
    },
  );

  const text =
    await response
      .text()
      .catch(() => "");

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

    if (
      replaced.length !==
      variations.length
    ) {
      throw new Error(
        "modelRepositoryHTTP: 一括置換後の件数がrequest件数と一致しません",
      );
    }

    return replaced.map(
      (variation) =>
        variation.id,
    );
  }

  const json =
    parseJSONResponseText(
      text,
    );

  const responseIds =
    parseResponseIds(
      json,
    );

  if (responseIds) {
    if (
      responseIds.length !==
      variations.length
    ) {
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

  if (
    replaced.length !==
    variations.length
  ) {
    throw new Error(
      "modelRepositoryHTTP: 一括置換responseの件数がrequest件数と一致しません",
    );
  }

  return replaced.map(
    (variation) =>
      variation.id,
  );
}

/* =========================================================
 * GET /models/by-blueprint/{productBlueprintId}/variations
 * =======================================================*/

export async function listModelVariationsByProductBlueprintId(
  productBlueprintId: string,
): Promise<ModelVariationResponse[]> {
  const normalizedProductBlueprintId =
    requireProductBlueprintId(
      productBlueprintId,
    );

  const url =
    `${API_BASE}/models/by-blueprint/${encodeURIComponent(
      normalizedProductBlueprintId,
    )}/variations`;

  const response = await fetch(
    url,
    {
      method: "GET",

      headers: {
        ...(
          await getAuthHeadersOrThrow()
        ),

        Accept: "application/json",
      },
    },
  );

  const text =
    await response
      .text()
      .catch(() => "");

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
    parseJSONResponseText(
      text,
    ),
    normalizedProductBlueprintId,
  );
}