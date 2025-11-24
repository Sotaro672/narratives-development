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

/**
 * backend/internal/domain/model.NewModelVariation に対応する想定のペイロード。
 *
 * Go 側:
 *   type NewModelVariation struct {
 *     ModelNumber  string
 *     Size         string
 *     Color        string
 *     Measurements map[string]float64
 *   }
 */
export type CreateModelVariationRequest = {
  modelNumber: string; // "LM-SB-S-WHT" など
  size: string;        // "S" / "M" / ...
  color: string;       // "ホワイト" など
  measurements?: Record<string, number | null | undefined>;
};

/**
 * 単一の ModelVariation を作成する HTTP 関数
 *
 * POST /models/{productId}/variations
 */
export async function createModelVariation(
  productId: string,
  payload: CreateModelVariationRequest,
): Promise<void> {
  const user = auth.currentUser;
  if (!user) {
    throw new Error("ログイン情報が見つかりません（未ログイン）");
  }
  const idToken = await user.getIdToken();

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
        // null / undefined は JSON から落としたいので軽くフィルタ
        measurements:
          payload.measurements &&
          Object.fromEntries(
            Object.entries(payload.measurements).filter(
              ([, v]) => typeof v === "number",
            ),
          ),
      }),
    },
  );

  if (!res.ok) {
    let detail: unknown;
    try {
      detail = await res.json();
    } catch {
      // ignore JSON parse error
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

/**
 * 複数の ModelVariation をまとめて作成するヘルパー
 * - まとめて作りたいときに productBlueprintCreateService などから呼べるようにしておく
 */
export async function createModelVariations(
  productId: string,
  variations: CreateModelVariationRequest[],
): Promise<void> {
  for (const v of variations) {
    await createModelVariation(productId, v);
  }
}
