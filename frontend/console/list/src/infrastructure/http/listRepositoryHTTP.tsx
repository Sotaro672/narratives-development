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

/**
 * ✅ NEW: Update 用
 * - listDetail 側の PriceRow は id = modelId なので、row.id も受ける
 * - backend に送るのは modelId + price だけ（DisallowUnknownFields 対策）
 */
export type UpdateListInput = {
  listId: string;

  title?: string;
  description?: string;

  // detail 側の priceRows（id=modelId）
  priceRows?: Array<{
    // create系: modelId
    modelId?: string;

    // detail系: id (= modelId)
    id?: string;

    price: number | null;

    // UI 用（backend には送らない）
    size?: string;
    color?: string;
    stock?: number;
    rgb?: number | null;
  }>;

  // UI の "list" | "hold" を backend の status に変換して送る（必要な場合のみ）
  decision?: "list" | "hold";

  assigneeId?: string;

  // バックエンドで auth から取るなら省略可
  updatedBy?: string;
};

export type ListDTO = Record<string, any>;
export type ListAggregateDTO = Record<string, any>;
export type ListImageDTO = Record<string, any>;

/**
 * ✅ ListDetail DTO（型ガイド用）
 * - ListDTO は Record<string, any> なので createdByName 等は “そのまま受け取れる”
 * - ただし UI 側で見落としやすいのでここで明示しておく
 */
export type ListDetailDTO = ListDTO & {
  createdByName?: string;
  updatedByName?: string;

  createdBy?: string;
  updatedBy?: string;

  createdAt?: string;
  updatedAt?: string;
};

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

  return rows.map((r) => {
    const modelId = s((r as any)?.modelId);
    const priceMaybe = toNumberOrNull((r as any)?.price);

    if (!modelId) {
      throw new Error("missing_modelId_in_priceRows");
    }

    if (priceMaybe === null) {
      throw new Error("missing_price_in_priceRows");
    }

    return { modelId, price: priceMaybe };
  });
}

/**
 * ✅ update 用: modelId を row.modelId または row.id から取得する
 */
function normalizePricesForBackendUpdate(
  rows: UpdateListInput["priceRows"],
): Array<{ modelId: string; price: number }> {
  if (!Array.isArray(rows)) return [];

  return rows.map((r, idx) => {
    const modelId = s((r as any)?.modelId) || s((r as any)?.id);
    const priceMaybe = toNumberOrNull((r as any)?.price);

    if (!modelId) {
      // update時も modelId が無いと更新できない
      throw new Error(`missing_modelId_in_priceRows_at_${idx}`);
    }
    if (priceMaybe === null) {
      throw new Error(`missing_price_in_priceRows_at_${idx}`);
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
    throw new Error("missing_id");
  }

  const title = s(input?.title);
  if (!title) {
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
 * ✅ NEW: Update payload（最小）
 * - unknown fields を送らない
 * - prices は Array<{modelId, price}> のみ
 * - decision は backend が status を受ける場合のみ送る（listing/hold）
 */
function buildUpdateListPayloadArray(input: UpdateListInput): Record<string, any> {
  const u = auth.currentUser;
  const uid = s(u?.uid);

  const title = s(input?.title);
  const description =
    input?.description === undefined ? undefined : String(input?.description ?? "");

  const prices = normalizePricesForBackendUpdate(input?.priceRows);

  // decision -> status (backend の status が "listing"/"hold" を想定するため)
  let status: string | undefined = undefined;
  if (input?.decision === "list") status = "listing";
  if (input?.decision === "hold") status = "hold";

  const payload: Record<string, any> = {
    // id は path で渡すので body に必須ではない（ただし backend 実装次第で必要なら入れてもOK）
    title: title || undefined,
    description,
    assigneeId: s(input?.assigneeId) || undefined,

    // ✅ backendへ送るのは modelId + price のみ
    prices,

    // ✅ backend が status 更新を受ける場合のみ
    status,

    // 絶対に送らない（名揺れ吸収しない）
    decision: undefined,

    updatedBy: s(input?.updatedBy) || uid || undefined,
  };

  // undefined を落とす
  for (const k of Object.keys(payload)) {
    if (payload[k] === undefined) delete payload[k];
  }

  return payload;
}

/**
 * ✅ fallback: prices を map で送る版
 * backend が `map[string]number` を期待している場合に通る
 */
function buildCreateListPayloadMap(input: CreateListInput): Record<string, any> {
  const base = buildCreateListPayloadArray(input);
  const pricesArray = Array.isArray((base as any).prices)
    ? ((base as any).prices as any[])
    : [];

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

/**
 * ✅ NEW: update fallback: prices を map で送る版
 */
function buildUpdateListPayloadMap(input: UpdateListInput): Record<string, any> {
  const base = buildUpdateListPayloadArray(input);
  const pricesArray = Array.isArray((base as any).prices)
    ? ((base as any).prices as any[])
    : [];

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

  // ✅ debug log
  debug?: {
    tag: string;
    url: string;
    method: string;
    body?: unknown;
  };
}): Promise<T> {
  const token = await getIdToken();
  const url = `${API_BASE}${args.path.startsWith("/") ? "" : "/"}${args.path}`;

  // ✅ debug: request payload
  if (args.debug) {
    try {
      // NOTE: ここで stringify すると "実際に送る JSON" が見える
      const bodyStr =
        args.debug.body === undefined ? undefined : JSON.stringify(args.debug.body);
      console.log(`[list/listRepositoryHTTP] ${args.debug.tag}`, {
        method: args.debug.method,
        url: args.debug.url,
        body: args.debug.body,
        bodyJSON: bodyStr,
      });
    } catch (e) {
      console.log(`[list/listRepositoryHTTP] ${args.debug.tag} (stringify_failed)`, {
        method: args.debug.method,
        url: args.debug.url,
        body: args.debug.body,
        err: String(e),
      });
    }
  }

  let bodyText: string | undefined = undefined;
  if (args.body !== undefined) {
    try {
      bodyText = JSON.stringify(args.body);
    } catch {
      throw new Error("invalid_json_stringify");
    }
  }

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

  if (!res.ok) {
    const msg =
      (json && typeof json === "object" && (json.error || json.message)) ||
      `http_error_${res.status}`;

    // ✅ debug: response error
    console.log(`[list/listRepositoryHTTP] response error`, {
      method: args.method,
      url,
      status: res.status,
      raw: text,
      json,
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
  const payloadArray = buildCreateListPayloadArray(input);

  // ✅ debug: create payload
  console.log("[list/listRepositoryHTTP] createListHTTP payload", payloadArray);

  try {
    return await requestJSON<ListDTO>({
      method: "POST",
      path: "/lists",
      body: payloadArray,
      debug: {
        tag: "POST /lists",
        url: `${API_BASE}/lists`,
        method: "POST",
        body: payloadArray,
      },
    });
  } catch (e) {
    const msg = String(e instanceof Error ? e.message : e);

    if (msg === "invalid json") {
      const payloadMap = buildCreateListPayloadMap(input);

      console.log("[list/listRepositoryHTTP] createListHTTP retry payload(map)", payloadMap);

      return await requestJSON<ListDTO>({
        method: "POST",
        path: "/lists",
        body: payloadMap,
        debug: {
          tag: "POST /lists (retry map)",
          url: `${API_BASE}/lists`,
          method: "POST",
          body: payloadMap,
        },
      });
    }

    throw e;
  }
}

/**
 * ✅ Update list
 * PUT /lists/{id}
 *
 * 1) prices: Array<{modelId, price}> で送る
 * 2) 400 invalid json のときだけ prices: map にして1回だけリトライ
 */
export async function updateListByIdHTTP(input: UpdateListInput): Promise<ListDTO> {
  const listId = s(input?.listId);
  if (!listId) throw new Error("invalid_list_id");

  const payloadArray = buildUpdateListPayloadArray(input);

  // ✅ debug: update payload
  console.log("[list/listRepositoryHTTP] updateListByIdHTTP payload", {
    listId,
    payload: payloadArray,
  });

  try {
    return await requestJSON<ListDTO>({
      method: "PUT",
      path: `/lists/${encodeURIComponent(listId)}`,
      body: payloadArray,
      debug: {
        tag: `PUT /lists/${listId}`,
        url: `${API_BASE}/lists/${encodeURIComponent(listId)}`,
        method: "PUT",
        body: payloadArray,
      },
    });
  } catch (e) {
    const msg = String(e instanceof Error ? e.message : e);

    if (msg === "invalid json") {
      const payloadMap = buildUpdateListPayloadMap(input);

      console.log("[list/listRepositoryHTTP] updateListByIdHTTP retry payload(map)", {
        listId,
        payload: payloadMap,
      });

      return await requestJSON<ListDTO>({
        method: "PUT",
        path: `/lists/${encodeURIComponent(listId)}`,
        body: payloadMap,
        debug: {
          tag: `PUT /lists/${listId} (retry map)`,
          url: `${API_BASE}/lists/${encodeURIComponent(listId)}`,
          method: "PUT",
          body: payloadMap,
        },
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
  return items as ListDTO[];
}

/**
 * GET /lists/{id}
 *
 * ✅ ListDetail で使うので、レスポンスに createdByName が入っているか確認できるログを追加
 */
export async function fetchListByIdHTTP(listId: string): Promise<ListDTO> {
  const id = String(listId ?? "").trim();
  if (!id) {
    throw new Error("invalid_list_id");
  }

  const dto = await requestJSON<ListDTO>({
    method: "GET",
    path: `/lists/${encodeURIComponent(id)}`,
    debug: {
      tag: `GET /lists/${id}`,
      url: `${API_BASE}/lists/${encodeURIComponent(id)}`,
      method: "GET",
    },
  });

  // ✅ ListDetail 取得内容が分かるログ（createdByName もチェック）
  try {
    const anyDto = dto as any;
    console.log("[list/listRepositoryHTTP] fetchListByIdHTTP ok", {
      listId: id,
      hasCreatedByName: Boolean(s(anyDto?.createdByName)),
      createdBy: s(anyDto?.createdBy),
      createdByName: s(anyDto?.createdByName),
      updatedBy: s(anyDto?.updatedBy),
      updatedByName: s(anyDto?.updatedByName),
      createdAt: s(anyDto?.createdAt),
      updatedAt: s(anyDto?.updatedAt),
      keys: anyDto && typeof anyDto === "object" ? Object.keys(anyDto) : [],
      dto,
    });
  } catch (e) {
    console.log("[list/listRepositoryHTTP] fetchListByIdHTTP ok (log_failed)", {
      listId: id,
      err: String(e),
    });
  }

  return dto;
}

/**
 * ✅ ListDetail 用（hook から移譲）
 * - 1) GET /lists/{id}
 * - 2) fallback: GET /lists?inventoryId=xxx（環境差分吸収）
 *
 * ✅ 「ListDetail画面を開いた際に取得したデータ」が分かるログをここにも追加
 */
export async function fetchListDetailHTTP(args: {
  listId: string;
  inventoryIdHint?: string;
}): Promise<ListDTO> {
  const listId = String(args.listId ?? "").trim();
  if (!listId) {
    throw new Error("invalid_list_id");
  }

  // ✅ ListDetail open log
  console.log("[list/listRepositoryHTTP] fetchListDetailHTTP start", {
    listId,
    inventoryIdHint: s(args.inventoryIdHint),
    url: `${API_BASE}/lists/${encodeURIComponent(listId)}`,
  });

  try {
    const dto = await fetchListByIdHTTP(listId);

    // ✅ ListDetail resolved log (primary)
    console.log("[list/listRepositoryHTTP] fetchListDetailHTTP resolved", {
      source: "GET /lists/{id}",
      listId,
      createdByName: s((dto as any)?.createdByName),
      updatedByName: s((dto as any)?.updatedByName),
      dto,
    });

    return dto;
  } catch (e1) {
    const inv = s(args.inventoryIdHint) || listId;

    console.log("[list/listRepositoryHTTP] fetchListDetailHTTP fallback start", {
      listId,
      inventoryId: inv,
      url: `${API_BASE}/lists?inventoryId=${encodeURIComponent(inv)}`,
      err: String(e1 instanceof Error ? e1.message : e1),
    });

    try {
      const json = await requestJSON<any>({
        method: "GET",
        path: `/lists?inventoryId=${encodeURIComponent(inv)}`,
        debug: {
          tag: `GET /lists?inventoryId=${inv}`,
          url: `${API_BASE}/lists?inventoryId=${encodeURIComponent(inv)}`,
          method: "GET",
        },
      });

      const first = extractFirstItemFromAny(json);
      if (!first) throw new Error("not_found");

      // ✅ ListDetail resolved log (fallback)
      console.log("[list/listRepositoryHTTP] fetchListDetailHTTP resolved", {
        source: "GET /lists?inventoryId=xxx",
        listId,
        inventoryId: inv,
        createdByName: s((first as any)?.createdByName),
        updatedByName: s((first as any)?.updatedByName),
        dto: first,
        raw: json,
      });

      return first as ListDTO;
    } catch (e2) {
      console.log("[list/listRepositoryHTTP] fetchListDetailHTTP fallback failed", {
        listId,
        inventoryId: inv,
        err: String(e2 instanceof Error ? e2.message : e2),
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
  if (!id) throw new Error("invalid_list_id");

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
  if (!id) throw new Error("invalid_list_id");

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
  if (!listId) throw new Error("invalid_list_id");

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
  if (!listId) throw new Error("invalid_list_id");

  const payload = {
    imageId: String(args.imageId ?? "").trim(),
    updatedBy: args.updatedBy ? String(args.updatedBy).trim() : undefined,
    now: args.now ? String(args.now).trim() : undefined,
  };

  return await requestJSON<ListDTO>({
    method: "PUT",
    path: `/lists/${encodeURIComponent(listId)}/primary-image`,
    body: payload,
  });
}
