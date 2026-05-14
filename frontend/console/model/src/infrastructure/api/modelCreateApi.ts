// frontend\console\model\src\infrastructure\api\modelCreateApi.ts
import { API_BASE } from "../../../../shell/src/shared/http/apiBase";
import {
  getAuthHeadersOrThrow,
  getAuthJsonHeadersOrThrow,
} from "../../../../shell/src/shared/http/authHeaders";

/* =========================================================
 * backend/internal/domain/model.NewModelVariation に対応
 * =======================================================*/

export type ModelVariationKind = "apparel" | "alcohol";

export type Volume = {
  value: number;
  unit: string;
};

export type CreateApparelModelVariationRequest = {
  kind: "apparel";
  productBlueprintId: string;
  modelNumber: string;
  size: string;
  color: string;
  rgb: number;
  measurements?: Record<string, number | null | undefined>;
};

export type CreateAlcoholModelVariationRequest = {
  kind: "alcohol";
  productBlueprintId: string;
  modelNumber: string;
  volume: Volume;
};

export type CreateModelVariationRequest =
  | CreateApparelModelVariationRequest
  | CreateAlcoholModelVariationRequest;

/* =========================================================
 * 正スキーマ（GET / list の正）
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
 * Helpers
 * =======================================================*/

function isRecord(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === "object" && !Array.isArray(value);
}

function isAlcoholCreatePayload(
  payload: CreateModelVariationRequest,
): payload is CreateAlcoholModelVariationRequest {
  return payload.kind === "alcohol";
}

function cleanMeasurements(
  value?: Record<string, number | null | undefined>,
): Record<string, number> | undefined {
  if (!value) return undefined;

  const out: Record<string, number> = {};

  for (const [key, rawValue] of Object.entries(value)) {
    if (!key) continue;
    if (typeof rawValue !== "number" || !Number.isFinite(rawValue)) continue;

    out[key] = rawValue;
  }

  return Object.keys(out).length > 0 ? out : undefined;
}

function parseModelVariationResponse(json: unknown): ModelVariationResponse {
  if (!isRecord(json)) {
    throw new Error("modelRepositoryHTTP: model variation response is not an object");
  }

  if (
    typeof json.id !== "string" ||
    typeof json.productBlueprintId !== "string" ||
    typeof json.kind !== "string" ||
    typeof json.modelNumber !== "string"
  ) {
    throw new Error("modelRepositoryHTTP: model variation response has invalid base fields");
  }

  const base = {
    id: json.id,
    productBlueprintId: json.productBlueprintId,
    modelNumber: json.modelNumber,
    createdAt: typeof json.createdAt === "string" ? json.createdAt : undefined,
    updatedAt: typeof json.updatedAt === "string" ? json.updatedAt : undefined,
  };

  if (json.kind === "alcohol") {
    if (!isRecord(json.volume)) {
      throw new Error("modelRepositoryHTTP: alcohol model variation response has invalid volume");
    }

    if (
      typeof json.volume.value !== "number" ||
      !Number.isFinite(json.volume.value) ||
      typeof json.volume.unit !== "string"
    ) {
      throw new Error(
        "modelRepositoryHTTP: alcohol model variation response has invalid volume fields",
      );
    }

    return {
      ...base,
      kind: "alcohol",
      volume: {
        value: json.volume.value,
        unit: json.volume.unit,
      },
    };
  }

  if (json.kind === "apparel") {
    if (typeof json.size !== "string") {
      throw new Error("modelRepositoryHTTP: apparel model variation response has invalid size");
    }

    if (!isRecord(json.color)) {
      throw new Error("modelRepositoryHTTP: apparel model variation response has invalid color");
    }

    if (
      typeof json.color.name !== "string" ||
      typeof json.color.rgb !== "number" ||
      !Number.isFinite(json.color.rgb)
    ) {
      throw new Error(
        "modelRepositoryHTTP: apparel model variation response has invalid color fields",
      );
    }

    const measurements: Record<string, number> = {};

    if (json.measurements !== undefined) {
      if (!isRecord(json.measurements)) {
        throw new Error(
          "modelRepositoryHTTP: apparel model variation response has invalid measurements",
        );
      }

      for (const [key, rawValue] of Object.entries(json.measurements)) {
        if (!key) continue;

        if (typeof rawValue !== "number" || !Number.isFinite(rawValue)) {
          throw new Error(
            "modelRepositoryHTTP: apparel model variation response has invalid measurement value",
          );
        }

        measurements[key] = rawValue;
      }
    }

    return {
      ...base,
      kind: "apparel",
      size: json.size,
      color: {
        name: json.color.name,
        rgb: json.color.rgb,
      },
      measurements:
        Object.keys(measurements).length > 0 ? measurements : undefined,
    };
  }

  throw new Error(`modelRepositoryHTTP: unsupported model variation kind: ${json.kind}`);
}

function resolveCreatedVariationId(json: unknown): string {
  if (!isRecord(json) || typeof json.id !== "string" || !json.id.trim()) {
    return "";
  }

  return json.id.trim();
}

function toCreateRequestBody(
  productBlueprintId: string,
  payload: CreateModelVariationRequest,
): Record<string, unknown> {
  if (isAlcoholCreatePayload(payload)) {
    return {
      kind: "alcohol",
      productBlueprintId,
      modelNumber: payload.modelNumber,
      volume: payload.volume,
    };
  }

  return {
    kind: "apparel",
    productBlueprintId,
    modelNumber: payload.modelNumber,
    size: payload.size,
    color: payload.color,
    rgb: payload.rgb,
    measurements: cleanMeasurements(payload.measurements),
  };
}

/* =========================================================
 * POST /models/{productBlueprintId}/variations
 * =======================================================*/

export async function createModelVariation(
  productBlueprintId: string,
  payload: CreateModelVariationRequest,
): Promise<ModelVariationResponse> {
  const normalizedProductBlueprintId = productBlueprintId.trim();

  const url = `${API_BASE}/models/${encodeURIComponent(
    normalizedProductBlueprintId,
  )}/variations`;

  const body = toCreateRequestBody(normalizedProductBlueprintId, {
    ...payload,
    productBlueprintId: normalizedProductBlueprintId,
  } as CreateModelVariationRequest);

  const res = await fetch(url, {
    method: "POST",
    headers: {
      ...(await getAuthJsonHeadersOrThrow()),
      Accept: "application/json",
    },
    body: JSON.stringify(body),
  });

  const text = await res.text().catch(() => "");

  if (!res.ok) {
    let detail: unknown = text;

    try {
      detail = text ? JSON.parse(text) : undefined;
    } catch {
      //
    }

    const msg =
      typeof detail === "string" ? detail : detail ? JSON.stringify(detail) : "";

    throw new Error(
      `モデルバリエーションの作成に失敗しました (${res.status}) ${
        res.statusText ?? ""
      } ${msg}`,
    );
  }

  const json = text ? JSON.parse(text) : {};
  const id = resolveCreatedVariationId(json);

  if (!id) {
    throw new Error("modelRepositoryHTTP: 作成成功後の model id を取得できませんでした");
  }

  return parseModelVariationResponse(json);
}

/* =========================================================
 * 複数作成
 * =======================================================*/

export async function createModelVariations(
  productBlueprintId: string,
  variations: CreateModelVariationRequest[],
): Promise<string[]> {
  const normalizedProductBlueprintId = productBlueprintId.trim();
  const ids: string[] = [];

  for (const variation of variations) {
    const created = await createModelVariation(normalizedProductBlueprintId, {
      ...variation,
      productBlueprintId: normalizedProductBlueprintId,
    } as CreateModelVariationRequest);

    const id = created.id.trim();
    if (!id) {
      throw new Error("modelRepositoryHTTP: 作成済み variation の id が空です");
    }

    ids.push(id);
  }

  return ids;
}

/* =========================================================
 * GET /models/{id}
 * =======================================================*/

export async function getModelVariationById(
  modelId: string,
): Promise<ModelVariationResponse> {
  const url = `${API_BASE}/models/${encodeURIComponent(modelId.trim())}`;

  const res = await fetch(url, {
    method: "GET",
    headers: {
      ...(await getAuthHeadersOrThrow()),
      Accept: "application/json",
    },
  });

  const text = await res.text().catch(() => "");

  if (!res.ok) {
    throw new Error(
      `モデルバリエーション取得失敗 (${res.status}) ${res.statusText}`,
    );
  }

  const json = text ? JSON.parse(text) : {};

  return parseModelVariationResponse(json);
}

/* =========================================================
 * GET /models/by-blueprint/{productBlueprintId}/variations
 * =======================================================*/

export async function listModelVariationsByProductBlueprintId(
  productBlueprintId: string,
): Promise<ModelVariationResponse[]> {
  const normalizedProductBlueprintId = productBlueprintId.trim();

  const url = `${API_BASE}/models/by-blueprint/${encodeURIComponent(
    normalizedProductBlueprintId,
  )}/variations`;

  const res = await fetch(url, {
    method: "GET",
    headers: {
      ...(await getAuthHeadersOrThrow()),
      Accept: "application/json",
    },
  });

  const text = await res.text().catch(() => "");

  if (!res.ok) {
    throw new Error(
      `モデルバリエーション一覧取得失敗 (${res.status}) ${res.statusText}`,
    );
  }

  const json = text ? JSON.parse(text) : [];

  if (!Array.isArray(json)) {
    throw new Error("modelRepositoryHTTP: model variation list response is not an array");
  }

  return json.map(parseModelVariationResponse);
}