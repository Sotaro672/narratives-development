// frontend/console/model/src/application/modelUpdateService.tsx

/// <reference types="vite/client" />

import { auth } from "../../../shell/src/auth/infrastructure/config/firebaseClient";
import { API_BASE } from "../infrastructure/repository/modelRepositoryHTTP";

/**
 * ModelVariation 更新リクエスト
 * backend の createModelVariationRequest と同じ構造を想定
 *
 *   type createModelVariationRequest struct {
 *     ProductBlueprintID string             `json:"productBlueprintId,omitempty"`
 *     ModelNumber        string             `json:"modelNumber"`
 *     Size               string             `json:"size"`
 *     Color              string             `json:"color"`
 *     RGB                int                `json:"rgb,omitempty"`
 *     Measurements       map[string]float64 `json:"measurements,omitempty"`
 *   }
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
  /** 着丈 / 身幅 / 股下 などの採寸値マップ（キーは日本語ラベル） */
  measurements?: Record<string, number>;
};

/**
 * backend/internal/domain/model/model.go の ModelVariation に対応するレスポンス想定
 * 必要に応じて実際の struct に合わせてフィールドを追加してください。
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
  deletedAt?: string | null;
  deletedBy?: string | null;
};

/**
 * モデルバリエーションを更新するサービス
 *
 * 呼び出し先:
 *   PUT /models/{id}
 * Handler:
 *   backend/internal/adapters/in/http/handlers/model_handler.go の updateVariation
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

  const body = {
    // backend の createModelVariationRequest に合わせてキー名を変換
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

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(
      `モデルバリエーションの更新に失敗しました（${res.status} ${res.statusText}）: ${text}`,
    );
  }

  const data = (await res.json()) as ModelVariationResponse;

  // ★ 保存後に backend から受け取ったデータのスナップショットをログ出力
  console.log("[updateModelVariation] response data:", {
    variationId: id,
    response: data,
  });

  return data;
}
