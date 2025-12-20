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
  // ルート（inventory/list/create から作成する想定）
  inventoryId?: string;

  // UI 入力
  title: string;
  description: string;

  // PriceCard の rows（必要に応じて拡張）
  // 例: [{ size:"S", color:"Red", stock:10, price:1000, rgb:123 }]
  priceRows?: Array<{
    size: string;
    color: string;
    stock: number;
    price: number | null;
    rgb?: number | null;
  }>;

  // 画面の「出品｜保留」
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
 * priceRows の 1行に "inventoryId" を付与して backend が拾える形にする。
 * - 現状の UI 行には id がないため、size+color を安定キーとして使う（空なら index）
 * - backend 側（list_handler.go）の extractPrices が
 *   priceRows[].inventoryId と priceRows[].price を読む想定
 */
function normalizePriceRowsForBackend(
  rows: CreateListInput["priceRows"],
): Array<{
  inventoryId: string; // ✅ backend が拾う
  price: number | null; // ✅ backend が拾う
  // 以降はデバッグ/将来拡張用（backend が無視してもOK）
  size: string;
  color: string;
  stock: number;
  rgb?: number | null;
}> {
  if (!Array.isArray(rows)) return [];
  return rows.map((r, i) => {
    const size = s((r as any)?.size);
    const color = s((r as any)?.color);
    const stock = Number((r as any)?.stock ?? 0);
    const price = toNumberOrNull((r as any)?.price);
    const rgb = (r as any)?.rgb ?? null;

    const keyBase = `${size}__${color}`.trim();
    const inventoryId = keyBase !== "__" && keyBase !== "" ? keyBase : `row_${i}`;

    return {
      inventoryId,
      price,
      size,
      color,
      stock: Number.isFinite(stock) ? stock : 0,
      rgb: rgb === null || rgb === undefined ? null : toNumberOrNull(rgb),
    };
  });
}

/**
 * CreateList の最終 payload を組み立てる
 * - createdBy を currentUser.uid で補完（未指定なら）
 * - priceRows を backend が拾える shape に正規化（inventoryId / price）
 */
function buildCreateListPayload(input: CreateListInput): Record<string, any> {
  const u = auth.currentUser;
  const uid = s(u?.uid);

  const priceRowsNormalized = normalizePriceRowsForBackend(input?.priceRows);

  const payload = {
    inventoryId: s(input?.inventoryId),
    title: s(input?.title),
    description: String(input?.description ?? ""), // description は空文字を許容したいので trim しない（UIの意図尊重）
    decision: input?.decision,
    assigneeId: s(input?.assigneeId) || undefined,

    // ✅ 重要: createdBy を currentMember(uid) に寄せる
    createdBy: s(input?.createdBy) || uid || "system",

    // ✅ 重要: backend が拾える priceRows にする
    priceRows: priceRowsNormalized,
  };

  return payload;
}

async function getIdToken(): Promise<string> {
  const u = auth.currentUser;
  if (!u) {
    throw new Error("not_authenticated");
  }
  return await u.getIdToken();
}

async function requestJSON<T>(args: {
  method: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  path: string;
  body?: unknown;
}): Promise<T> {
  const token = await getIdToken();
  const url = `${API_BASE}${args.path.startsWith("/") ? "" : "/"}${args.path}`;

  // ✅ backend に渡す payload が “分かる” ログ（JSON 文字列も出す / 長い場合は truncate）
  let bodyPreview: any = args.body;
  let bodyJSON = "";
  try {
    bodyJSON = args.body === undefined ? "" : JSON.stringify(args.body);
  } catch {
    bodyJSON = "<json_stringify_failed>";
  }
  const bodyJSONPreview =
    bodyJSON.length > 3000 ? bodyJSON.slice(0, 3000) + "...(truncated)" : bodyJSON;

  // eslint-disable-next-line no-console
  console.log("[list/listRepositoryHTTP] request", {
    method: args.method,
    url,
    body: bodyPreview,
    bodyJsonLen: bodyJSON.length,
    bodyJsonPreview: bodyJSONPreview,
  });

  const res = await fetch(url, {
    method: args.method,
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
    body: args.body === undefined ? undefined : JSON.stringify(args.body),
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

/**
 * ===========
 * API
 * ===========
 *
 * Backend:
 * - POST /lists            (create)
 * - GET  /lists            (list)  ✅ NEW: 一覧取得
 * - GET  /lists/{id}       (detail)
 * - GET  /lists/{id}/aggregate
 * - GET  /lists/{id}/images
 * - POST /lists/{id}/images
 * - PUT|POST|PATCH /lists/{id}/primary-image
 */

/**
 * ✅ Create list
 * POST /lists
 */
export async function createListHTTP(input: CreateListInput): Promise<ListDTO> {
  const u = auth.currentUser;
  const uid = s(u?.uid);
  const email = s((u as any)?.email);

  // ✅ 修正: backend が拾える payload に変換（createdBy / priceRows）
  const payload = buildCreateListPayload(input);

  // eslint-disable-next-line no-console
  console.log("[list/listRepositoryHTTP] createListHTTP (input)", {
    uid,
    email,
    input,
    titleLen: String(input?.title ?? "").length,
    descriptionLen: String(input?.description ?? "").length,
    priceRowsCount: Array.isArray(input?.priceRows) ? input.priceRows.length : 0,
  });

  // eslint-disable-next-line no-console
  console.log("[list/listRepositoryHTTP] createListHTTP (payload to backend)", {
    payload,
    createdByResolved: payload.createdBy,
    priceRowsCount: Array.isArray((payload as any)?.priceRows)
      ? (payload as any).priceRows.length
      : 0,
    priceRowsSample:
      Array.isArray((payload as any)?.priceRows) && (payload as any).priceRows.length > 0
        ? (payload as any).priceRows.slice(0, 5)
        : [],
  });

  return await requestJSON<ListDTO>({
    method: "POST",
    path: "/lists",
    body: payload,
  });
}

/**
 * ✅ List lists
 * GET /lists
 * - レスポンス形が揺れても配列として返す（service 側が HTTP 差分を気にしないで済むように）
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
  if (!id) throw new Error("invalid_list_id");

  return await requestJSON<ListDTO>({
    method: "GET",
    path: `/lists/${encodeURIComponent(id)}`,
  });
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
  if (!listId) throw new Error("invalid_list_id");

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
