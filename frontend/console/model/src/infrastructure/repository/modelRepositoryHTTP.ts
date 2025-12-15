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
};

// Firestore / Go 構造体からの生 JSON をフロント用に正規化するヘルパー
function mapRawToModelVariation(raw: any): ModelVariationResponse {
  if (!raw || typeof raw !== "object") {
    return {
      id: "",
      productBlueprintId: "",
      modelNumber: "",
      size: "",
      color: { name: "", rgb: null },
      measurements: {},
      createdAt: null,
      createdBy: null,
      updatedAt: null,
      updatedBy: null,
    };
  }

  const id = raw.id ?? raw.ID ?? "";
  const productBlueprintId =
    raw.productBlueprintId ?? raw.ProductBlueprintID ?? "";
  const modelNumber = raw.modelNumber ?? raw.ModelNumber ?? "";
  const size = raw.size ?? raw.Size ?? "";

  // Color 構造体のケースいろいろを吸収
  const colorObj = raw.color ?? raw.Color ?? null;

  const colorName =
    colorObj?.name ?? colorObj?.Name ?? raw.colorName ?? raw.ColorName ?? "";
  const colorRgb =
    colorObj?.rgb ?? colorObj?.RGB ?? raw.rgb ?? raw.RGB ?? null;

  const measurements = raw.measurements ?? raw.Measurements ?? undefined;

  const createdAt = raw.createdAt ?? raw.CreatedAt ?? null;
  const createdBy = raw.createdBy ?? raw.CreatedBy ?? null;
  const updatedAt = raw.updatedAt ?? raw.UpdatedAt ?? null;
  const updatedBy = raw.updatedBy ?? raw.UpdatedBy ?? null;

  const normalized: ModelVariationResponse = {
    id,
    productBlueprintId,
    modelNumber,
    size,
    color: {
      name: colorName,
      rgb: colorRgb,
    },
    measurements,
    createdAt,
    createdBy,
    updatedAt,
    updatedBy,
  };

  return normalized;
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
    // detail は握りつぶさずにエラー文面に残す（ただし UI 表示に不要ならここは消してOK）
    const detailMsg =
      typeof detail === "string" ? detail : detail ? JSON.stringify(detail) : "";
    throw new Error(
      `モデルバリエーションの作成に失敗しました（${res.status} ${
        res.statusText ?? ""
      }）${detailMsg ? `: ${detailMsg}` : ""}`,
    );
  }

  const raw = text ? JSON.parse(text) : {};
  const data = mapRawToModelVariation(raw);

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

  const raw = text ? JSON.parse(text) : {};
  const data = mapRawToModelVariation(raw);

  return data;
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

  const rawList = text ? JSON.parse(text) : [];

  const list = Array.isArray(rawList)
    ? rawList.map((raw) => mapRawToModelVariation(raw))
    : [];

  return list;
}
