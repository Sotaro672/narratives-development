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

// ---------------------------------------------------------
// 共通: Firebase トークン取得
// ---------------------------------------------------------
async function getIdTokenOrThrow(): Promise<string> {
  const user = auth.currentUser;
  if (!user) {
    throw new Error("ログイン情報が見つかりません（未ログイン）");
  }
  return user.getIdToken();
}

/* =========================================================
 * backend/internal/domain/model.NewModelVariation に対応
 * （dto を正: camelCase / rgb 必須）
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
  /** カラーの RGB 値（0xRRGGBB の int）。✅ 必須（0=黒も正） */
  rgb: number;
  /** 採寸値（"ウエスト" など MeasurementKey の日本語ラベルをキーとする） */
  measurements?: Record<string, number | null | undefined>;
};

/* =========================================================
 * backend/internal/domain/model.ModelVariation に対応するレスポンス想定
 * （dto を正: camelCase / color.rgb 必須）
 * =======================================================*/

export type ModelVariationResponse = {
  id: string;
  productBlueprintId: string;
  modelNumber: string;
  size: string;
  color: {
    name: string;
    rgb: number; // ✅ 必須（0=黒も正）
  };
  measurements?: Record<string, number>;
  createdAt?: string | null;
  createdBy?: string | null;
  updatedAt?: string | null;
  updatedBy?: string | null;
};

/**
 * レスポンス JSON から variation id を抽出（キー揺れ吸収）
 * さらに Location ヘッダからもフォールバックする。
 */
function extractVariationId(json: any, locationHeader?: string | null): string {
  const raw =
    json?.id ??
    json?.ID ??
    json?.docId ??
    json?.docID ??
    json?.modelId ??
    json?.modelID ??
    json?.variationId ??
    json?.variationID;

  const idFromJson = typeof raw === "string" ? raw.trim() : "";
  if (idFromJson) return idFromJson;

  // Location: /models/{id} あるいは .../models/{id} のような形式を想定
  const loc = typeof locationHeader === "string" ? locationHeader.trim() : "";
  if (loc) {
    const m = loc.match(/\/models\/([^/?#]+)(?:[/?#]|$)/);
    if (m?.[1]) return decodeURIComponent(m[1]).trim();
  }

  return "";
}

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
      Object.entries(payload.measurements).filter(([, v]) => {
        const ok = typeof v === "number" && Number.isFinite(v);
        return ok;
      }),
    );

  const url = `${API_BASE}/models/${encodeURIComponent(
    productBlueprintId,
  )}/variations`;

  // dto を正: camelCase / rgb 必須
  const body = {
    productBlueprintId,
    modelNumber: payload.modelNumber,
    size: payload.size,
    color: payload.color,
    rgb: payload.rgb, // ✅ 常に送る（0=黒も正）
    measurements: cleanedMeasurements,
  };

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
    const detailMsg =
      typeof detail === "string" ? detail : detail ? JSON.stringify(detail) : "";
    throw new Error(
      `モデルバリエーションの作成に失敗しました（${res.status} ${
        res.statusText ?? ""
      }）${detailMsg ? `: ${detailMsg}` : ""}`,
    );
  }

  // ここが今回の最重要：id を必ず抽出する
  const jsonAny = text ? (JSON.parse(text) as any) : {};
  const id = extractVariationId(jsonAny, res.headers.get("Location"));

  if (!id) {
    // 作成自体は成功している前提なので、レスポンス仕様不備を明確化
    // サーバー側修正（id を返す）を促すため、body も付けて投げる
    throw new Error(
      `modelRepositoryHTTP: ModelVariation は作成されましたが id が返りませんでした（response=${text || "{}"}）`,
    );
  }

  // 返ってきた JSON を優先しつつ、id だけは必ず保証する
  return {
    ...(jsonAny as any),
    id,
  } as ModelVariationResponse;
}

/* =========================================================
 * 複数 ModelVariation の連続作成
 * createModelVariationsFromProductBlueprint() 用
 *
 * ★返り値を modelIds(string[]) に統一（要件）
 * =======================================================*/

export async function createModelVariations(
  productBlueprintId: string,
  variations: CreateModelVariationRequest[],
): Promise<string[]> {
  const ids: string[] = [];

  for (const v of variations) {
    // 各要素にも productBlueprintId を補完して渡す
    const enriched: CreateModelVariationRequest = {
      ...v,
      productBlueprintId,
    };

    const created = await createModelVariation(productBlueprintId, enriched);

    const id = String((created as any)?.id ?? "").trim();
    if (!id) {
      // createModelVariation が id 保証するので通常ここには来ないが、念のため
      throw new Error(
        "modelRepositoryHTTP: ModelVariation は作成されましたが id を抽出できませんでした",
      );
    }

    ids.push(id);
  }

  return ids;
}

/* =========================================================
 * 単一 ModelVariation 取得 API
 * GET /models/{id}
 * =======================================================*/

export async function getModelVariationById(
  id: string,
): Promise<ModelVariationResponse> {
  const token = await getIdTokenOrThrow();
  const safeId = encodeURIComponent(id.trim());

  const url = `${API_BASE}/models/${safeId}`;

  const res = await fetch(url, {
    method: "GET",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
      Accept: "application/json",
    },
  });

  const text = await res.text().catch(() => "");

  if (!res.ok) {
    throw new Error(
      `モデルバリエーションの取得に失敗しました（${res.status} ${
        res.statusText ?? ""
      }）`,
    );
  }

  const jsonAny = (text ? JSON.parse(text) : {}) as any;
  const extractedId = extractVariationId(jsonAny, res.headers.get("Location"));
  const finalId = extractedId || String(jsonAny?.id ?? "").trim();

  if (!finalId) {
    throw new Error(
      `modelRepositoryHTTP: getModelVariationById のレスポンスに id がありません（response=${text || "{}"}）`,
    );
  }

  return {
    ...(jsonAny as any),
    id: finalId,
  } as ModelVariationResponse;
}

/* =========================================================
 * Blueprint 単位での ModelVariation 一覧取得
 * GET /models/by-blueprint/{productBlueprintId}/variations
 * =======================================================*/

export async function listModelVariationsByProductBlueprintId(
  productBlueprintId: string,
): Promise<ModelVariationResponse[]> {
  const token = await getIdTokenOrThrow();
  const safeId = encodeURIComponent(productBlueprintId.trim());

  const url = `${API_BASE}/models/by-blueprint/${safeId}/variations`;

  const res = await fetch(url, {
    method: "GET",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
      Accept: "application/json",
    },
  });

  const text = await res.text().catch(() => "");

  if (!res.ok) {
    throw new Error(
      `モデルバリエーション一覧の取得に失敗しました（${res.status} ${
        res.statusText ?? ""
      }）`,
    );
  }

  const data = (text ? JSON.parse(text) : []) as any[];
  if (!Array.isArray(data)) return [];

  // 一覧系も id 揺れを吸収して正規化しておく（後段の型崩れ防止）
  return data
    .map((row) => {
      const id = extractVariationId(row, null);
      if (!id) return null;
      return { ...(row as any), id } as ModelVariationResponse;
    })
    .filter(Boolean) as ModelVariationResponse[];
}
