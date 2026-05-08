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
 * 正スキーマ（GET / list の正）
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

/**
 * POST /models/{productBlueprintId}/variations の成功レスポンスから id を決定する
 * - まず JSON body の id を使う
 * - 無ければ Location ヘッダ (/models/{id}) から補完する
 */
function resolveCreatedVariationId(
  json: unknown,
  locationHeader: string | null,
): string {
  const bodyId =
    typeof (json as any)?.id === "string" ? String((json as any).id).trim() : "";
  if (bodyId) return bodyId;

  const location = String(locationHeader ?? "").trim();
  if (!location) return "";

  const match = location.match(/\/models\/([^/?#]+)(?:[/?#]|$)/);
  if (!match?.[1]) return "";

  return decodeURIComponent(match[1]).trim();
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
  const id = resolveCreatedVariationId(json, res.headers.get("Location"));

  if (!id) {
    throw new Error("modelRepositoryHTTP: 作成成功後の model id を取得できませんでした");
  }

  return {
    ...(json as object),
    id,
    productBlueprintId:
      typeof (json as any)?.productBlueprintId === "string" &&
      String((json as any).productBlueprintId).trim()
        ? String((json as any).productBlueprintId).trim()
        : productBlueprintId,
    modelNumber:
      typeof (json as any)?.modelNumber === "string"
        ? String((json as any).modelNumber)
        : payload.modelNumber,
    size:
      typeof (json as any)?.size === "string"
        ? String((json as any).size)
        : payload.size,
    color:
      (json as any)?.color &&
      typeof (json as any).color === "object" &&
      typeof (json as any).color.name === "string" &&
      typeof (json as any).color.rgb === "number"
        ? {
            name: String((json as any).color.name),
            rgb: Number((json as any).color.rgb),
          }
        : {
            name: payload.color,
            rgb: payload.rgb,
          },
    measurements:
      (json as any)?.measurements &&
      typeof (json as any).measurements === "object"
        ? ((json as any).measurements as Record<string, number>)
        : (cleanedMeasurements as Record<string, number> | undefined),
    createdAt:
      typeof (json as any)?.createdAt === "string"
        ? String((json as any).createdAt)
        : undefined,
    updatedAt:
      typeof (json as any)?.updatedAt === "string"
        ? String((json as any).updatedAt)
        : undefined,
  };
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

  return json as ModelVariationResponse;
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

  return json as ModelVariationResponse[];
}