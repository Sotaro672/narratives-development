// frontend/console/list/src/infrastructure/http/listRepositoryHTTP.tsx

// Firebase Auth から ID トークンを取得
import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";

/**
 * Backend base URL
 */
const ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)?.replace(
    /\/+$/g,
    "",
  ) ?? "";

const FALLBACK_BASE =
  "https://narratives-backend-871263659099.asia-northeast1.run.app";

export const API_BASE = ENV_BASE || FALLBACK_BASE;

// eslint-disable-next-line no-console
console.log("[list/listRepositoryHTTP] API_BASE resolved =", API_BASE, {
  ENV_BASE,
  usingFallback: !ENV_BASE,
});

/**
 * ===========
 * Types
 * ===========
 * ※ backend の List エンティティに完全一致しなくてもOK（必要な分だけ）
 */

export type CreateListInput = {
  // backend が docId を要求する場合に備えて（未指定なら inventoryId を採用）
  id?: string;

  // ルート（inventory/list/create から作成する想定）
  inventoryId?: string;

  // UI 入力
  title: string;
  description: string;

  // PriceCard の rows（UI 側は保持していてOK / backend には modelId + price のみ送る）
  priceRows?: Array<{
    modelId?: string;
    price: number | null;

    // UI 用（backend には送らない）
    size: string;
    color: string;
    stock: number;
    rgb?: number | null;
  }>;

  // 画面の「出品｜保留」（※ create payload には送らない）
  decision?: "list" | "hold";

  // 担当者など（必要に応じて）
  assigneeId?: string;

  // 作成者など（バックエンドで auth から取るなら省略可）
  createdBy?: string;
};

export type ListDTO = Record<string, any>;
export type ListAggregateDTO = Record<string, any>;
export type ListImageDTO = Record<string, any>;

/**
 * ===========
 * Internal helpers
 * ===========
 */

function s(v: unknown): string {
  return String(v ?? "").trim();
}

function toNumberOrNull(v: unknown): number | null {
  if (v === null || v === undefined) return null;
  const n = Number(v);
  if (!Number.isFinite(n)) return null;
  return n;
}

/**
 * ✅ create 用の prices を正規化する（modelId + price ONLY）
 *
 * - modelId が無い行があれば例外（送信しない）
 * - price が null / NaN なら例外（Go 側が非nullableの可能性が高い）
 */
function normalizePricesForBackend(
  rows: CreateListInput["priceRows"],
): Array<{ modelId: string; price: number }> {
  if (!Array.isArray(rows)) return [];

  return rows.map((r, i) => {
    const modelId = s((r as any)?.modelId);
    const priceMaybe = toNumberOrNull((r as any)?.price);

    if (!modelId) {
      // eslint-disable-next-line no-console
      console.error("[list/listRepositoryHTTP] priceRows row missing modelId", {
        index: i,
        row: r,
      });
      throw new Error("missing_modelId_in_priceRows");
    }

    if (priceMaybe === null) {
      // eslint-disable-next-line no-console
      console.error("[list/listRepositoryHTTP] priceRows row missing price", {
        index: i,
        row: r,
        modelId,
        rawPrice: (r as any)?.price,
      });
      throw new Error("missing_price_in_priceRows");
    }

    return { modelId, price: priceMaybe };
  });
}

/**
 * ✅ CreateList payload（最小）
 * - 「create時に送るのは modelId と price」の方針を厳守
 * - decision/status/priceRows 等は送らない（DisallowUnknownFields対策）
 */
function buildCreateListPayloadArray(input: CreateListInput): Record<string, any> {
  const u = auth.currentUser;
  const uid = s(u?.uid);

  const inventoryId = s(input?.inventoryId);
  const id = s(input?.id) || inventoryId;

  if (!id) {
    // eslint-disable-next-line no-console
    console.error("[list/listRepositoryHTTP] missing id (and inventoryId)", {
      input,
      inventoryId,
      id,
    });
    throw new Error("missing_id");
  }

  const title = s(input?.title);
  if (!title) {
    // eslint-disable-next-line no-console
    console.error("[list/listRepositoryHTTP] missing title", { input });
    throw new Error("missing_title");
  }

  const prices = normalizePricesForBackend(input?.priceRows);

  return {
    id,
    inventoryId,
    title,
    description: String(input?.description ?? ""),
    assigneeId: s(input?.assigneeId) || undefined,
    createdBy: s(input?.createdBy) || uid || "system",

    // ✅ backendへ送るのは modelId + price のみ
    prices, // Array<{modelId, price}>
  };
}

/**
 * ✅ fallback: prices を map で送る版
 * backend が `map[string]number` を期待している場合に通る
 */
function buildCreateListPayloadMap(input: CreateListInput): Record<string, any> {
  const base = buildCreateListPayloadArray(input);
  const pricesArray = Array.isArray((base as any).prices) ? ((base as any).prices as any[]) : [];

  const pricesMap: Record<string, number> = {};
  for (const p of pricesArray) {
    const modelId = s((p as any)?.modelId);
    const price = Number((p as any)?.price);
    if (!modelId || !Number.isFinite(price)) continue;
    pricesMap[modelId] = price;
  }

  return {
    ...base,
    prices: pricesMap, // Record<string, number>
  };
}

async function getIdToken(): Promise<string> {
  const u = auth.currentUser;
  if (!u) throw new Error("not_authenticated");
  return await u.getIdToken();
}

async function requestJSON<T>(args: {
  method: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  path: string;
  body?: unknown;
}): Promise<T> {
  const token = await getIdToken();
  const url = `${API_BASE}${args.path.startsWith("/") ? "" : "/"}${args.path}`;

  let bodyText: string | undefined = undefined;
  if (args.body !== undefined) {
    try {
      bodyText = JSON.stringify(args.body);
    } catch (e) {
      // eslint-disable-next-line no-console
      console.error("[list/listRepositoryHTTP] JSON.stringify failed", {
        method: args.method,
        url,
        body: args.body,
        error: String(e instanceof Error ? e.message : e),
        raw: e,
      });
      throw new Error("invalid_json_stringify");
    }
  }

  const bodyJsonLen = bodyText ? bodyText.length : 0;
  const bodyJsonPreview =
    bodyText && bodyText.length > 3000 ? bodyText.slice(0, 3000) + "...(truncated)" : bodyText;

  // eslint-disable-next-line no-console
  console.log("[list/listRepositoryHTTP] request", {
    method: args.method,
    url,
    bodyText,
    bodyJsonLen,
    bodyJsonPreview,
  });

  const res = await fetch(url, {
    method: args.method,
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
    body: bodyText,
  });

  const text = await res.text();
  let json: any = null;
  try {
    json = text ? JSON.parse(text) : null;
  } catch {
    json = { raw: text };
  }

  // eslint-disable-next-line no-console
  console.log("[list/listRepositoryHTTP] response", {
    method: args.method,
    url,
    status: res.status,
    ok: res.ok,
    keys: json && typeof json === "object" ? Object.keys(json) : [],
    body: json,
  });

  if (!res.ok) {
    const msg =
      (json && typeof json === "object" && (json.error || json.message)) ||
      `http_error_${res.status}`;

    // eslint-disable-next-line no-console
    console.error("[list/listRepositoryHTTP] http error", {
      method: args.method,
      url,
      status: res.status,
      message: String(msg),
      body: json,
      requestBodyPreview: bodyJsonPreview,
    });

    throw new Error(String(msg));
  }

  return json as T;
}

function extractItemsArrayFromAny(json: any): any[] {
  if (Array.isArray(json)) return json;
  if (json && typeof json === "object") {
    if (Array.isArray((json as any).items)) return (json as any).items;
    if (Array.isArray((json as any).Items)) return (json as any).Items;
    if (Array.isArray((json as any).data)) return (json as any).data;
  }
  return [];
}

function extractFirstItemFromAny(json: any): any | null {
  if (!json) return null;
  if (Array.isArray(json)) return json[0] ?? null;

  if (json && typeof json === "object") {
    // list が単体で返る場合
    if ((json as any).id) return json;

    const items = extractItemsArrayFromAny(json);
    return items[0] ?? null;
  }

  return null;
}

/**
 * ===========
 * API
 * ===========
 */

/**
 * ✅ Create list
 * POST /lists
 *
 * 1) prices: Array<{modelId, price}> で送る
 * 2) 400 invalid json のときだけ prices: map にして1回だけリトライ
 */
export async function createListHTTP(input: CreateListInput): Promise<ListDTO> {
  const u = auth.currentUser;
  const uid = s(u?.uid);
  const email = s((u as any)?.email);

  const payloadArray = buildCreateListPayloadArray(input);

  // eslint-disable-next-line no-console
  console.log("[list/listRepositoryHTTP] createListHTTP (input)", {
    uid,
    email,
    titleLen: String(input?.title ?? "").length,
    descriptionLen: String(input?.description ?? "").length,
    priceRowsCount: Array.isArray(input?.priceRows) ? input.priceRows.length : 0,
  });

  // eslint-disable-next-line no-console
  console.log("[list/listRepositoryHTTP] createListHTTP (payload:Array)", {
    payload: payloadArray,
    pricesType: "array",
  });

  try {
    return await requestJSON<ListDTO>({
      method: "POST",
      path: "/lists",
      body: payloadArray,
    });
  } catch (e) {
    const msg = String(e instanceof Error ? e.message : e);

    // 400で返ってくる "invalid json" は「構造がDTOと合ってない」可能性が高いので map で再試行
    if (msg === "invalid json") {
      const payloadMap = buildCreateListPayloadMap(input);

      // eslint-disable-next-line no-console
      console.warn("[list/listRepositoryHTTP] retry with prices map payload", {
        pricesType: "map",
        payload: payloadMap,
      });

      return await requestJSON<ListDTO>({
        method: "POST",
        path: "/lists",
        body: payloadMap,
      });
    }

    throw e;
  }
}

/**
 * ✅ List lists
 * GET /lists
 */
export async function fetchListsHTTP(): Promise<ListDTO[]> {
  const json = await requestJSON<any>({
    method: "GET",
    path: "/lists",
  });

  const items = extractItemsArrayFromAny(json);

  // eslint-disable-next-line no-console
  console.log("[list/listRepositoryHTTP] fetchListsHTTP extracted", {
    count: items.length,
    sample: items.slice(0, 3),
  });

  return items as ListDTO[];
}

/**
 * GET /lists/{id}
 */
export async function fetchListByIdHTTP(listId: string): Promise<ListDTO> {
  const id = String(listId ?? "").trim();
  if (!id) {
    // eslint-disable-next-line no-console
    console.error("[list/listRepositoryHTTP] invalid_list_id (empty)", { listId });
    throw new Error("invalid_list_id");
  }

  return await requestJSON<ListDTO>({
    method: "GET",
    path: `/lists/${encodeURIComponent(id)}`,
  });
}

/**
 * ✅ ListDetail 用（hook から移譲）
 * - 1) GET /lists/{id}
 * - 2) fallback: GET /lists?inventoryId=xxx（環境差分吸収）
 */
export async function fetchListDetailHTTP(args: {
  listId: string;
  inventoryIdHint?: string;
}): Promise<ListDTO> {
  const listId = String(args.listId ?? "").trim();
  if (!listId) {
    // eslint-disable-next-line no-console
    console.error("[list/listRepositoryHTTP] invalid_list_id (empty)", { args });
    throw new Error("invalid_list_id");
  }

  try {
    return await fetchListByIdHTTP(listId);
  } catch (e1) {
    const msg1 = String(e1 instanceof Error ? e1.message : e1);

    // eslint-disable-next-line no-console
    console.warn("[list/listRepositoryHTTP] fetchListByIdHTTP failed -> fallback", {
      listId,
      inventoryIdHint: s(args.inventoryIdHint),
      message: msg1,
      raw: e1,
    });

    // fallback inventoryId は hint を優先、無ければ listId を使う（後方互換）
    const inv = s(args.inventoryIdHint) || listId;

    // クエリAPIが無い環境もありえるため、fallback も失敗したら e1 を投げる
    try {
      const json = await requestJSON<any>({
        method: "GET",
        path: `/lists?inventoryId=${encodeURIComponent(inv)}`,
      });

      const first = extractFirstItemFromAny(json);
      if (!first) {
        throw new Error("not_found");
      }
      return first as ListDTO;
    } catch (e2) {
      // eslint-disable-next-line no-console
      console.warn("[list/listRepositoryHTTP] fallback /lists?inventoryId failed", {
        listId,
        inventoryId: inv,
        message: String(e2 instanceof Error ? e2.message : e2),
        raw: e2,
      });

      throw e1;
    }
  }
}

/**
 * GET /lists/{id}/aggregate
 */
export async function fetchListAggregateHTTP(listId: string): Promise<ListAggregateDTO> {
  const id = String(listId ?? "").trim();
  if (!id) {
    // eslint-disable-next-line no-console
    console.error("[list/listRepositoryHTTP] invalid_list_id (empty)", { listId });
    throw new Error("invalid_list_id");
  }

  return await requestJSON<ListAggregateDTO>({
    method: "GET",
    path: `/lists/${encodeURIComponent(id)}/aggregate`,
  });
}

/**
 * GET /lists/{id}/images
 */
export async function fetchListImagesHTTP(listId: string): Promise<ListImageDTO[]> {
  const id = String(listId ?? "").trim();
  if (!id) {
    // eslint-disable-next-line no-console
    console.error("[list/listRepositoryHTTP] invalid_list_id (empty)", { listId });
    throw new Error("invalid_list_id");
  }

  return await requestJSON<ListImageDTO[]>({
    method: "GET",
    path: `/lists/${encodeURIComponent(id)}/images`,
  });
}

/**
 * POST /lists/{id}/images
 * - GCS objectPath を登録する（アップロード自体は別途）
 */
export async function saveListImageFromGCSHTTP(args: {
  listId: string;
  id: string; // ListImage.ID
  fileName?: string;
  bucket?: string; // optional
  objectPath: string;
  size: number; // bytes
  displayOrder: number;
  createdBy?: string;
  createdAt?: string; // RFC3339 optional
}): Promise<ListImageDTO> {
  const listId = String(args.listId ?? "").trim();
  if (!listId) {
    // eslint-disable-next-line no-console
    console.error("[list/listRepositoryHTTP] invalid_list_id (empty)", { args });
    throw new Error("invalid_list_id");
  }

  const payload = {
    id: String(args.id ?? "").trim(),
    fileName: String(args.fileName ?? "").trim(),
    bucket: String(args.bucket ?? "").trim(),
    objectPath: String(args.objectPath ?? "").trim(),
    size: Number(args.size ?? 0),
    displayOrder: Number(args.displayOrder ?? 0),
    createdBy: String(args.createdBy ?? "").trim(),
    createdAt: args.createdAt ? String(args.createdAt).trim() : undefined,
  };

  // eslint-disable-next-line no-console
  console.log("[list/listRepositoryHTTP] saveListImageFromGCSHTTP payload", payload);

  return await requestJSON<ListImageDTO>({
    method: "POST",
    path: `/lists/${encodeURIComponent(listId)}/images`,
    body: payload,
  });
}

/**
 * PUT|POST|PATCH /lists/{id}/primary-image
 */
export async function setListPrimaryImageHTTP(args: {
  listId: string;
  imageId: string;
  updatedBy?: string;
  now?: string; // RFC3339 optional
}): Promise<ListDTO> {
  const listId = String(args.listId ?? "").trim();
  if (!listId) {
    // eslint-disable-next-line no-console
    console.error("[list/listRepositoryHTTP] invalid_list_id (empty)", { args });
    throw new Error("invalid_list_id");
  }

  const payload = {
    imageId: String(args.imageId ?? "").trim(),
    updatedBy: args.updatedBy ? String(args.updatedBy).trim() : undefined,
    now: args.now ? String(args.now).trim() : undefined,
  };

  // eslint-disable-next-line no-console
  console.log("[list/listRepositoryHTTP] setListPrimaryImageHTTP payload", payload);

  return await requestJSON<ListDTO>({
    method: "PUT",
    path: `/lists/${encodeURIComponent(listId)}/primary-image`,
    body: payload,
  });
}
