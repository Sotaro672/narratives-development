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
  /** 現在の version（あれば +1 して送信される） */
  version?: number;
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
  /** 現在の version（backend 側の struct に合わせて利用） */
  version?: number;
};

/**
 * 現在の ModelVariation を取得して version を知るためのヘルパー
 */
async function fetchCurrentModelVariation(
  variationId: string,
  idToken: string,
): Promise<ModelVariationResponse | null> {
  const id = variationId.trim();
  if (!id) return null;

  const url = `${API_BASE}/models/${encodeURIComponent(id)}`;
  const res = await fetch(url, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${idToken}`,
      Accept: "application/json",
    },
  });

  if (!res.ok) {
    return null;
  }

  const data = (await res.json()) as ModelVariationResponse;
  return data;
}

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

  // 1. 現在の version を取得し、nextVersion を決定
  let nextVersion: number | undefined = undefined;

  // まず backend に保存されている現在の version を優先して利用
  try {
    const current = await fetchCurrentModelVariation(id, idToken);
    if (current && typeof current.version === "number") {
      nextVersion = current.version + 1;
    }
  } catch {
    // 取得に失敗した場合は payload.version にフォールバック
  }

  // backend から取れなかった場合、payload.version を元に +1
  if (
    nextVersion === undefined &&
    typeof payload.version === "number" &&
    !Number.isNaN(payload.version)
  ) {
    nextVersion = payload.version + 1;
  }

  const url = `${API_BASE}/models/${encodeURIComponent(id)}`;

  const body: any = {
    // backend の createModelVariationRequest に合わせてキー名を変換
    modelNumber: payload.modelNumber,
    size: payload.size,
    color: payload.color,
    rgb: payload.rgb,
    measurements: payload.measurements,
  };

  // version が決まっていれば送信（backend 側で Version フィールドとして利用想定）
  if (typeof nextVersion === "number") {
    body.version = nextVersion;
  }

  const res = await fetch(url, {
    method: "PUT",
    headers: {
      Authorization: `Bearer ${idToken}`,
      "Content-Type": "application/json",
      Accept: "application/json",
    },
    body: JSON.stringify(body),
  });

  // ★ 404 の場合は「その variation は存在しない」とみなしてスキップ扱いにする
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
      deletedAt: null,
      deletedBy: null,
      version: nextVersion,
    };

    return dummy;
  }

  // 404 以外のエラー時だけ text を読む（JSON をパースする前に body を消費しない）
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
 * モデルバリエーションを削除するサービス
 *
 * 呼び出し先:
 *   DELETE /models/{id}
 * Handler:
 *   backend/internal/adapters/in/http/handlers/model_handler.go の deleteVariation
 *
 * 呼び出し元（例）:
 *   - サイズ削除時 / カラー削除時に、対応する variation を物理削除したい場合
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

  // ★ 404 の場合は「既に存在しない」とみなして成功扱いにする
  if (res.status === 404) {
    return;
  }

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(
      `モデルバリエーションの削除に失敗しました（${res.status} ${res.statusText}）: ${text}`,
    );
  }

  // 正常時はレスポンスボディは特に利用しない
}
