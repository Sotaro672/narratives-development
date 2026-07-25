// frontend/console/model/src/infrastructure/api/modelUpdateApi.ts
/// <reference types="vite/client" />

import { API_BASE } from "../../../../shell/src/shared/http/apiBase";
import {
  getAuthHeadersOrThrow,
  getAuthJsonHeadersOrThrow,
} from "../../../../shell/src/shared/http/authHeaders";

/* =========================================================
 * Types
 * =======================================================*/

export type ModelVariationKind = "apparel" | "alcohol";

export type Volume = {
  value: number;
  unit: string;
};

export type ApparelModelVariationUpdateRequest = {
  kind: "apparel";
  /** モデル番号 (例: "LM-SB-S-WHT") */
  modelNumber: string;
  /** サイズ (例: "S" / "M" / ...) */
  size: string;
  /** カラー名 (例: "ホワイト") */
  color: string;
  /** RGB 値 (0xRRGGBB 想定。0=黒も正) */
  rgb: number;
  /** 着丈 / 身幅 / 股下などの採寸値マップ */
  measurements?: Record<string, number>;
};

export type AlcoholModelVariationUpdateRequest = {
  kind: "alcohol";
  /** モデル番号 (例: "SAKE-720") */
  modelNumber: string;
  /** 容量 */
  volume: Volume;
};

/**
 * ModelVariation 更新リクエスト
 */
export type ModelVariationUpdateRequest =
  | ApparelModelVariationUpdateRequest
  | AlcoholModelVariationUpdateRequest;

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

/**
 * ModelVariation のレスポンス
 * backend response の camelCase を正とする
 */
export type ModelVariationResponse =
  | ApparelModelVariationResponse
  | AlcoholModelVariationResponse;

/* =========================================================
 * Helpers
 * =======================================================*/

function isRecord(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === "object" && !Array.isArray(value);
}

function isAlcoholUpdatePayload(
  payload: ModelVariationUpdateRequest,
): payload is AlcoholModelVariationUpdateRequest {
  return payload.kind === "alcohol";
}

function cleanMeasurements(
  value?: Record<string, number>,
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

function toUpdateRequestBody(
  payload: ModelVariationUpdateRequest,
): Record<string, unknown> {
  if (isAlcoholUpdatePayload(payload)) {
    return {
      kind: "alcohol",
      modelNumber: payload.modelNumber,
      volume: payload.volume,
    };
  }

  return {
    kind: "apparel",
    modelNumber: payload.modelNumber,
    size: payload.size,
    color: payload.color,
    rgb: payload.rgb,
    measurements: cleanMeasurements(payload.measurements),
  };
}

function parseModelVariationResponse(json: unknown): ModelVariationResponse {
  if (!isRecord(json)) {
    throw new Error("modelUpdateApi: model variation response is not an object");
  }

  if (
    typeof json.id !== "string" ||
    typeof json.productBlueprintId !== "string" ||
    typeof json.kind !== "string" ||
    typeof json.modelNumber !== "string"
  ) {
    throw new Error(
      "modelUpdateApi: model variation response has invalid base fields",
    );
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
      throw new Error(
        "modelUpdateApi: alcohol model variation response has invalid volume",
      );
    }

    if (
      typeof json.volume.value !== "number" ||
      !Number.isFinite(json.volume.value) ||
      typeof json.volume.unit !== "string"
    ) {
      throw new Error(
        "modelUpdateApi: alcohol model variation response has invalid volume fields",
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
      throw new Error(
        "modelUpdateApi: apparel model variation response has invalid size",
      );
    }

    if (!isRecord(json.color)) {
      throw new Error(
        "modelUpdateApi: apparel model variation response has invalid color",
      );
    }

    if (
      typeof json.color.name !== "string" ||
      typeof json.color.rgb !== "number" ||
      !Number.isFinite(json.color.rgb)
    ) {
      throw new Error(
        "modelUpdateApi: apparel model variation response has invalid color fields",
      );
    }

    const measurements: Record<string, number> = {};

    if (json.measurements !== undefined) {
      if (!isRecord(json.measurements)) {
        throw new Error(
          "modelUpdateApi: apparel model variation response has invalid measurements",
        );
      }

      for (const [key, rawValue] of Object.entries(json.measurements)) {
        if (!key) continue;

        if (typeof rawValue !== "number" || !Number.isFinite(rawValue)) {
          throw new Error(
            "modelUpdateApi: apparel model variation response has invalid measurement value",
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

  throw new Error(`modelUpdateApi: unsupported model variation kind: ${json.kind}`);
}

/* =========================================================
 * PUT /models/{id}
 * =======================================================*/

/**
 * モデルバリエーションの更新 API
 */
export async function updateModelVariation(
  variationId: string,
  payload: ModelVariationUpdateRequest,
): Promise<ModelVariationResponse> {
  const id = variationId.trim();
  if (!id) {
    throw new Error("variationId が空です");
  }

  const url = `${API_BASE}/models/${encodeURIComponent(id)}`;
  const body = toUpdateRequestBody(payload);

  const res = await fetch(url, {
    method: "PUT",
    headers: {
      ...(await getAuthJsonHeadersOrThrow()),
      Accept: "application/json",
    },
    body: JSON.stringify(body),
  });

  const text = await res.text().catch(() => "");

  if (!res.ok) {
    throw new Error(
      `モデルバリエーションの更新に失敗しました（${res.status} ${res.statusText}）: ${text}`,
    );
  }

  const json = text ? JSON.parse(text) : null;
  if (!json) {
    throw new Error("モデルバリエーション更新レスポンスが空です");
  }

  return parseModelVariationResponse(json);
}

/* =========================================================
 * DELETE /models/{id}
 * =======================================================*/

/**
 * ModelVariation 削除 API
 */
export async function deleteModelVariation(
  variationId: string,
): Promise<void> {
  const id = variationId.trim();
  if (!id) {
    throw new Error("variationId が空です");
  }

  const url = `${API_BASE}/models/${encodeURIComponent(id)}`;

  const res = await fetch(url, {
    method: "DELETE",
    headers: {
      ...(await getAuthHeadersOrThrow()),
      Accept: "application/json",
    },
  });

  if (res.status === 404) return;

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(
      `モデルバリエーションの削除に失敗しました（${res.status} ${res.statusText}）: ${text}`,
    );
  }
}