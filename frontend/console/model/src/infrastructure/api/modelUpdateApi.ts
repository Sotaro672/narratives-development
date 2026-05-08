/// <reference types="vite/client" />

import { API_BASE } from "../../../../shell/src/shared/http/apiBase";
import {
  getAuthHeadersOrThrow,
  getAuthJsonHeadersOrThrow,
} from "../../../../shell/src/shared/http/authHeaders";

/**
 * ModelVariation 更新リクエスト
 */
export type ModelVariationUpdateRequest = {
  /** モデル番号 (例: "LM-SB-S-WHT") */
  modelNumber: string;
  /** サイズ (例: "S" / "M" / ...) */
  size: string;
  /** カラー名 (例: "ホワイト") */
  color: string;
  /** RGB 値 (0xRRGGBB 想定) */
  rgb?: number;
  /** 着丈 / 身幅 / 股下などの採寸値マップ */
  measurements?: Record<string, number>;
};

/**
 * ModelVariation のレスポンス
 * 正スキーマに統一
 */
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
 * モデルバリエーションの更新 API
 *
 * PUT /models/{id}
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

  const body = {
    modelNumber: payload.modelNumber,
    size: payload.size,
    color: payload.color,
    rgb: payload.rgb,
    measurements: payload.measurements,
  };

  const res = await fetch(url, {
    method: "PUT",
    headers: {
      ...(await getAuthJsonHeadersOrThrow()),
      Accept: "application/json",
    },
    body: JSON.stringify(body),
  });

  if (res.status === 404) {
    return {
      id,
      productBlueprintId: "",
      modelNumber: payload.modelNumber,
      size: payload.size,
      color: {
        name: payload.color,
        rgb: payload.rgb ?? 0,
      },
      measurements: payload.measurements ?? {},
      createdAt: "",
      updatedAt: "",
    };
  }

  const text = await res.text().catch(() => "");

  if (!res.ok) {
    throw new Error(
      `モデルバリエーションの更新に失敗しました（${res.status} ${res.statusText}）: ${text}`,
    );
  }

  const data = text ? (JSON.parse(text) as ModelVariationResponse) : null;
  if (!data) {
    throw new Error("モデルバリエーション更新レスポンスが空です");
  }

  return data;
}

/**
 * ModelVariation 削除 API
 *
 * DELETE /models/{id}
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