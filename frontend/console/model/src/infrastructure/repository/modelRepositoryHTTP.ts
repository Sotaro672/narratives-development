// frontend/console/model/src/infrastructure/repository/modelRepositoryHTTP.ts

import { API_BASE } from "../../../../shell/src/shared/http/apiBase";
import {
  getAuthHeadersOrThrow,
  getAuthJsonHeadersOrThrow,
} from "../../../../shell/src/shared/http/authHeaders";

/* =========================================================
 * backend/internal/domain/model.NewModelVariation に対応
 * =======================================================*/

export type CreateModelVariationRequest = {
  productBlueprintId: string;
  modelNumber: string;
  size: string;
  color: string;
  rgb: number;
  measurements?: Record<string, number | null | undefined>;
};

/* =========================================================
 * 正スキーマ（frontend内部では camelCase を正とする）
 * =======================================================*/

export type ModelVariationResponse = {
  id: string;
  productBlueprintId: string;
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

function normalizeMeasurements(
  value: unknown,
): Record<string, number> | undefined {
  if (!value || typeof value !== "object") return undefined;

  const out: Record<string, number> = {};
  for (const [k, v] of Object.entries(value as Record<string, unknown>)) {
    if (typeof v === "number" && Number.isFinite(v)) {
      out[k] = v;
    }
  }

  return Object.keys(out).length > 0 ? out : undefined;
}

function normalizeModelVariationResponse(
  json: unknown,
  fallback: {
    productBlueprintId: string;
    modelNumber?: string;
    size?: string;
    colorName?: string;
    colorRgb?: number;
    measurements?: Record<string, number>;
  },
): ModelVariationResponse {
  const anyJson = (json ?? {}) as Record<string, any>;

  const colorObj =
    anyJson?.color && typeof anyJson.color === "object"
      ? anyJson.color
      : anyJson?.Color && typeof anyJson.Color === "object"
        ? anyJson.Color
        : undefined;

  const id =
    typeof anyJson?.id === "string" && anyJson.id.trim()
      ? anyJson.id.trim()
      : typeof anyJson?.ID === "string" && anyJson.ID.trim()
        ? anyJson.ID.trim()
        : "";

  const productBlueprintId =
    typeof anyJson?.productBlueprintId === "string" &&
    anyJson.productBlueprintId.trim()
      ? anyJson.productBlueprintId.trim()
      : typeof anyJson?.ProductBlueprintID === "string" &&
          anyJson.ProductBlueprintID.trim()
        ? anyJson.ProductBlueprintID.trim()
        : fallback.productBlueprintId;

  const modelNumber =
    typeof anyJson?.modelNumber === "string"
      ? String(anyJson.modelNumber)
      : typeof anyJson?.ModelNumber === "string"
        ? String(anyJson.ModelNumber)
        : String(fallback.modelNumber ?? "");

  const size =
    typeof anyJson?.size === "string"
      ? String(anyJson.size)
      : typeof anyJson?.Size === "string"
        ? String(anyJson.Size)
        : String(fallback.size ?? "");

  const colorName =
    typeof colorObj?.name === "string"
      ? String(colorObj.name)
      : typeof colorObj?.Name === "string"
        ? String(colorObj.Name)
        : String(fallback.colorName ?? "");

  const colorRgb =
    typeof colorObj?.rgb === "number" && Number.isFinite(colorObj.rgb)
      ? Number(colorObj.rgb)
      : typeof colorObj?.RGB === "number" && Number.isFinite(colorObj.RGB)
        ? Number(colorObj.RGB)
        : Number(fallback.colorRgb ?? 0);

  const measurements =
    normalizeMeasurements(anyJson?.measurements) ??
    normalizeMeasurements(anyJson?.Measurements) ??
    fallback.measurements;

  const createdAt =
    typeof anyJson?.createdAt === "string"
      ? String(anyJson.createdAt)
      : typeof anyJson?.CreatedAt === "string"
        ? String(anyJson.CreatedAt)
        : undefined;

  const updatedAt =
    typeof anyJson?.updatedAt === "string"
      ? String(anyJson.updatedAt)
      : typeof anyJson?.UpdatedAt === "string"
        ? String(anyJson.UpdatedAt)
        : undefined;

  return {
    id,
    productBlueprintId,
    modelNumber,
    size,
    color: {
      name: colorName,
      rgb: colorRgb,
    },
    measurements,
    createdAt,
    updatedAt,
  };
}

/* =========================================================
 * POST /models/{productBlueprintId}/variations
 * =======================================================*/

export async function createModelVariation(
  productBlueprintId: string,
  payload: CreateModelVariationRequest,
): Promise<ModelVariationResponse> {
  const cleanedMeasurements =
    payload.measurements &&
    Object.fromEntries(
      Object.entries(payload.measurements).filter(
        ([, v]) => typeof v === "number" && Number.isFinite(v),
      ),
    );

  const url = `${API_BASE}/models/${encodeURIComponent(
    productBlueprintId,
  )}/variations`;

  const body = {
    productBlueprintId,
    modelNumber: payload.modelNumber,
    size: payload.size,
    color: payload.color,
    rgb: payload.rgb,
    measurements: cleanedMeasurements,
  };

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

  const normalized = normalizeModelVariationResponse(json, {
    productBlueprintId,
    modelNumber: payload.modelNumber,
    size: payload.size,
    colorName: payload.color,
    colorRgb: payload.rgb,
    measurements: cleanedMeasurements as Record<string, number> | undefined,
  });

  if (!normalized.id) {
    throw new Error("modelRepositoryHTTP: 作成成功後の model id を取得できませんでした");
  }

  return normalized;
}

/* =========================================================
 * 複数作成
 * =======================================================*/

export async function createModelVariations(
  productBlueprintId: string,
  variations: CreateModelVariationRequest[],
): Promise<string[]> {
  const ids: string[] = [];

  for (const v of variations) {
    const created = await createModelVariation(productBlueprintId, {
      ...v,
      productBlueprintId,
    });

    const id = String(created.id ?? "").trim();
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

  return normalizeModelVariationResponse(json, {
    productBlueprintId: "",
  });
}

/* =========================================================
 * GET /models/by-blueprint/{productBlueprintId}/variations
 * =======================================================*/

export async function listModelVariationsByProductBlueprintId(
  productBlueprintId: string,
): Promise<ModelVariationResponse[]> {
  const url = `${API_BASE}/models/by-blueprint/${encodeURIComponent(
    productBlueprintId.trim(),
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
    return [];
  }

  return json.map((item) =>
    normalizeModelVariationResponse(item, {
      productBlueprintId,
    }),
  );
}