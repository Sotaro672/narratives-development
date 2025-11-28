/// <reference types="vite/client" />

import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";
import { API_BASE } from "../repository/modelRepositoryHTTP";

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
 */
export type ModelVariationResponse = {
  id: string;
  productBlueprintId: string;
  modelNumber: string;
  size: string;
  color: {
    name: string;
    rgb?: number | null;
  };
  measurements?: Record<string, number | null>;
  createdAt?: string | null;
  createdBy?: string | null;
  updatedAt?: string | null;
  updatedBy?: string | null;
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

  const user = auth.currentUser;
  if (!user) {
    throw new Error("ログイン情報が見つかりません（未ログイン）");
  }

  const idToken = await user.getIdToken();
  const url = `${API_BASE}/models/${encodeURIComponent(id)}`;

  const body: any = {
    modelNumber: payload.modelNumber,
    size: payload.size,
    color: payload.color,
    rgb: payload.rgb,
    measurements: payload.measurements,
  };

  const res = await fetch(url, {
    method: "PUT",
    headers: {
      Authorization: `Bearer ${idToken}`,
      "Content-Type": "application/json",
      Accept: "application/json",
    },
    body: JSON.stringify(body),
  });

  // ★ 404 → 「存在なし」とみなして疑似レスポンス返却
  if (res.status === 404) {
    const dummy: ModelVariationResponse = {
      id,
      productBlueprintId: "",
      modelNumber: payload.modelNumber,
      size: payload.size,
      color: {
        name: payload.color,
        rgb: payload.rgb ?? null,
      },
      measurements: payload.measurements ?? {},
      createdAt: null,
      createdBy: null,
      updatedAt: null,
      updatedBy: null,
    };
    return dummy;
  }

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(
      `モデルバリエーションの更新に失敗しました（${res.status} ${res.statusText}）: ${text}`,
    );
  }

  const data = (await res.json()) as ModelVariationResponse;
  return data;
}

/**
 * ModelVariation 削除 API
 *
 * DELETE /models/{id}
 */
export async function deleteModelVariation(variationId: string): Promise<void> {
  const id = variationId.trim();
  if (!id) {
    throw new Error("variationId が空です");
  }

  const user = auth.currentUser;
  if (!user) {
    throw new Error("ログイン情報が見つかりません（未ログイン）");
  }

  const idToken = await user.getIdToken();
  const url = `${API_BASE}/models/${encodeURIComponent(id)}`;

  const res = await fetch(url, {
    method: "DELETE",
    headers: {
      Authorization: `Bearer ${idToken}`,
      Accept: "application/json",
    },
  });

  // 404 → 既にないので成功扱い
  if (res.status === 404) return;

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(
      `モデルバリエーションの削除に失敗しました（${res.status} ${res.statusText}）: ${text}`,
    );
  }
}
