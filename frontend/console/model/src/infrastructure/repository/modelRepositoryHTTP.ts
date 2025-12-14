// frontend/console/model/src/infrastructure/repository/modelRepositoryHTTP.ts

// Firebase Auth から ID トークンを取得
import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";

// ============================================================
// Debug logging (取得したデータが分かるログ)
// ============================================================

const LOG_PREFIX = "[model/modelRepositoryHTTP]";
function log(...args: any[]) {
  // eslint-disable-next-line no-console
  console.log(LOG_PREFIX, ...args);
}
function warn(...args: any[]) {
  // eslint-disable-next-line no-console
  console.warn(LOG_PREFIX, ...args);
}
function errorLog(...args: any[]) {
  // eslint-disable-next-line no-console
  console.error(LOG_PREFIX, ...args);
}

// 🔙 BACKEND の BASE URL
const ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)?.replace(
    /\/+$/g,
    "",
  ) ?? "";

const FALLBACK_BASE =
  "https://narratives-backend-871263659099.asia-northeast1.run.app";

export const API_BASE = ENV_BASE || FALLBACK_BASE;

log("API_BASE resolved =", API_BASE, {
  ENV_BASE,
  usingFallback: !ENV_BASE,
});

// ---------------------------------------------------------
// 共通: Firebase トークン取得
// ---------------------------------------------------------
async function getIdTokenOrThrow(): Promise<string> {
  const user = auth.currentUser;
  if (!user) {
    errorLog("getIdTokenOrThrow: auth.currentUser is null (not logged in)");
    throw new Error("ログイン情報が見つかりません（未ログイン）");
  }
  // トークンそのものはログに出さない（秘匿）
  const token = await user.getIdToken();
  log("getIdTokenOrThrow: idToken acquired (masked)", {
    uid: user.uid,
    email: user.email ?? null,
  });
  return token;
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
    warn("mapRawToModelVariation: raw is not an object -> return empty", { raw });
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
    errorLog("createModelVariation: auth.currentUser is null (not logged in)");
    throw new Error("ログイン情報が見つかりません（未ログイン）");
  }
  const idToken = await user.getIdToken();

  log("createModelVariation: input", {
    productBlueprintId,
    payload: {
      ...payload,
      // measurements は大きくなりがちなので別で
      measurements: payload.measurements ? "(present)" : "(none)",
    },
    user: { uid: user.uid, email: user.email ?? null },
  });

  const cleanedMeasurements =
    payload.measurements &&
    Object.fromEntries(
      Object.entries(payload.measurements).filter(([k, v]) => {
        const ok = typeof v === "number" && Number.isFinite(v);
        if (!ok) {
          // null/undefined/NaN/非数値は送らない（何が落ちたか分かるログ）
          log("createModelVariation: drop measurement (non-number)", { key: k, value: v });
        }
        return ok;
      }),
    );

  log("createModelVariation: cleanedMeasurements", cleanedMeasurements ?? null);

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

  log("createModelVariation: request", {
    method: "POST",
    url,
    body,
    headers: {
      "Content-Type": "application/json",
      Authorization: "Bearer (masked)",
      Accept: "application/json",
    },
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

  log("createModelVariation: response", {
    ok: res.ok,
    status: res.status,
    statusText: res.statusText ?? "",
    contentType: res.headers.get("content-type"),
    bodyTextPreview: text ? text.slice(0, 500) : "",
  });

  if (!res.ok) {
    let detail: unknown = text;
    try {
      detail = text ? JSON.parse(text) : undefined;
    } catch {
      /* ignore JSON parse error */
    }
    errorLog("createModelVariation: error detail", detail);
    throw new Error(
      `モデルバリエーションの作成に失敗しました（${res.status} ${
        res.statusText ?? ""
      }）`,
    );
  }

  const raw = text ? JSON.parse(text) : {};
  log("createModelVariation: raw parsed", raw);

  const data = mapRawToModelVariation(raw);
  log("createModelVariation: mapped data", data);

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
  log("createModelVariations: start", {
    productBlueprintId,
    count: variations.length,
  });

  const results: ModelVariationResponse[] = [];

  for (const v of variations) {
    // 各要素にも productBlueprintId を補完して渡す
    const enriched: CreateModelVariationRequest = {
      ...v,
      productBlueprintId,
    };

    log("createModelVariations: creating one", {
      modelNumber: enriched.modelNumber,
      size: enriched.size,
      color: enriched.color,
      rgb: typeof enriched.rgb === "number" ? enriched.rgb : null,
      measurements: enriched.measurements ? "(present)" : "(none)",
    });

    const created = await createModelVariation(productBlueprintId, enriched);
    results.push(created);

    log("createModelVariations: created", {
      id: created.id,
      modelNumber: created.modelNumber,
      size: created.size,
      color: created.color,
    });
  }

  log("createModelVariations: done", { createdCount: results.length });
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

  log("getModelVariationById: request", {
    method: "GET",
    id,
    url,
    headers: {
      "Content-Type": "application/json",
      Authorization: "Bearer (masked)",
      Accept: "application/json",
    },
  });

  const res = await fetch(url, {
    method: "GET",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
      Accept: "application/json",
    },
  });

  const text = await res.text().catch(() => "");

  log("getModelVariationById: response", {
    ok: res.ok,
    status: res.status,
    statusText: res.statusText ?? "",
    contentType: res.headers.get("content-type"),
    bodyTextPreview: text ? text.slice(0, 500) : "",
  });

  if (!res.ok) {
    errorLog("getModelVariationById: failed", { status: res.status, text });
    throw new Error(
      `モデルバリエーションの取得に失敗しました（${res.status} ${
        res.statusText ?? ""
      }）`,
    );
  }

  const raw = text ? JSON.parse(text) : {};
  log("getModelVariationById: raw parsed", raw);

  const data = mapRawToModelVariation(raw);
  log("getModelVariationById: mapped data", data);

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

  log("listModelVariationsByProductBlueprintId: request", {
    method: "GET",
    productBlueprintId,
    url,
    headers: {
      "Content-Type": "application/json",
      Authorization: "Bearer (masked)",
      Accept: "application/json",
    },
  });

  const res = await fetch(url, {
    method: "GET",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
      Accept: "application/json",
    },
  });

  const text = await res.text().catch(() => "");

  log("listModelVariationsByProductBlueprintId: response", {
    ok: res.ok,
    status: res.status,
    statusText: res.statusText ?? "",
    contentType: res.headers.get("content-type"),
    bodyTextPreview: text ? text.slice(0, 500) : "",
  });

  if (!res.ok) {
    errorLog("listModelVariationsByProductBlueprintId: failed", {
      status: res.status,
      text,
    });
    throw new Error(
      `モデルバリエーション一覧の取得に失敗しました（${res.status} ${
        res.statusText ?? ""
      }）`,
    );
  }

  const rawList = text ? JSON.parse(text) : [];
  log("listModelVariationsByProductBlueprintId: raw parsed", {
    isArray: Array.isArray(rawList),
    length: Array.isArray(rawList) ? rawList.length : 0,
    sample0: Array.isArray(rawList) && rawList.length > 0 ? rawList[0] : null,
  });

  const list = Array.isArray(rawList)
    ? rawList.map((raw) => mapRawToModelVariation(raw))
    : [];

  log("listModelVariationsByProductBlueprintId: mapped list", {
    length: list.length,
    sample0: list.length > 0 ? list[0] : null,
    ids: list.slice(0, 10).map((v) => v.id),
  });

  return list;
}
