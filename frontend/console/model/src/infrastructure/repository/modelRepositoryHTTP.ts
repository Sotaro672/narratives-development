// frontend/console/model/src/infrastructure/repository/modelRepositoryHTTP.ts

// Firebase Auth から ID トークンを取得
import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";

// 🔙 BACKEND の BASE URL
const ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)?.replace(
    /\/+$/g,
    "",
  ) ?? "";

const FALLBACK_BASE =
  "https://narratives-backend-871263659099.asia-northeast1.run.app";

export const API_BASE = ENV_BASE || FALLBACK_BASE;

/* =========================================================
 * backend/internal/domain/model.NewModelVariation に対応
 * =======================================================*/

/**
 * backend/internal/domain/model.NewModelVariation と互換
 */
export type CreateModelVariationRequest = {
  /** Firestore の productBlueprintId として保存するために必須 */
  productBlueprintId: string;
  /** モデルナンバー（例: "LM-SB-S-WHT"） */
  modelNumber: string;
  /** サイズラベル（"S" / "M" / ...） */
  size: string;
  /** カラー名（"ホワイト" など） */
  color: string;
  /** カラーの RGB 値（0xRRGGBB の int など、backend 側の仕様に合わせる） */
  rgb?: number;
  /** 採寸値（"ウエスト" など MeasurementKey の日本語ラベルをキーとする） */
  measurements?: Record<string, number | null | undefined>;
};

/* =========================================================
 * backend/internal/domain/model.ModelVariation に対応するレスポンス想定
 * =======================================================*/

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

/* =========================================================
 * 単一 ModelVariation 作成 API
 * POST /models/{productBlueprintId}/variations
 * =======================================================*/

export async function createModelVariation(
  productBlueprintId: string,
  payload: CreateModelVariationRequest,
): Promise<ModelVariationResponse> {
  const user = auth.currentUser;
  if (!user) {
    throw new Error("ログイン情報が見つかりません（未ログイン）");
  }
  const idToken = await user.getIdToken();

  const cleanedMeasurements =
    payload.measurements &&
    Object.fromEntries(
      Object.entries(payload.measurements).filter(([_, v]) => {
        return typeof v === "number" && Number.isFinite(v);
      }),
    );

  const url = `${API_BASE}/models/${encodeURIComponent(
    productBlueprintId,
  )}/variations`;

  const body: any = {
    productBlueprintId,
    modelNumber: payload.modelNumber,
    size: payload.size,
    color: payload.color,
    measurements: cleanedMeasurements,
  };

  // rgb が数値のときだけ送る（undefined の場合はフィールド自体を省略）
  if (typeof payload.rgb === "number" && Number.isFinite(payload.rgb)) {
    body.rgb = payload.rgb;
  }

  console.log("[modelRepositoryHTTP] createModelVariation request:", {
    url,
    body,
  });

  const res = await fetch(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${idToken}`,
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
      /* ignore JSON parse error */
    }
    console.error("[modelRepositoryHTTP] createModelVariation failed", {
      status: res.status,
      statusText: res.statusText,
      detail,
    });
    throw new Error(
      `モデルバリエーションの作成に失敗しました（${res.status} ${
        res.statusText ?? ""
      }）`,
    );
  }

  const data = (text ? JSON.parse(text) : {}) as ModelVariationResponse;

  console.log("[modelRepositoryHTTP] createModelVariation response:", data);

  return data;
}

/* =========================================================
 * 複数 ModelVariation の連続作成
 * createModelVariationsFromProductBlueprint() 用
 * =======================================================*/

export async function createModelVariations(
  productBlueprintId: string,
  variations: CreateModelVariationRequest[],
): Promise<ModelVariationResponse[]> {
  const results: ModelVariationResponse[] = [];

  for (const v of variations) {
    // 各要素にも productBlueprintId を補完して渡す
    const enriched: CreateModelVariationRequest = {
      ...v,
      productBlueprintId,
    };
    const created = await createModelVariation(productBlueprintId, enriched);
    results.push(created);
  }

  return results;
}
