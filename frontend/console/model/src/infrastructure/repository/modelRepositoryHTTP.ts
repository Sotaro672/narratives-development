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
 * backend/internal/domain/model.NewModelVariation
 *
 * Go 側:
 *   type NewModelVariation struct {
 *     ModelNumber  string
 *     Size         string
 *     Color        string
 *     Measurements map[string]float64
 *   }
 *
 * ※ modelCreateService.tsx の NewModelVariationPayload と互換
 */
export type CreateModelVariationRequest = {
  modelNumber: string; // "LM-SB-S-WHT"
  size: string;        // "S" / "M" / ...
  color: string;       // "ホワイト"
  measurements?: Record<string, number | null | undefined>;
};

/* =========================================================
 * 単一 ModelVariation 作成 API
 * POST /models/{productId}/variations
 * =======================================================*/

export async function createModelVariation(
  productId: string,
  payload: CreateModelVariationRequest,
): Promise<void> {
  const user = auth.currentUser;
  if (!user) {
    throw new Error("ログイン情報が見つかりません（未ログイン）");
  }
  const idToken = await user.getIdToken();

  const cleanedMeasurements =
    payload.measurements &&
    Object.fromEntries(
      Object.entries(payload.measurements).filter(
        ([, v]) => typeof v === "number",
      ),
    );

  const res = await fetch(
    `${API_BASE}/models/${encodeURIComponent(productId)}/variations`,
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${idToken}`,
      },
      body: JSON.stringify({
        modelNumber: payload.modelNumber,
        size: payload.size,
        color: payload.color,
        measurements: cleanedMeasurements,
      }),
    },
  );

  if (!res.ok) {
    let detail: unknown;
    try {
      detail = await res.json();
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
}

/* =========================================================
 * 複数 ModelVariation の連続作成
 * createModelVariationsFromProductBlueprint() 用
 * =======================================================*/

export async function createModelVariations(
  productId: string,
  variations: CreateModelVariationRequest[],
): Promise<void> {
  for (const v of variations) {
    await createModelVariation(productId, v);
  }
}
